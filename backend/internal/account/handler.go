package account

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/internal/audit"
	"bank-app-ioc/backend/internal/httpx"
	"bank-app-ioc/backend/repository"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const defaultPageSize = 20
const maxPageSize = 100

type Handler struct {
	service *Service
	audit   *audit.Service
}

func NewHandler(service *Service, auditService *audit.Service) *Handler {
	return &Handler{service: service, audit: auditService}
}

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
func (h *Handler) Get(c *gin.Context) {
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
	data, err := h.service.Find(c, s.UserID, id)
	if err != nil {
		if err == repository.ErrNotFound {
			h.denied(c, s.UserID)
			httpx.Fail(c, apperr.NotFound())
			return
		}
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	httpx.OK(c, http.StatusOK, data, nil)
}
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
	filter, err := transactionFilter(c)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if _, err := h.service.Find(c, s.UserID, id); err != nil {
		if err == repository.ErrNotFound {
			h.denied(c, s.UserID)
			httpx.Fail(c, apperr.NotFound())
			return
		}
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	data, total, err := h.service.Transactions(c, s.UserID, id, filter)
	if err != nil {
		httpx.Fail(c, apperr.Internal(err))
		return
	}
	pages := (total + filter.PageSize - 1) / filter.PageSize
	httpx.OK(c, http.StatusOK, data, gin.H{"page": filter.Page, "pageSize": filter.PageSize, "totalItems": total, "totalPages": pages})
}
func (h *Handler) denied(c *gin.Context, userID uuid.UUID) {
	h.audit.Record(c, &userID, "ACCESS_DENIED", c.Request.Method, "FAILURE", c.ClientIP(), correlationID(c), nil)
}
func correlationID(c *gin.Context) uuid.UUID {
	id, _ := uuid.Parse(c.GetString("correlation_id"))
	return id
}

func transactionFilter(c *gin.Context) (repository.TransactionFilter, error) {
	f := repository.TransactionFilter{Page: 1, PageSize: defaultPageSize, Sort: "createdAt", Direction: "DESC", Type: c.Query("type"), MinAmount: c.Query("minAmount"), MaxAmount: c.Query("maxAmount")}
	v := c.Query("page")
	if v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 {
			return f, apperr.Validation(map[string]string{"page": "must be a positive integer"})
		}
		f.Page = n
	}
	v = c.Query("pageSize")
	if v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 || n > maxPageSize {
			return f, apperr.Validation(map[string]string{"pageSize": "must be between 1 and 100"})
		}
		f.PageSize = n
	}
	if f.Type != "" && f.Type != "DEBIT" && f.Type != "CREDIT" {
		return f, apperr.Validation(map[string]string{"type": "must be DEBIT or CREDIT"})
	}
	if v = c.Query("sort"); v != "" {
		if v != "createdAt" && v != "amount" && v != "type" {
			return f, apperr.Validation(map[string]string{"sort": "unsupported sort"})
		}
		f.Sort = v
	}
	if v = c.Query("direction"); v != "" {
		if v != "ASC" && v != "DESC" {
			return f, apperr.Validation(map[string]string{"direction": "must be ASC or DESC"})
		}
		f.Direction = v
	}
	for name, value := range map[string]string{"from": c.Query("from"), "to": c.Query("to")} {
		if value == "" {
			continue
		}
		parsed, e := time.Parse(time.RFC3339, value)
		if e != nil {
			return f, apperr.Validation(map[string]string{name: "must be RFC 3339"})
		}
		if name == "from" {
			f.From = &parsed
		} else {
			f.To = &parsed
		}
	}
	for name, value := range map[string]string{"minAmount": f.MinAmount, "maxAmount": f.MaxAmount} {
		if value == "" {
			continue
		}
		if _, ok := money(value); !ok {
			return f, apperr.Validation(map[string]string{name: "must be a decimal with at most 2 decimal places"})
		}
	}
	if f.MinAmount != "" && f.MaxAmount != "" {
		min, _ := money(f.MinAmount)
		max, _ := money(f.MaxAmount)
		if min > max {
			return f, apperr.Validation(map[string]string{"minAmount": "must not exceed maxAmount"})
		}
	}
	return f, nil
}
func money(value string) (int64, bool) {
	var whole, frac string
	parts := strings.Split(value, ".")
	if len(parts) > 2 || value == "" {
		return 0, false
	}
	whole = parts[0]
	if len(parts) == 2 {
		frac = parts[1]
	}
	if whole == "" || len(frac) > 2 {
		return 0, false
	}
	for _, r := range whole + frac {
		if r < '0' || r > '9' {
			return 0, false
		}
	}
	if len(whole) > 1 && whole[0] == '0' {
		return 0, false
	}
	for len(frac) < 2 {
		frac += "0"
	}
	w, _ := strconv.ParseInt(whole, 10, 64)
	f := int64(0)
	if frac != "" {
		f, _ = strconv.ParseInt(frac, 10, 64)
	}
	return w*100 + f, true
}
