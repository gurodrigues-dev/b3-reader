BEGIN;

CREATE INDEX idx_trades_brin_date ON trades USING brin (data_negocio);

COMMIT;
