CREATE TABLE IF NOT EXISTS customers (
    id               TEXT PRIMARY KEY,
    outstanding_kobo INTEGER NOT NULL DEFAULT 100000000, -- 1,000,000 NGN in kobo
    total_paid_kobo  INTEGER NOT NULL DEFAULT 0,
    created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS payments (
    id                    INTEGER PRIMARY KEY AUTOINCREMENT,
    customer_id           TEXT NOT NULL,
    amount_kobo           INTEGER NOT NULL,
    transaction_reference TEXT NOT NULL,
    transaction_date      DATETIME NOT NULL,
    created_at            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (customer_id) REFERENCES customers(id),
    UNIQUE (transaction_reference)
);

CREATE INDEX IF NOT EXISTS idx_payments_customer_id ON payments(customer_id);
