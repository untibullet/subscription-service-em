package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	_ "github.com/untibullet/subscription-service-em/docs"
	"github.com/untibullet/subscription-service-em/internal/config"
	"github.com/untibullet/subscription-service-em/internal/repository"
	"github.com/untibullet/subscription-service-em/internal/service"
	"go.uber.org/zap"
)

// @title Subscription Service API
// @version 1.0
// @description API для управления подписками
// @host localhost:9000
// @BasePath /api/v1
func main() {
	// Конфиг
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Логгер
	var logger *zap.Logger
	if cfg.Env == "production" {
		logger, _ = zap.NewProduction()
	} else {
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	// БД
	pool, err := pgxpool.New(context.Background(), cfg.Database.GetDSN())
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal("failed to ping database", zap.Error(err))
	}
	logger.Info("database connected")

	// Repository
	repo := repository.NewPostgresSubscriptionRepo(pool)

	// Сервис
	httpService := service.NewHTTPService(repo, logger)

	// Echo
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Ручки
	httpService.RegisterRoutes(e)

	// Swagger UI
	e.GET("/swagger/*", echoSwagger.WrapHandler)

	// Старт
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	logger.Info("starting server", zap.String("addr", addr))
	if err := e.Start(addr); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}
}
