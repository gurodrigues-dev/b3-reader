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
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

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
	r.GET("/api/v1/trades", ctrl.GetTrade)
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
