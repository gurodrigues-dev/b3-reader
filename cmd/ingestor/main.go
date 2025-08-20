package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/gurodrigues-dev/b3-reader/config"
	"github.com/gurodrigues-dev/b3-reader/internal/reader"
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

	migrations(cfg.DatabaseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		log.Fatal(err)
	}

	csvReader := reader.NewCSVReader(cfg.FilePath, ';', -1, l)
	repository := storage.NewTradeRepository(pool)
	service := trade.NewService(repository, csvReader, l)

	l.Info("data ingestion started")
	err = service.IngestFiles(ctx, cfg.FilePath)
	if err != nil {
		log.Fatal(err)
	}
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
