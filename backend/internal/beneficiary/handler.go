package beneficiary

import (
	"errors"
	"net/http"

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

// List returns the current customer's beneficiaries.
// @Summary List beneficiaries
// @Tags beneficiaries
// @Accept json
// @Produce json
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Router /beneficiaries [get]
func (h *Handler) List(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}
	data, err := h.service.List(c, s.UserID)
	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	httpx.OK(c, http.StatusOK, data, nil)
}

// Create adds a beneficiary.
// @Summary Create beneficiary
// @Tags beneficiaries
// @Accept json
// @Produce json
// @Param request body CreateRequest true "Beneficiary request"
// @Success 201 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 409 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /beneficiaries [post]
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
	data, err := h.service.Create(c, s.UserID, req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			httpx.Fail(c, apperr.NotFound())
			return
		}
		if errors.Is(err, repository.ErrAlreadyExists) {
			httpx.Fail(c, apperr.Conflict("DUPLICATE_BENEFICIARY", "Beneficiary already exists."))
			return
		}
		if _, ok := err.(*apperr.Error); ok {
			httpx.Fail(c, err)
			return
		}
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	h.audit.Record(c, &s.UserID, "BENEFICIARY_CREATED", "CREATE", "SUCCESS", c.ClientIP(), correlationID(c), nil)
	httpx.OK(c, http.StatusCreated, data, nil)
}

// UpdateStatus changes a beneficiary status.
// @Summary Update beneficiary status
// @Tags beneficiaries
// @Accept json
// @Produce json
// @Param beneficiaryId path string true "Beneficiary UUID"
// @Param request body StatusRequest true "Status request"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /beneficiaries/{beneficiaryId}/status [patch]
func (h *Handler) UpdateStatus(c *gin.Context) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return
	}
	id, err := uuid.Parse(c.Param("beneficiaryId"))
	if err != nil {
		h.denied(c, s.UserID)
		httpx.Fail(c, apperr.NotFound())
		return
	}
	var req StatusRequest
	if err = httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	data, err := h.service.UpdateStatus(c, s.UserID, id, req.Status)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			h.denied(c, s.UserID)
			httpx.Fail(c, apperr.NotFound())
			return
		}
		if _, ok := err.(*apperr.Error); ok {
			httpx.Fail(c, err)
			return
		}
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	if data.Status == "DEACTIVATED" {
		h.audit.Record(c, &s.UserID, "BENEFICIARY_DEACTIVATED", "UPDATE", "SUCCESS", c.ClientIP(), correlationID(c), nil)
	}
	httpx.OK(c, http.StatusOK, data, nil)
}
func (h *Handler) denied(c *gin.Context, userID uuid.UUID) {
	h.audit.Record(c, &userID, "ACCESS_DENIED", c.Request.Method, "FAILURE", c.ClientIP(), correlationID(c), nil)
}
func correlationID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString("correlation_id"))
	return id
}
