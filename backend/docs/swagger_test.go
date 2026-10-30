package docs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

func TestSwaggerDocumentServedAndCoversAPI(t *testing.T) {
	raw, err := os.ReadFile("swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}

	router := gin.New()
	router.GET("/swagger/doc.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json; charset=utf-8", raw)
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("swagger status = %d", recorder.Code)
	}

	routes := []struct {
		method string
		path   string
	}{
		{"GET", "/accounts"}, {"GET", "/accounts/:accountId"}, {"GET", "/accounts/:accountId/transactions"},
		{"POST", "/admin/account-assignments"}, {"GET", "/admin/audit-events"},
		{"GET", "/admin/config/transaction-limit"}, {"PATCH", "/admin/config/transaction-limit"},
		{"GET", "/admin/users"}, {"POST", "/admin/users"}, {"POST", "/admin/users/:userId/reset-password"},
		{"PATCH", "/admin/users/:userId/role"}, {"PATCH", "/admin/users/:userId/status"},
		{"POST", "/auth/login"}, {"POST", "/auth/logout"}, {"POST", "/auth/signup"}, {"GET", "/me"},
		{"GET", "/beneficiaries"}, {"POST", "/beneficiaries"},
		{"PATCH", "/beneficiaries/:beneficiaryId/status"}, {"GET", "/transfers"},
		{"POST", "/transfers"}, {"GET", "/transfers/:referenceNumber"},
	}
	for _, route := range routes {
		path := route.path
		for i := 0; i < len(path); i++ {
			if path[i] == ':' {
				end := i + 1
				for end < len(path) && path[end] != '/' {
					end++
				}
				path = path[:i] + "{" + path[i+1:end] + "}" + path[end:]
				i = end
			}
		}
		if _, ok := document.Paths[path]; !ok {
			t.Errorf("%s %s is missing from swagger paths", route.method, path)
			continue
		}
		if _, ok := document.Paths[path][strings.ToLower(route.method)]; !ok {
			t.Errorf("%s %s is missing from swagger operations", route.method, path)
		}
	}
	if _, err := swag.ReadDoc(); err != nil {
		t.Fatal(err)
	}
}
