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
	defer testDB.TearDown()

	testRouter = gin.Default()
	gin.SetMode(gin.TestMode)

	orderHandler = order.NewHandler(
		testDBInstance,
		slog.Default(),
	)

	os.Exit(m.Run())
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
		in             order.OrderInputSchema
		wantStatusCode int
	}{
		{
			in: order.OrderInputSchema{
				CustomerName: "Volodymyr Rud",
				OrderNumber:  "1234",
				TotalAmount:  100.24,
			},
			wantStatusCode: http.StatusOK,
		},
		{
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
				"Incorrect status code (want: %d, have: %d)",
				writer.statusCode,
				test.wantStatusCode,
			)
			continue
		}
	}
}

func TestGetOrders(t *testing.T) {

}
