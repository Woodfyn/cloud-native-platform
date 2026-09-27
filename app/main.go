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
	"github.com/jackc/pgx/v5"
)

type Config struct {
	PostgresAddress string
	LogLevel        string
}

func initConfig() *Config {
	get := func(key string) string {
		if value := os.Getenv(key); value == "" {
			panic(fmt.Errorf("Value %s must be privode", key))
		} else {
			return value
		}
	}

	return &Config{
		PostgresAddress: get("POSTGRES_ADDRESS"),
		LogLevel:        get("LOG_LEVEL"),
	}
}

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
		graceful.WithShutdownTimeout(10*time.Second),
		graceful.WithServerTimeouts(10*time.Second, 15*time.Second, 30*time.Second),
		graceful.WithAfterShutdown(func(ctx context.Context) error {
			var result error
			return errors.Join(
				result,
				psqlConn.Close(ctx),
			)
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

	if err := router.RunWithContext(rootCtx); err != nil && errors.Is(err, context.Canceled) {
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
	psqlAddress string,
) *pgx.Conn {
	conn, err := pgx.Connect(ctx, psqlAddress)
	if err != nil {
		panic(err)
	}

	return conn
}
