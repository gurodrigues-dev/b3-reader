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
			err := filepath.Walk(r.path, func(filePath string, info os.FileInfo, err error) error {
				if err != nil {
					return err
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
				case <-ctx.Done():
					return ctx.Err()
				}

				return nil
			})

			if err != nil {
				errChan <- fmt.Errorf("list files in path error: %w", err)
			}
		} else {
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
