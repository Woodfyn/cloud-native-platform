package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Woodfyn/cloud-native-platform/controllers/order"
	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	PostgresAddress string
	LogLevel        string
}

func initConfig() *Config {
	get := func(key string) string {
		value := os.Getenv(key)
		if value == "" {
			panic(fmt.Errorf("value %s must be privode", key))
		}

		return value
	}

	return &Config{
		PostgresAddress: get("POSTGRES_ADDRESS"),
		LogLevel:        get("LOG_LEVEL"),
	}
}

const (
	_shutdownTimeout    = 10 * time.Second
	_serverReadTimeout  = 10 * time.Second
	_serverWriteTimeout = 15 * time.Second
	_serverIdleTimeout  = 30 * time.Second
)

func main() {
	rootCtx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	conf := initConfig()
	logger := initLogger(conf.LogLevel)
	psqlConn := initPSQLConn(rootCtx, conf.PostgresAddress)
	orderHandler := order.NewHandler(psqlConn, logger)

	router, err := graceful.Default(
		graceful.WithShutdownTimeout(_shutdownTimeout),
		graceful.WithServerTimeouts(
			_serverReadTimeout,
			_serverWriteTimeout,
			_serverIdleTimeout,
		),
		graceful.WithAfterShutdown(func(_ context.Context) error {
			psqlConn.Close()
			return nil
		}),
	)
	if err != nil {
		panic(err)
	}
	defer router.Close()

	router.GET("/orders", orderHandler.Get)
	router.POST("/orders", orderHandler.Create)

	router.GET("/healthz", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "OK")
	})

	router.GET("/metricsz", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "OK")
	})

	router.GET("/readyz", func(ctx *gin.Context) {
		ctx.String(http.StatusOK, "OK")
	})

	if err = router.RunWithContext(rootCtx); err != nil && errors.Is(err, context.Canceled) {
		panic(err)
	}
}

func initLogger(logLevel string) *slog.Logger {
	slogLevel := slog.LevelInfo
	switch logLevel {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "error":
		slogLevel = slog.LevelError
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slogLevel,
	}))
}

func initPSQLConn(
	ctx context.Context,
	address string,
) *pgxpool.Pool {
	conn, err := pgxpool.New(ctx, address)
	if err != nil {
		panic(err)
	}

	return conn
}
