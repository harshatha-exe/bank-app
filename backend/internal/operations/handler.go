package operations

import (
	"net/http"
	"strconv"
	"time"

	"bank-app-ioc/backend/internal/account"
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

// ListAccounts returns accounts assigned to the operations user.
// @Summary List assigned accounts
// @Tags operations
// @Produce json
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response{data=[]bank-app-ioc_backend_internal_operations.Account}
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Router /operations/accounts [get]
func (h *Handler) ListAccounts(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}
	data, err := h.service.ListAccounts(c, s.UserID)
	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	httpx.OK(c, http.StatusOK, data, nil)
}

// Transactions lists transactions for an assigned account.
// @Summary List assigned account transactions
// @Tags operations
// @Produce json
// @Param accountId path string true "Account UUID, e.g. 7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"
// @Param type query string false "Transaction type, e.g. DEBIT" Enums(DEBIT,CREDIT)
// @Param page query int false "Page number, e.g. 1"
// @Param pageSize query int false "Page size, e.g. 20"
// @Param sort query string false "Sort field, e.g. createdAt" Enums(createdAt,amount,type)
// @Param direction query string false "Sort direction, e.g. DESC" Enums(ASC,DESC)
// @Param from query string false "RFC3339 start time, e.g. 2026-10-01T00:00:00Z"
// @Param to query string false "RFC3339 end time, e.g. 2026-10-31T23:59:59Z"
// @Param minAmount query string false "Minimum amount, e.g. 100.00"
// @Param maxAmount query string false "Maximum amount, e.g. 5000.00"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response{data=[]bank-app-ioc_backend_internal_operations.Transaction}
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Router /operations/accounts/{accountId}/transactions [get]
func (h *Handler) Transactions(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		h.denied(c, s.UserID)
		httpx.Fail(c, apperr.NotFound())
		return
	}
	assigned, err := h.service.Assigned(c, s.UserID, id)
	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	if !assigned {
		h.denied(c, s.UserID)
		httpx.Fail(c, apperr.NotFound())
		return
	}
	filter, err := account.ParseTransactionFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	data, total, err := h.service.Transactions(c, s.UserID, id, filter)
	if err != nil {
		if IsNotFound(err) {
			h.denied(c, s.UserID)
			httpx.Fail(c, apperr.NotFound())
			return
		}
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	httpx.OK(c, http.StatusOK, data, gin.H{"page": filter.Page, "pageSize": filter.PageSize, "totalItems": total, "totalPages": (total + filter.PageSize - 1) / filter.PageSize})
}

func parsePage(c *gin.Context) (int, int, error) {
	page, size := 1, 20
	if value := c.Query("page"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return 0, 0, apperr.Validation(map[string]string{"page": "must be a positive integer"})
		}
		page = n
	}
	if value := c.Query("pageSize"); value != "" {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 || n > 100 {
			return 0, 0, apperr.Validation(map[string]string{"pageSize": "must be between 1 and 100"})
		}
		size = n
	}
	return page, size, nil
}
func parseTime(value, name string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, apperr.Validation(map[string]string{name: "must be RFC 3339"})
	}
	return &parsed, nil
}

// ListTransfers returns transfers touching assigned accounts.
// @Summary List assigned transfers
// @Tags operations
// @Produce json
// @Param reference query string false "Exact reference number, e.g. TRF-9F3A7C2E51B8D604"
// @Param status query string false "Transfer status, e.g. COMPLETED" Enums(PENDING,COMPLETED,FAILED)
// @Param reviewStatus query string false "Review status, e.g. UNDER_REVIEW" Enums(NONE,UNDER_REVIEW,RESOLVED)
// @Param dateFrom query string false "RFC3339 start time, e.g. 2026-10-01T00:00:00Z"
// @Param dateTo query string false "RFC3339 end time, e.g. 2026-10-31T23:59:59Z"
// @Param page query int false "Page number, e.g. 1"
// @Param pageSize query int false "Page size, e.g. 20"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response{data=[]bank-app-ioc_backend_internal_operations.Transfer}
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Router /operations/transfers [get]
func (h *Handler) ListTransfers(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}
	page, size, err := parsePage(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	status, review := c.Query("status"), c.Query("reviewStatus")
	if status != "" && status != "PENDING" && status != "COMPLETED" && status != "FAILED" {
		httpx.Fail(c, apperr.Validation(map[string]string{"status": "must be PENDING, COMPLETED, or FAILED"}))
		return
	}
	if review != "" && review != "NONE" && review != "UNDER_REVIEW" && review != "RESOLVED" {
		httpx.Fail(c, apperr.Validation(map[string]string{"reviewStatus": "must be NONE, UNDER_REVIEW, or RESOLVED"}))
		return
	}
	from, err := parseTime(c.Query("dateFrom"), "dateFrom")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	to, err := parseTime(c.Query("dateTo"), "dateTo")
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	data, total, err := h.service.ListTransfers(c, s.UserID, repository.OperationsTransferFilter{Reference: c.Query("reference"), Status: status, ReviewStatus: review, DateFrom: from, DateTo: to, Page: page, PageSize: size})
	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	httpx.OK(c, http.StatusOK, data, gin.H{"page": page, "pageSize": size, "totalItems": total, "totalPages": (total + size - 1) / size})
}

