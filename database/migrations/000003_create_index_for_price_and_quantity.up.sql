BEGIN;

CREATE INDEX idx_trades_ticker_date_preco ON trades (codigo_instrumento, data_negocio, preco_negocio);
CREATE INDEX idx_trades_ticker_date_qtd ON trades (codigo_instrumento, data_negocio, quantidade_negociada);

COMMIT;
