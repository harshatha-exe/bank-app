package admin

import (
	"testing"

	"bank-app-ioc/backend/internal/apperr"
	"bank-app-ioc/backend/repository"
)

func TestUserQueryClean(t *testing.T) {
	longSearch := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	cases := []struct {
		name       string
		query      UserQuery
		wantSearch string
		wantRole   string
		wantStatus string
		wantPage   int
		wantSize   int
		field      string
	}{
		{name: "defaults", wantPage: 1, wantSize: 25},
		{name: "valid values", query: UserQuery{Search: "  alice ", Role: " customer ", Status: " active ", Page: "2", PageSize: "10"}, wantSearch: "alice", wantRole: "CUSTOMER", wantStatus: "ACTIVE", wantPage: 2, wantSize: 10},
		{name: "invalid page", query: UserQuery{Page: "0"}, field: "page"},
		{name: "invalid page size", query: UserQuery{PageSize: "101"}, field: "pageSize"},
		{name: "invalid role", query: UserQuery{Role: "ROOT"}, field: "role"},
		{name: "invalid status", query: UserQuery{Status: "PENDING"}, field: "status"},
		{name: "search too long", query: UserQuery{Search: longSearch}, field: "search"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			filter, page, size, err := tc.query.clean()
			if tc.field != "" {
				if err == nil {
					t.Fatal("expected validation error")
				}
				appErr, ok := err.(*apperr.Error)
				if !ok || appErr.Fields[tc.field] == "" {
					t.Fatalf("error fields = %#v, want %q", appErr.Fields, tc.field)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			want := repository.UserFilter{Search: tc.wantSearch, Role: tc.wantRole, Status: tc.wantStatus}
			if filter != want || page != tc.wantPage || size != tc.wantSize {
				t.Fatalf("clean() = %#v, %d, %d; want %#v, %d, %d", filter, page, size, want, tc.wantPage, tc.wantSize)
			}
		})
	}
}
