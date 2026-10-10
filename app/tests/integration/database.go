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
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	DBName     = "test_db"
	DBUser     = "test_user"
	DBPassword = "test_password"
)

type TestDatabase struct {
	DBInstance *pgxpool.Pool
	DBURL      string
	container  *postgres.PostgresContainer
}

func SetupTestDatabase() *TestDatabase {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		60*time.Second,
	)
	defer cancel()

	container, dbInstance, dbURL, err := createContainer(ctx)
	if err != nil {
		log.Fatal("failed to setup test: ", err)
	}

	if err := migrateDB(ctx, dbURL); err != nil {
		_ = testcontainers.TerminateContainer(container)

		log.Fatal("failed to perform db migration: ", err)
	}

	return &TestDatabase{
		DBInstance: dbInstance,
		DBURL:      dbURL,
		container:  container,
	}
}

func (tdb *TestDatabase) TearDown() {
	if tdb.DBInstance != nil {
		tdb.DBInstance.Close()
	}

	if tdb.container != nil {
		if err := testcontainers.TerminateContainer(tdb.container); err != nil {
			log.Printf("failed to terminate postgres container: %v", err)
		}
	}
}

func createContainer(
	ctx context.Context,
) (
	*postgres.PostgresContainer,
	*pgxpool.Pool,
	string,
	error,
) {
	container, err := postgres.Run(
		ctx,
		"postgres:14-alpine",

		postgres.WithDatabase(DBName),
		postgres.WithUsername(DBUser),
		postgres.WithPassword(DBPassword),

		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, nil, "", fmt.Errorf(
			"failed to start postgres container: %w",
			err,
		)
	}

	dbURL, err := container.ConnectionString(
		ctx,
		"sslmode=disable",
	)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)

		return nil, nil, "", fmt.Errorf(
			"failed to get postgres connection string: %w",
			err,
		)
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)

		return nil, nil, "", fmt.Errorf(
			"failed to create database pool: %w",
			err,
		)
	}

	if err := db.Ping(ctx); err != nil {
		db.Close()
		_ = testcontainers.TerminateContainer(container)

		return nil, nil, "", fmt.Errorf(
			"failed to connect to database: %w",
			err,
		)
	}

	return container, db, dbURL, nil
}

func migrateDB(
	ctx context.Context,
	dbURL string,
) error {
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return fmt.Errorf(
			"open migration database: %w",
			err,
		)
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf(
			"ping migration database: %w",
			err,
		)
	}

	goose.SetBaseFS(schema.SQL)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf(
			"set goose dialect: %w",
			err,
		)
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf(
			"run goose migrations: %w",
			err,
		)
	}

	log.Println("migration done")

	return nil
}
