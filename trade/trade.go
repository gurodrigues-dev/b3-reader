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

type Writer interface {
	// Insert trades in batches into the database.
	SaveBatch(ctx context.Context, trades []Trade) (int64, error)
}

type Reader interface {
	// Search aggregated data by date and volume of a trade.
	GetAggregatedData(ctx context.Context, ticker string, startDate time.Time) (float64, int, error)
}

type Repository interface {
	Writer
	Reader
}

type Usecase interface {
	// Ingest data into the database based on a csv folder or a single csv file.
	IngestFiles(ctx context.Context, filePath string) error
	// Search for volume and aggregation of a trade, using filters.
	GetAggregatedData(ctx context.Context, ticker string, startDate *time.Time) (map[string]interface{}, error)
}
