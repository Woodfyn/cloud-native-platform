package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/Woodfyn/cloud-native-platform/schema"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	DBName     = "test_db"
	DBUser     = "test_user"
	DBPassword = "test_password"
)

func psqlURL(address string) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?sslmode=disable",
		DBUser,
		DBPassword,
		address,
		DBName,
	)
}

type TestDatabase struct {
	DBInstance *pgxpool.Pool
	DBAddress  string
	container  testcontainers.Container
}

func SetupTestDatabase() *TestDatabase {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	container, dbInstance, dbAddr, err := createContainer(ctx)
	if err != nil {
		log.Fatal("failed to setup test: ", err)
	}

	if err := migrateDB(ctx, dbAddr); err != nil {
		log.Fatal("failed to perform db migration: ", err)
	}

	return &TestDatabase{
		container:  container,
		DBInstance: dbInstance,
		DBAddress:  dbAddr,
	}
}

func (tdb *TestDatabase) TearDown() {
	tdb.DBInstance.Close()
	_ = tdb.container.Terminate(context.Background())
}

func createContainer(ctx context.Context) (
	testcontainers.Container,
	*pgxpool.Pool,
	string,
	error,
) {
	env := map[string]string{
		"POSTGRES_PASSWORD": DBPassword,
		"POSTGRES_USER":     DBUser,
		"POSTGRES_DB":       DBName,
	}

	req := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:14-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env:          env,
			WaitingFor: wait.ForLog(
				"database system is ready to accept connections",
			),
		},
		Started: true,
	}

	container, err := testcontainers.GenericContainer(ctx, req)
	if err != nil {
		return nil, nil, "", fmt.Errorf(
			"failed to start container: %w",
			err,
		)
	}

	p, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return container, nil, "", fmt.Errorf(
			"failed to get container external port: %w",
			err,
		)
	}

	dbAddr := fmt.Sprintf("localhost:%s", p.Port())

	db, err := pgxpool.New(ctx, psqlURL(dbAddr))
	if err != nil {
		return container, nil, dbAddr, fmt.Errorf(
			"failed to create database pool: %w",
			err,
		)
	}

	// pgxpool.New doesn't actually verify the connection.
	if err := db.Ping(ctx); err != nil {
		return container, nil, dbAddr, fmt.Errorf(
			"failed to connect to database: %w",
			err,
		)
	}

	return container, db, dbAddr, nil
}

func migrateDB(
	ctx context.Context,
	address string,
) error {
	db, err := sql.Open("pgx", psqlURL(address))
	if err != nil {
		return fmt.Errorf("open migration database: %w", err)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping migration database: %w", err)
	}

	goose.SetBaseFS(schema.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	// Important: your embed path is schema/*.sql,
	// therefore the goose directory must be "schema".
	if err := goose.UpContext(ctx, db, "schema"); err != nil {
		return fmt.Errorf("run goose migrations: %w", err)
	}

	log.Println("migration done")

	return nil
}
