-- migrations/create_workers_table.up.sql
CREATE TABLE workers (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name TEXT UNIQUE NOT NULL,
    type TEXT NOT NULL,
    status TEXT DEFAULT 'online',
    last_seen TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    orders_processed INTEGER DEFAULT 0
);