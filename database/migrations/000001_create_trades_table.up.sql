BEGIN;

CREATE TABLE trades (
    id SERIAL PRIMARY KEY,
    data_negocio DATE NOT NULL,
    codigo_instrumento VARCHAR(50) NOT NULL,
    preco_negocio NUMERIC(10, 2) NOT NULL,
    quantidade_negociada INT NOT NULL,
    hora_fechamento TIME NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

COMMIT;
