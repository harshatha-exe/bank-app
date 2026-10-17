package transfer

import (
	"errors"
	"net/http"
	"strconv"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/httpx"
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
	data, _, err := h.service.Create(c, s.UserID, correlationID(c), req, c.GetHeader("Idempotency-Key"), c.ClientIP())
	if err != nil {
		if _, ok := err.(*apperr.Error); ok {
			httpx.Fail(c, err)
		} else {
			httpx.Fail(c, apperr.Internal(err))
		}
		return
	}
	httpx.OK(c, http.StatusCreated, data, nil)
}
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
		data, err = h.service.GetOperations(c, s.UserID, c.Param("referenceNumber"))
	} else {
		data, err = h.service.GetCustomer(c, s.UserID, c.Param("referenceNumber"))
	}
	if errors.Is(err, repository.ErrNotFound) {
		h.audit.Record(c, &s.UserID, "ACCESS_DENIED", "GET", "FAILURE", c.ClientIP(), correlationID(c), nil)
		httpx.Fail(c, apperr.NotFound())
		return
	}
	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	httpx.OK(c, http.StatusOK, data, nil)
}
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
			httpx.Fail(c, apperr.Validation(map[string]string{"page": "must be a positive integer"}))
			return
		}
		page = n
	}
	if value := c.Query("pageSize"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 100 {
			httpx.Fail(c, apperr.Validation(map[string]string{"pageSize": "must be between 1 and 100"}))
			return
		}
		size = n
	}
	data, total, err := h.service.List(c, s.UserID, page, size)
	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	httpx.OK(c, http.StatusOK, data, gin.H{"page": page, "pageSize": size, "totalItems": total, "totalPages": (total + size - 1) / size})
}
func correlationID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString("correlation_id"))
	return id
}
