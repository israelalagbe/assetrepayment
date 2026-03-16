-- Seed a test customer for development and integration testing.
INSERT OR IGNORE INTO customers (id, outstanding_kobo, total_paid_kobo)
VALUES ('GIGXXXXX', 100000000, 0);
