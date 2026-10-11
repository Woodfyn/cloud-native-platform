package tests

import "embed"

//go:embed 20261010195218_integration_test_get_orders_data.sql
var GetOrders embed.FS

//go:embed 20261011134605_integration_test_create_orders_data.sql
var CreateOrders embed.FS
