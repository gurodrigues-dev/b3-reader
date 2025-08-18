package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gurodrigues-dev/b3-reader/config"
	"github.com/gurodrigues-dev/b3-reader/internal/controllers"
	"github.com/gurodrigues-dev/b3-reader/trade"
	"github.com/gurodrigues-dev/b3-reader/trade/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

func main() {
	l, _ := zap.NewProduction()
	defer l.Sync()

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

	repository := storage.NewTradeRepository(pool)
	service := trade.NewService(repository, nil, l)
	controller := controllers.NewController(service, l)

	router := setupRouter(controller)

	l.Info("starting server", zap.String("port", cfg.ServerPort))
	router.Run(fmt.Sprintf(":%s", cfg.ServerPort))
}

func setupRouter(ctrl *controllers.Controller) *gin.Engine {
	r := gin.Default()
	r.GET("/api/v1/trades", ctrl.GetTrade)
	return r
}
