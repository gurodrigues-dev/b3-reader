BEGIN;

DROP INDEX IF EXISTS idx_trades_ticker_date_preco;
DROP INDEX IF EXISTS idx_trades_ticker_date_qtd;

COMMIT;
