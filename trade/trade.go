package trade

import (
	"context"
	"time"
)

type Trade struct {
	ID                  uint
	CodigoInstrumento   string
	HoraFechamento      string
	QuantidadeNegociada int
	PrecoNegocio        float64
	DataNegocio         time.Time
	CreatedAt           time.Time
}

type Repository interface {
	SaveBatch(ctx context.Context, trades []Trade) (int64, error)
	GetAggregatedData(ctx context.Context, ticker string, startDate time.Time) (float64, int, error)
}

type Usecase interface {
	IngestFiles(ctx context.Context, filePath string) error
	GetAggregatedData(ctx context.Context, ticker string, startDate *time.Time) (map[string]interface{}, error)
}
