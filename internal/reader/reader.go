package reader

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

type Reader interface {
	// Use read to read a folder of CSV files or a single CSV. It sends columns and rows to the expected channel.
	Read(ctx context.Context) (<-chan [][]string, <-chan error)
}

type CSVReader struct {
	path    string
	sep     rune
	records int
	logger  *zap.Logger
}

func NewCSVReader(path string, sep rune, rec int, logger *zap.Logger) *CSVReader {
	return &CSVReader{
		path:    path,
		sep:     sep,
		records: rec,
		logger:  logger,
	}
}

func (r *CSVReader) Read(ctx context.Context) (<-chan [][]string, <-chan error) {
	recordsChan := make(chan [][]string)
	errChan := make(chan error)

	go func() {
		defer close(recordsChan)
		defer close(errChan)

		info, err := os.Stat(r.path)
		if err != nil {
			errChan <- fmt.Errorf("access path error: %w", err)
			return
		}

		if info.IsDir() {
			r.readDir(ctx, recordsChan, errChan)
			return
		}

		r.readSingleFile(ctx, recordsChan, errChan)
	}()

	return recordsChan, errChan
}

func (r *CSVReader) readFile(filePath string) ([][]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file error: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	reader.Comma = r.sep
	reader.FieldsPerRecord = r.records

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("file csv read error: %w", err)
	}

	return records, nil
}

func (r *CSVReader) readDir(ctx context.Context, recordsChan chan<- [][]string, errChan chan<- error) {
	err := filepath.Walk(r.path, func(filePath string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}

		r.logger.Info("reading new file", zap.String("file", info.Name()))
		records, err := r.readFile(filePath)
		if err != nil {
			return fmt.Errorf("read file error %s: %w", filePath, err)
		}

		r.logger.Info("sending columns and rows to channel")
		select {
		case recordsChan <- records:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	if err != nil {
		errChan <- fmt.Errorf("list files in path error: %w", err)
	}
}

func (r *CSVReader) readSingleFile(ctx context.Context, recordsChan chan<- [][]string, errChan chan<- error) {
	r.logger.Info("reading file")
	records, err := r.readFile(r.path)
	if err != nil {
		errChan <- fmt.Errorf("read file error %s: %w", r.path, err)
		return
	}

	r.logger.Info("sending columns and rows to channel")
	select {
	case recordsChan <- records:
	case <-ctx.Done():
		return
	}
}
