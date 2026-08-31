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

	"github.com/gin-contrib/graceful"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type config struct {
	PostgresAddress string
	LogLevel        string
}

func initConfig() *config {
	get := func(key string) string {
		if value := os.Getenv(key); value == "" {
			panic(fmt.Errorf("Value %s must be privode", key))
		} else {
			return value
		}
	}

	return &config{
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
	ctrl := newController(psqlConn, logger)

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

	router.Use(func(ctx *gin.Context) {
		logger.Debug("%s: %s", ctx.Request.Method, ctx.Request.RequestURI)
	})

	router.GET("/orders", ctrl.getOrders)
	router.POST("/orders", ctrl.createOrder)

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

type controller struct {
	psqlConn *pgx.Conn
	log      *slog.Logger
}

func newController(
	psqlConn *pgx.Conn,
	log *slog.Logger,
) *controller {
	return &controller{
		psqlConn: psqlConn,
		log:      log,
	}
}

func (c *controller) getOrders(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, map[string]string{
		"order_1": "Hello!",
		"order_2": "World!",
	})
}

func (c *controller) createOrder(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, map[string]string{
		"order_1": "Hello!",
		"order_2": "World!",
	})
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
