package auth

import (
	"net/http"
	"time"

	"bank-app-ioc/backend/internal/audit"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const sessionCookie = "session_id"

type Handler struct {
	service    *Service
	audit      *audit.Service
	csrfSecret string
}

func NewHandler(service *Service, auditService *audit.Service, csrfSecret string) *Handler {
	return &Handler{service: service, audit: auditService, csrfSecret: csrfSecret}
}

func (h *Handler) Signup(c *gin.Context) {
	var req SignupRequest
	if c.ShouldBindJSON(&req) != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		return
	}
	u, err := h.service.Signup(c, req)
	if err != nil {
		if IsUniqueViolation(err) {
			writeError(c, http.StatusConflict, "USERNAME_TAKEN", "Username is already taken.")
			return
		}
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"user": PublicUserFor(u)}, "meta": meta(c, "")})
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if c.ShouldBindJSON(&req) != nil {
		writeError(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request.")
		return
	}
	u, sessionID, err := h.service.Login(c, req)
	if err != nil {
		h.audit.Record(c, nil, "LOGIN_FAILURE", "LOGIN", "FAILURE", c.ClientIP(), correlationID(c), nil)
		writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid username or password.")
		return
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: sessionCookie, Value: sessionID, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, Expires: time.Now().Add(h.service.maxLifetime)})
	h.audit.Record(c, &u.ID, "LOGIN_SUCCESS", "LOGIN", "SUCCESS", c.ClientIP(), correlationID(c), nil)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"user": PublicUserFor(u)}, "meta": meta(c, csrfToken(sessionID, h.csrfSecret))})
}

func (h *Handler) Logout(c *gin.Context) {
	sessionID, _ := c.Cookie(sessionCookie)
	if err := h.service.Logout(c, sessionID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error.")
		return
	}
	c.SetCookie(sessionCookie, "", -1, "/", "", true, true)
	h.audit.Record(c, actorID(c), "LOGOUT", "LOGOUT", "SUCCESS", c.ClientIP(), correlationID(c), nil)
	c.Status(http.StatusNoContent)
}

func (h *Handler) Me(c *gin.Context) {
	value, _ := c.Get("session")
	s, ok := value.(Session)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication required.")
		return
	}
	raw, _ := c.Cookie(sessionCookie)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"id": s.UserID, "username": s.Username, "role": s.Role}, "meta": meta(c, csrfToken(raw, h.csrfSecret))})
}

func meta(c *gin.Context, csrf string) gin.H {
	value := gin.H{"correlationId": correlationID(c)}
	if csrf != "" {
		value["csrfToken"] = csrf
	}
	return value
}
func correlationID(c *gin.Context) uuid.UUID {
	value, _ := uuid.Parse(c.GetString("correlation_id"))
	return value
}
func actorID(c *gin.Context) *uuid.UUID {
	value, ok := c.Get("session")
	if !ok {
		return nil
	}
	s, ok := value.(Session)
	if !ok {
		return nil
	}
	return &s.UserID
}
func writeError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{"error": gin.H{"code": code, "message": message}, "meta": gin.H{"correlationId": correlationID(c)}})
}
