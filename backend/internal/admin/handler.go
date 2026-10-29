package admin

import (
	"net/http"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func reqMeta(c *gin.Context) RequestMeta {
	return RequestMeta{IP: c.ClientIP(), CorrelationID: httpx.CorrelationID(c)}
}

// caller returns the logged-in admin's ID. Identity ALWAYS comes from the session.
func caller(c *gin.Context) (uuid.UUID, bool) {
	s, ok := httpx.Session(c)
	if !ok {
		httpx.Fail(c, apperr.Unauthorized())
		return uuid.Nil, false
	}
	return s.UserID, true
}

// pathID reads a UUID from the URL. A malformed ID is just "not found".
func pathID(c *gin.Context, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(name))
	if err != nil {
		httpx.Fail(c, apperr.NotFound())
		return uuid.Nil, false
	}
	return id, true
}

// CreateUser creates an operations or customer user.
// @Summary Create user
// @Tags admin
// @Accept json
// @Produce json
// @Param request body CreateUserRequest true "User request"
// @Success 201 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 409 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /admin/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	actor, ok := caller(c)
	if !ok {
		return
	}
	var req CreateUserRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	u, err := h.svc.CreateUser(c.Request.Context(), actor, req, reqMeta(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, u, nil)
}

// SetStatus changes a user's status.
// @Summary Set user status
// @Tags admin
// @Accept json
// @Produce json
// @Param userId path string true "User UUID"
// @Param request body StatusRequest true "Status request"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /admin/users/{userId}/status [patch]
func (h *Handler) SetStatus(c *gin.Context) {
	actor, ok := caller(c)
	if !ok {
		return
	}
	target, ok := pathID(c, "userId")
	if !ok {
		return
	}
	var req StatusRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	status, err := h.svc.SetStatus(c.Request.Context(), actor, target, req, reqMeta(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, gin.H{"id": target, "status": status}, nil)
}

// ChangeRole changes a user's role.
// @Summary Change user role
// @Tags admin
// @Accept json
// @Produce json
// @Param userId path string true "User UUID"
// @Param request body RoleRequest true "Role request"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /admin/users/{userId}/role [patch]
func (h *Handler) ChangeRole(c *gin.Context) {
	actor, ok := caller(c)
	if !ok {
		return
	}
	target, ok := pathID(c, "userId")
	if !ok {
		return
	}
	var req RoleRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	role, err := h.svc.ChangeRole(c.Request.Context(), actor, target, req, reqMeta(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, gin.H{"id": target, "role": role}, nil)
}

// ResetPassword resets a user's password.
// @Summary Reset user password
// @Tags admin
// @Accept json
// @Produce json
// @Param userId path string true "User UUID"
// @Param request body ResetPasswordRequest true "Password request"
// @Success 204
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 404 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /admin/users/{userId}/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	actor, ok := caller(c)
	if !ok {
		return
	}
	target, ok := pathID(c, "userId")
	if !ok {
		return
	}
	var req ResetPasswordRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), actor, target, req, reqMeta(c)); err != nil {
		httpx.Fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// AssignAccount assigns an account to an operations user.
// @Summary Assign account
// @Tags admin
// @Accept json
// @Produce json
// @Param request body AssignRequest true "Assignment request"
// @Success 201 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 409 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /admin/account-assignments [post]
func (h *Handler) AssignAccount(c *gin.Context) {
	actor, ok := caller(c)
	if !ok {
		return
	}
	var req AssignRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	a, err := h.svc.AssignAccount(c.Request.Context(), actor, req, reqMeta(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, http.StatusCreated, a, nil)
}

// GetLimit returns the configured transaction limit.
// @Summary Get transaction limit
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Router /admin/config/transaction-limit [get]
func (h *Handler) GetLimit(c *gin.Context) {
	l, err := h.svc.GetLimit(c.Request.Context())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, l, nil)
}

// SetLimit updates the transaction limit.
// @Summary Set transaction limit
// @Tags admin
// @Accept json
// @Produce json
// @Param request body LimitRequest true "Limit request"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Security CSRFToken
// @Router /admin/config/transaction-limit [patch]
func (h *Handler) SetLimit(c *gin.Context) {
	actor, ok := caller(c)
	if !ok {
		return
	}
	var req LimitRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		httpx.Fail(c, err)
		return
	}
	l, err := h.svc.SetLimit(c.Request.Context(), actor, req, reqMeta(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, l, nil)
}

// SearchAudit searches audit events.
// @Summary Search audit events
// @Tags admin
// @Accept json
// @Produce json
// @Param eventType query string false "Event type"
// @Param actorUserId query string false "Actor user UUID"
// @Param entityType query string false "Entity type"
// @Param entityId query string false "Entity UUID"
// @Param result query string false "Result"
// @Param from query string false "RFC3339 start time"
// @Param to query string false "RFC3339 end time"
// @Param page query int false "Page number"
// @Param pageSize query int false "Page size"
// @Success 200 {object} bank-app-ioc_backend_internal_httpx.Response
// @Failure 400 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 401 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Failure 403 {object} bank-app-ioc_backend_internal_httpx.ErrorResponse
// @Security SessionAuth
// @Router /admin/audit-events [get]
func (h *Handler) SearchAudit(c *gin.Context) {
	q := AuditQuery{
		EventType: c.Query("eventType"), ActorUserID: c.Query("actorUserId"), EntityType: c.Query("entityType"),
		EntityID: c.Query("entityId"), Result: c.Query("result"), From: c.Query("from"), To: c.Query("to"),
		Page: c.Query("page"), PageSize: c.Query("pageSize"),
	}
	res, err := h.svc.SearchAudit(c.Request.Context(), q)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, http.StatusOK, res.Events, gin.H{
		"page": res.Page, "pageSize": res.Size, "totalItems": res.Total,
		"totalPages": (res.Total + res.Size - 1) / res.Size,
	})
}
