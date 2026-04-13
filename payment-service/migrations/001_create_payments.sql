CREATE TABLE IF NOT EXISTS payments (
    id             TEXT   PRIMARY KEY,
    order_id       TEXT   NOT NULL UNIQUE,
    transaction_id TEXT   NOT NULL UNIQUE,
    amount         BIGINT NOT NULL,
    status         TEXT   NOT NULL
);
