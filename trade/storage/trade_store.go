package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/gurodrigues-dev/b3-reader/trade"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TradeRepository struct {
	pool *pgxpool.Pool
}

func NewTradeRepository(pool *pgxpool.Pool) *TradeRepository {
	return &TradeRepository{
		pool: pool,
	}
}

func (r *TradeRepository) SaveBatch(ctx context.Context, trades []trade.Trade) (int64, error) {
	if len(trades) == 0 {
		return 0, nil
	}

	tableName := "trades"
	columns := []string{
		"data_negocio",
		"codigo_instrumento",
		"preco_negocio",
		"quantidade_negociada",
		"hora_fechamento",
		"created_at",
	}

	rows := make([][]interface{}, len(trades))
	for i, t := range trades {
		if t.CreatedAt.IsZero() {
			t.CreatedAt = time.Now()
		}
		rows[i] = []interface{}{
			t.DataNegocio,
			t.CodigoInstrumento,
			t.PrecoNegocio,
			t.QuantidadeNegociada,
			t.HoraFechamento,
			t.CreatedAt,
		}
	}

	conn, err := r.pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("pool error: %w", err)
	}
	defer conn.Release()

	count, err := conn.Conn().CopyFrom(
		ctx,
		pgx.Identifier{tableName},
		columns,
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return 0, fmt.Errorf("sql copy error: %w", err)
	}

	return count, nil
}

func (r *TradeRepository) GetAggregatedData(ctx context.Context, ticker string, startDate time.Time) (float64, int, error) {
	query := `
		SELECT
			MAX(preco_negocio) AS max_range_value,
			MAX(SUM(quantidade_negociada)) OVER() AS max_daily_volume
		FROM trades
		WHERE codigo_instrumento = $1
			AND data_negocio >= $2
		GROUP BY data_negocio;
	`

	var maxRangeValue float64
	var maxDailyVolume int

	err := r.pool.QueryRow(ctx, query, ticker, startDate).Scan(&maxRangeValue, &maxDailyVolume)
	if err != nil {
		return 0, 0, fmt.Errorf("error querying aggregated data: %w", err)
	}

	return maxRangeValue, maxDailyVolume, nil
}
