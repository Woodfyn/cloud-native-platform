CREATE TABLE IF NOT EXISTS orders (
    id UUID PRIMARY KEY,
    customer_id UUID NOT NULL,
    order_number VARCHAR(50) NOT NULL UNIQUE,

    status VARCHAR(30) NOT NULL DEFAULT 'pending',

    pickup_address TEXT,
    delivery_address TEXT,

    pickup_date TIMESTAMP,
    delivery_date TIMESTAMP,

    total_amount NUMERIC(12, 2) NOT NULL DEFAULT 0,

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);