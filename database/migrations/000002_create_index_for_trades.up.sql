BEGIN;

CREATE INDEX idx_trades_instrumento_data
    ON trades (codigo_instrumento, data_negocio);

CREATE INDEX idx_trades_data
    ON trades (data_negocio);

COMMIT;
