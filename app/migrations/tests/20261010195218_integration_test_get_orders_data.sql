-- +goose Up

INSERT INTO orders (
    order_id,
    customer_name,
    order_number,
    total_amount
) VALUES
('a0000000-0000-4000-8000-000000000001', 'John Smith',    'ORD-0001', 150.00),
('a0000000-0000-4000-8000-000000000002', 'Alice Johnson', 'ORD-0002', 275.50),
('a0000000-0000-4000-8000-000000000003', 'Bob Williams',  'ORD-0003', 420.00);

-- +goose Down

DELETE FROM orders
WHERE order_id IN (
    'a0000000-0000-4000-8000-000000000001',
    'a0000000-0000-4000-8000-000000000002',
    'a0000000-0000-4000-8000-000000000003'
);