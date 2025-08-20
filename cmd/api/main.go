package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/gurodrigues-dev/b3-reader/config"
	"github.com/gurodrigues-dev/b3-reader/internal/controllers"
	"github.com/gurodrigues-dev/b3-reader/trade"
	"github.com/gurodrigues-dev/b3-reader/trade/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/gurodrigues-dev/b3-reader/docs"
)

// @title B3 Reader API
// @version 1.0
// @description API para leitura e agregação de dados de trades da B3.
// @BasePath /api/v1
func main() {
	l, _ := zap.NewProduction()

	cfg, err := config.LoadEnvs()
	if err != nil {
		log.Fatalf("fail to load envs: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		panic(err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		panic(err)
	}

	migrations(cfg.DatabaseURL)

	repository := storage.NewTradeRepository(pool)
	service := trade.NewService(repository, nil, l)
	controller := controllers.NewController(service, l)

	router := setupRouter(controller)

	l.Info("starting server", zap.String("port", cfg.ServerPort))
	err = router.Run(fmt.Sprintf(":%s", cfg.ServerPort))
	if err != nil {
		panic(err)
	}
}

func setupRouter(ctrl *controllers.Controller) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api/v1")
	api.GET("/trades", ctrl.GetTrade)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	return r
}

func migrations(database string) {
	m, err := migrate.New(
		"file://database/migrations",
		database,
	)
	if err != nil {
		log.Fatalf("migrate error: %v", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Fatalf("upload migrations error: %v", err)
	}

	log.Println("migrations finished.")
}
