package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bank-app-ioc/backend/internal/auth"
	"bank-app-ioc/backend/internal/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type listUsersStub struct {
	page UserPage
	got  UserQuery
}

func (s *listUsersStub) ListUsers(_ context.Context, q UserQuery) (UserPage, error) {
	s.got = q
	return s.page, nil
}

func TestListUsersMockedHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	id := uuid.New()
	stub := &listUsersStub{page: UserPage{
		Users: []UserDTO{{ID: id, Username: "alice", Role: "CUSTOMER", Status: "ACTIVE", CreatedAt: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)}},
		Total: 31, Page: 2, Size: 15,
	}}
	router := gin.New()
	router.Use(middleware.ErrorHandler(), func(c *gin.Context) {
		c.Set("session", auth.Session{UserID: uuid.New(), Role: "ADMIN"})
		c.Next()
	})
	router.GET("/users", (&Handler{users: stub}).ListUsers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users?search=%20alice%20&role=customer&status=active&page=2&pageSize=15", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data []map[string]any `json:"data"`
		Meta map[string]any   `json:"meta"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if len(response.Data) != 1 {
		t.Fatalf("data length = %d", len(response.Data))
	}
	for key := range response.Data[0] {
		switch key {
		case "id", "username", "role", "status", "createdAt":
		default:
			t.Errorf("unexpected user response key %q", key)
		}
	}
	if response.Meta["page"] != float64(2) || response.Meta["pageSize"] != float64(15) || response.Meta["totalItems"] != float64(31) || response.Meta["totalPages"] != float64(3) {
		t.Fatalf("meta = %#v", response.Meta)
	}
	if stub.got.Search != " alice " || stub.got.Role != "customer" || stub.got.Status != "active" || stub.got.Page != "2" || stub.got.PageSize != "15" {
		t.Fatalf("query passed to stub = %#v", stub.got)
	}
}

func TestListUsersMockedHandlerRejectsInvalidQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	stub := &listUsersStub{}
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	router.GET("/users", (&Handler{users: stub}).ListUsers)

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/users?role=ROOT&page=0", nil)
	router.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Error struct {
			Code   string            `json:"code"`
			Fields map[string]string `json:"fields"`
		} `json:"error"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != "VALIDATION_ERROR" || response.Error.Fields["role"] == "" || response.Error.Fields["page"] == "" {
		t.Fatalf("error = %#v", response.Error)
	}
	if stub.got != (UserQuery{}) {
		t.Fatalf("stub was called for invalid query: %#v", stub.got)
	}
}
