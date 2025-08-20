package trade

import (
	"context"
	"fmt"
	"time"

	"github.com/gurodrigues-dev/b3-reader/internal/batcher"
	"github.com/gurodrigues-dev/b3-reader/internal/reader"
	"go.uber.org/zap"
)

const (
	batchSize = 5000
)

type Service struct {
	repository Repository
	csvreader  reader.Reader
	logger     *zap.Logger
}

func NewService(r Repository, csv reader.Reader, l *zap.Logger) *Service {
	return &Service{
		repository: r,
		csvreader:  csv,
		logger:     l,
	}
}

func (s *Service) IngestFiles(ctx context.Context, filePath string) error {
	s.logger.Info("ingesting files...")
	recordsChan, errChan := s.csvreader.Read(ctx)

	for {
		select {
		case records, ok := <-recordsChan:
			if !ok {
				return nil
			}
			if err := s.processRecords(ctx, filePath, records); err != nil {
				return err
			}

		case err, ok := <-errChan:
			if ok {
				return fmt.Errorf("file read error: %w", err)
			}

		case <-ctx.Done():
			s.logger.Info("context canceled")
			return ctx.Err()
		}
	}
}

func (s *Service) GetAggregatedData(ctx context.Context, ticker string, startDate *time.Time) (*AggregatedData, error) {
	if ticker == "" {
		return nil, fmt.Errorf("ticker is required")
	}

	if startDate == nil {
		defaultStartDate := time.Now().AddDate(0, 0, -7)
		startDate = &defaultStartDate
	}

	maxRangeValue, maxDailyVolume, err := s.repository.GetAggregatedData(ctx, ticker, *startDate)
	if err != nil {
		return nil, fmt.Errorf("fetching aggregated data error: %w", err)
	}

	return &AggregatedData{
		Ticker:         ticker,
		MaxDailyVolume: maxDailyVolume,
		MaxRangeValue:  maxRangeValue,
	}, nil
}

func (s *Service) processRecords(ctx context.Context, filePath string, records [][]string) error {
	if len(records) > 0 {
		records = records[1:]
	}

	s.logger.Info("making parse records")
	trades, err := parseTrade(records)
	if err != nil {
		return fmt.Errorf("parse error in file %s: %w", filePath, err)
	}

	s.logger.Info("creating batches")
	batches, err := batcher.Batch(trades, batchSize)
	if err != nil {
		return fmt.Errorf("batch error: %w", err)
	}

	s.logger.Info("inserting batch in db")
	for idx, batch := range batches {
		if _, err := s.repository.SaveBatch(ctx, batch); err != nil {
			return fmt.Errorf("database save batch error %d: %w", idx+1, err)
		}
	}

	return nil
}
