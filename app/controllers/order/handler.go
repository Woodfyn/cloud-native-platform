package order

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"time"
	"uuid"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type Handler struct {
	psqlConn *pgx.Conn
	log      *slog.Logger
}

func NewHandler(
	psqlConn *pgx.Conn,
	log *slog.Logger,
) *Handler {
	return &Handler{
		psqlConn: psqlConn,
		log:      log,
	}
}

func setError(
	ctx *gin.Context,
	code int,
	message string,
	errs ...error,
) {
	result := map[string]any{
		"message": message,
	}

	if len(errs) > 0 {
		result["internal"] = errs[0]
	}

	ctx.JSON(code, result)
}

const _getQuery = `SELECT
order_id,
customer_name,
order_number,
total_amount,
created_at,
updated_at
FROM orders`

func (h *Handler) Get(ctx *gin.Context) {
	rows, err := h.psqlConn.Query(
		ctx.Request.Context(),
		_getQuery,
	)
	if err != nil {
		setError(
			ctx,
			http.StatusInternalServerError,
			"Failed to query orders",
			err,
		)
		return
	}
	defer rows.Close()

	type orderDB struct {
		OrderID      string
		CustomerName string
		OrderNumber  string
		TotalAmount  float64
		CreatedAt    time.Time
		UpdatedAt    sql.NullTime
	}

	orderRows := make([]orderDB, 0)
	for rows.Next() {
		var row orderDB
		if err := rows.Scan(
			&row.OrderID,
			&row.CustomerName,
			&row.OrderNumber,
			&row.TotalAmount,
			&row.CreatedAt,
			&row.UpdatedAt,
		); err != nil {
			setError(
				ctx,
				http.StatusInternalServerError,
				"Failed to scan orders",
				err,
			)
			return
		}

		orderRows = append(orderRows, row)
	}

	if err := rows.Err(); err != nil {
		setError(
			ctx,
			http.StatusInternalServerError,
			"Failed to iterate orders",
			err,
		)
		return
	}

	type orderResult struct {
		OrderID      string  `json:"order_id"`
		CustomerName string  `json:"customer_name"`
		OrderNumber  string  `json:"order_number"`
		TotalAmount  float64 `json:"total_amount"`
		CreatedAt    string  `json:"created_at"`
		UpdatedAt    *string `json:"updated_at,omitempty"`
	}

	orderResults := make([]orderResult, len(orderRows))

	for i, row := range orderRows {
		var updatedAt *string
		if row.UpdatedAt.Valid {
			value := row.UpdatedAt.Time.Format(time.DateTime)
			updatedAt = &value
		}

		orderResults[i] = orderResult{
			OrderID:      row.OrderID,
			CustomerName: row.CustomerName,
			OrderNumber:  row.OrderNumber,
			TotalAmount:  row.TotalAmount,
			CreatedAt:    row.CreatedAt.Format(time.DateTime),
			UpdatedAt:    updatedAt,
		}
	}

	ctx.JSON(http.StatusOK, orderResults)
}

const _createQuery = `INSERT INTO orders (
    order_id,
    customer_name,
    order_number,
    total_amount
) VALUES (
    $1,
	$2,
	$3,
	$4
);`

func (h *Handler) Create(ctx *gin.Context) {
	type orderInputSchema struct {
		CustomerName string  `json:"customer_name"`
		OrderNumber  string  `json:"order_number"`
		TotalAmount  float64 `json:"total_amount"`
	}

	var orderInput orderInputSchema
	if err := ctx.BindJSON(&orderInput); err != nil {
		setError(
			ctx,
			http.StatusBadRequest,
			"Invalid request body",
			err,
		)
		return
	}
	if orderInput.TotalAmount <= 0 {
		setError(
			ctx,
			http.StatusBadRequest,
			"Invalid order total amount",
		)
		return
	}

	orderID := uuid.New().String()
	totalAmount := strconv.FormatFloat(orderInput.TotalAmount, 'f', 2, 64)
	if _, err := h.psqlConn.Exec(
		ctx.Request.Context(),
		_createQuery,
		orderID,
		orderInput.CustomerName,
		orderInput.OrderNumber,
		totalAmount,
	); err != nil {
		setError(
			ctx,
			http.StatusInternalServerError,
			"Failed to create order",
			err,
		)
		return
	}

	ctx.JSON(http.StatusOK, map[string]string{
		"message":  "Order was created",
		"order_id": orderID,
	})
}