// ReviewTransfer changes a transfer review status.
// @Summary Review transfer
// @Tags operations
// @Accept json
// @Produce json
// @Param reference path string true "Transfer reference number, e.g. TRF-9F3A7C2E51B8D604"
// @Param request body ReviewRequest true "Review request"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response{data=bank-app-ioc_backend_internal_operations.Transfer}
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /operations/transfers/{reference}/review [patch]
func (h *Handler) ReviewTransfer(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}
	var req ReviewRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	data, old, err := h.service.ReviewTransfer(c, s.UserID, c.Param("reference"), req.ReviewStatus)
	if err != nil {
		if IsNotFound(err) {
			h.denied(c, s.UserID)
			httpx.Fail(c, apperr.NotFound())
			return
		}
		httpx.Fail(c, err)
		return
	}
	h.audit.Record(c, &s.UserID, "TRANSFER_REVIEWED", "UPDATE", "SUCCESS", c.ClientIP(), httpx.CorrelationID(c), map[string]string{"reference": data.ReferenceNumber, "oldStatus": old, "newStatus": data.ReviewStatus})
	httpx.OK(c, http.StatusOK, data, nil)
}

// Freeze freezes an assigned account.
// @Summary Freeze account
// @Tags operations
// @Produce json
// @Param accountId path string true "Account UUID, e.g. 7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response{data=bank-app-ioc_backend_internal_operations.Account}
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /operations/accounts/{accountId}/freeze [post]
func (h *Handler) Freeze(c *gin.Context) { h.changeStatus(c, "FROZEN", "ACCOUNT_FROZEN") }

// Unfreeze unfreezes an assigned account.
// @Summary Unfreeze account
// @Tags operations
// @Produce json
// @Param accountId path string true "Account UUID, e.g. 7f4e2a10-8c3d-4b91-a6e2-1d5f7c9b2048"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response{data=bank-app-ioc_backend_internal_operations.Account}
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /operations/accounts/{accountId}/unfreeze [post]
func (h *Handler) Unfreeze(c *gin.Context) { h.changeStatus(c, "ACTIVE", "ACCOUNT_UNFROZEN") }
func (h *Handler) changeStatus(c *gin.Context, status, event string) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}
	id, err := uuid.Parse(c.Param("accountId"))
	if err != nil {
		h.denied(c, s.UserID)
		httpx.Fail(c, apperr.NotFound())
		return
	}
	data, changed, err := h.service.SetAccountStatus(c, s.UserID, id, status)
	if err != nil {
		if IsNotFound(err) {
			h.denied(c, s.UserID)
			httpx.Fail(c, apperr.NotFound())
			return
		}
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	if changed {
		h.audit.Record(c, &s.UserID, event, "UPDATE", "SUCCESS", c.ClientIP(), httpx.CorrelationID(c), map[string]string{"accountId": id.String()})
	}
	httpx.OK(c, http.StatusOK, data, nil)
}

func (h *Handler) denied(c *gin.Context, userID uuid.UUID) {
	h.audit.Record(c, &userID, "ACCESS_DENIED", c.Request.Method, "FAILURE", c.ClientIP(), httpx.CorrelationID(c), nil)
}
