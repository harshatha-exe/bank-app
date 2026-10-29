package transfer

import (
	"errors"
	"net/http"
	"strconv"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/httpx"
	"bank-app-ioc/backend/internal/telemetry"
	"bank-app-ioc/backend/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
	audit   *audit.Service
}

func NewHandler(service *Service, auditService *audit.Service) *Handler {
	return &Handler{service: service, audit: auditService}
}

// Create executes a transfer.
// @Summary Create transfer
// @Tags transfers
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Transfer request"
// @Param Idempotency-Key header string true "Unique idempotency key"
// @Success 201 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 409 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse "IDEMPOTENCY_CONFLICT"
// @Failure 422 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse "INSUFFICIENT_FUNDS, ACCOUNT_FROZEN, BENEFICIARY_INACTIVE, TRANSFER_LIMIT_EXCEEDED, VALIDATION_ERROR"
// @Failure 429 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse "RATE_LIMITED; Retry-After contains seconds until retry"
// @Header 429 {string} Retry-After
// @Security SessionAuth
// @Security CSRFToken
// @Router /transfers [post]
func (h *Handler) Create(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}

	var req CreateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}

	data, _, err := h.service.Create(
		c,
		s.UserID,
		correlationID(c),
		req,
		c.GetHeader("Idempotency-Key"),
		c.ClientIP(),
	)

	if err != nil {
		code := transferFailureCode(err)

		telemetry.CountTransfer(
			c.Request.Context(),
			code,
		)

		metadata := map[string]string{
			"reason":               code,
			"sourceAccountId":      req.SourceAccountID.String(),
			"destinationAccountId": req.DestinationAccountID.String(),
			"amount":               req.Amount,
		}

		h.audit.Record(
			c,
			&s.UserID,
			"TRANSFER_FAILED",
			"CREATE",
			"FAILURE",
			c.ClientIP(),
			correlationID(c),
			metadata,
		)

		if _, ok := err.(*apperr.Error); ok {
			httpx.Fail(c, err)
		} else {
			httpx.Fail(c, apperr.Internal(err))
		}

		return
	}

	telemetry.CountTransfer(
		c.Request.Context(),
		"completed",
	)

	httpx.OK(c, http.StatusCreated, data, nil)
}

func transferFailureCode(err error) string {
	const internal = "INTERNAL_ERROR"

	var appErr *apperr.Error

	if errors.As(err, &appErr) {
		switch appErr.Code {
		case "INSUFFICIENT_FUNDS", "ACCOUNT_FROZEN", "BENEFICIARY_INACTIVE",
			"TRANSFER_LIMIT_EXCEEDED", "VALIDATION_ERROR", "IDEMPOTENCY_CONFLICT", "NOT_FOUND":
			return appErr.Code
		}
	}

	return internal
}

// Get returns a transfer by reference number.
// @Summary Get transfer
// @Tags transfers
// @Accept json
// @Produce json
// @Param referenceNumber path string true "Transfer reference number"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Router /transfers/{referenceNumber} [get]
func (h *Handler) Get(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}

	if s.Role != "CUSTOMER" && s.Role != "OPERATIONS" {
		httpx.Fail(c, apperr.Forbidden())
		return
	}

	var data Transfer
	var err error

	if s.Role == "OPERATIONS" {
		data, err = h.service.GetOperations(
			c,
			s.UserID,
			c.Param("referenceNumber"),
		)
	} else {
		data, err = h.service.GetCustomer(
			c,
			s.UserID,
			c.Param("referenceNumber"),
		)
	}

	if errors.Is(err, repository.ErrNotFound) {
		h.audit.Record(
			c,
			&s.UserID,
			"ACCESS_DENIED",
			"GET",
			"FAILURE",
			c.ClientIP(),
			correlationID(c),
			nil,
		)

		httpx.Fail(c, apperr.NotFound())
		return
	}

	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}

	httpx.OK(c, http.StatusOK, data, nil)
}

// List returns transfers for the current customer.
// @Summary List transfers
// @Tags transfers
// @Accept json
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Router /transfers [get]
func (h *Handler) List(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}

	page, size := 1, 20

	if value := c.Query("page"); value != "" {
		n, err := strconv.Atoi(value)

		if err != nil || n < 1 {
			httpx.Fail(
				c,
				apperr.Validation(
					map[string]string{
						"page": "must be a positive integer",
					},
				),
			)

			return
		}

		page = n
	}

	if value := c.Query("pageSize"); value != "" {
		n, err := strconv.Atoi(value)

		if err != nil || n < 1 || n > 100 {
			httpx.Fail(
				c,
				apperr.Validation(
					map[string]string{
						"pageSize": "must be between 1 and 100",
					},
				),
			)

			return
		}

		size = n
	}

	data, total, err := h.service.List(
		c,
		s.UserID,
		page,
		size,
	)

	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}

	httpx.OK(
		c,
		http.StatusOK,
		data,
		gin.H{
			"page":       page,
			"pageSize":   size,
			"totalItems": total,
			"totalPages": (total + size - 1) / size,
		},
	)
}

func correlationID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString("correlation_id"))
	return id
}
