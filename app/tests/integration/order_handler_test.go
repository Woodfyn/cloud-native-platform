//nolint:gochecknoglobals // For tests global variable is able.
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/Woodfyn/cloud-native-platform/controllers/order"
	"github.com/Woodfyn/cloud-native-platform/migrations/tests"
	"github.com/gin-gonic/gin"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pressly/goose/v3"
)

var (
	testDB       *TestDatabase
	orderHandler *order.Handler
	testRouter   *gin.Engine
)

// https://medium.com/@dilshataliev/integration-tests-with-golang-test-containers-and-postgres-abb49e8096c5
func TestMain(m *testing.M) {
	testDB = SetupTestDatabase()

	orderHandler = order.NewHandler(
		testDB.DBInstance,
		slog.Default(),
	)

	testRouter = gin.Default()
	gin.SetMode(gin.TestMode)

	code := m.Run()

	testDB.TearDown()

	os.Exit(code)
}

type mockHTTPWriter struct {
	bytes.Buffer

	statusCode int
	header     http.Header
}

func (m *mockHTTPWriter) Header() http.Header {
	return m.header
}

func (m *mockHTTPWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func TestCreateOrder(t *testing.T) {
	tt := []struct {
		name           string
		in             order.InputSchema
		wantStatusCode int
	}{
		{
			name: "Test_1_happy_test",
			in: order.InputSchema{
				CustomerName: "Volodymyr Rud",
				OrderNumber:  "1234",
				TotalAmount:  100.24,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "Test_2_negative_total_amount",
			in: order.InputSchema{
				CustomerName: "Volodymyr Rud",
				OrderNumber:  "1234",
				TotalAmount:  -10.48,
			},
			wantStatusCode: http.StatusBadRequest,
		},
	}

	for _, test := range tt {
		writer := &mockHTTPWriter{
			Buffer: *bytes.NewBuffer(nil),
			header: make(http.Header),
		}

		inBytes, err := json.Marshal(&test.in)
		if err != nil {
			t.Errorf("Failed to marshal test payload: %v", err)
			return
		}

		ctx := gin.CreateTestContextOnly(writer, testRouter)
		ctx.Request = &http.Request{
			Body:   io.NopCloser(bytes.NewReader(inBytes)),
			Header: make(http.Header),
		}

		orderHandler.Create(ctx)

		if writer.statusCode != test.wantStatusCode {
			t.Errorf(
				"%s: Incorrect status code (want: %d, have: %d)",
				test.name,
				writer.statusCode,
				test.wantStatusCode,
			)
			continue
		}

		t.Logf("%s: Passed", test.name)
	}
}

func TestGetOrders(t *testing.T) {
	dbURL, err := testDB.container.ConnectionString(
		t.Context(),
		"sslmode=disable",
	)
	if err != nil {
		t.Errorf(
			"Failed to get connection string: %v",
			err,
		)
		return
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Errorf(
			"Failed to open sql connection: %v",
			err,
		)
		return
	}

	if err = db.PingContext(t.Context()); err != nil {
		t.Errorf(
			"Failed to ping to connection: %v",
			err,
		)
		return
	}

	goose.SetBaseFS(tests.Migrations)

	if err = goose.SetDialect("postgres"); err != nil {
		t.Errorf(
			"Failed to set dialect for goose: %v",
			err,
		)
		return
	}

	if err = goose.UpContext(t.Context(), db, "."); err != nil {
		t.Errorf(
			"Failed to run goose migrations: %v",
			err,
		)
		return
	}

	t.Cleanup(func() {
		defer db.Close()

		if err = goose.DownContext(context.Background(), db, "."); err != nil {
			t.Errorf(
				"Failed to run goose migrations: %v",
				err,
			)
			return
		}
	})

	tt := []struct {
		name           string
		wantOrders     []order.Order
		wantStatusCode int
	}{
		{
			name: "Test_1_happy_test",
			wantOrders: []order.Order{
				{
					OrderID:      "a0000000-0000-4000-8000-000000000001",
					CustomerName: "John Smith",
					OrderNumber:  "ORD-0001",
					TotalAmount:  150.00,
					CreatedAt:    "",
					UpdatedAt:    nil,
				},
				{
					OrderID:      "a0000000-0000-4000-8000-000000000002",
					CustomerName: "Alice Johnson",
					OrderNumber:  "ORD-0002",
					TotalAmount:  275.50,
					CreatedAt:    "",
					UpdatedAt:    nil,
				},
				{
					OrderID:      "a0000000-0000-4000-8000-000000000003",
					CustomerName: "Bob Williams",
					OrderNumber:  "ORD-0003",
					TotalAmount:  420.00,
					CreatedAt:    "",
					UpdatedAt:    nil,
				},
			},
			wantStatusCode: http.StatusOK,
		},
	}

	for _, test := range tt {
		mockWriter := &mockHTTPWriter{
			Buffer: *bytes.NewBuffer([]byte{}),
			header: make(http.Header),
		}

		ctx := gin.CreateTestContextOnly(mockWriter, testRouter)
		ctx.Request = &http.Request{
			Header: make(http.Header),
		}

		orderHandler.Get(ctx)

		if mockWriter.statusCode != test.wantStatusCode {
			t.Errorf(
				"%s: Incorrect status code (want: %d, have: %d)",
				test.name,
				mockWriter.statusCode,
				test.wantStatusCode,
			)
			continue
		}

		var resultBytes []byte
		resultBytes, err = io.ReadAll(&mockWriter.Buffer)
		if err != nil {
			t.Errorf(
				"Failed to read mock writer buffer: %v",
				err,
			)
			continue
		}

		var resultOrders []order.Order
		if err = json.Unmarshal(resultBytes, &resultOrders); err != nil {
			t.Errorf(
				"Failed to unmarshal response: %v",
				err,
			)
			continue
		}

		// t.Logf(
		// 	"Orders reuslt, want: %+v; got: %+v",
		// 	test.wantOrders,
		// 	resultOrders,
		// )

		diff := cmp.Diff(
			test.wantOrders,
			resultOrders,
			cmpopts.IgnoreFields(
				order.Order{},
				"CreatedAt",
				"UpdatedAt",
			),
		)
		if diff != "" {
			t.Errorf("Orders mismatch (-want +got):\n%s", diff)
			continue
		}

		t.Logf("%s: Passed", test.name)
	}
}
