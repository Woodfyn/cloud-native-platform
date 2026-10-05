package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/Woodfyn/cloud-native-platform/controllers/order"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	testDBInstance *pgxpool.Pool
	orderHandler   *order.Handler
	testRouter     *gin.Engine
)

// https://medium.com/@dilshataliev/integration-tests-with-golang-test-containers-and-postgres-abb49e8096c5
func TestMain(m *testing.M) {
	testDB := SetupTestDatabase()

	testDBInstance = testDB.DBInstance

	orderHandler = order.NewHandler(
		testDBInstance,
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
		in             order.OrderInputSchema
		wantStatusCode int
	}{
		{
			name: "Test_1_happy_test",
			in: order.OrderInputSchema{
				CustomerName: "Volodymyr Rud",
				OrderNumber:  "1234",
				TotalAmount:  100.24,
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "Test_2_negative_total_amount",
			in: order.OrderInputSchema{
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
	// t.Errorf("TODO: Implement me!")
}
