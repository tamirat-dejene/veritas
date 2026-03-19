package pagination

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseGin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name          string
		query         string
		expectPage    int
		expectLimit   int
		expectSort    string
		expectSortDir string
	}{
		{
			name:          "No params",
			query:         "/",
			expectPage:    1,
			expectLimit:   10,
			expectSort:    "created_at",
			expectSortDir: "desc",
		},
		{
			name:          "Valid params",
			query:         "/?page=2&limit=20&sort=email&sort_dir=asc",
			expectPage:    2,
			expectLimit:   20,
			expectSort:    "email",
			expectSortDir: "asc",
		},
		{
			name:          "Invalid types fallback",
			query:         "/?page=abc&limit=xyz",
			expectPage:    1,
			expectLimit:   10,
			expectSort:    "created_at",
			expectSortDir: "desc",
		},
		{
			name:          "Negative limits ignored by parser (handled by getters)",
			query:         "/?page=-1&limit=-10",
			expectPage:    1,
			expectLimit:   10,
			expectSort:    "created_at",
			expectSortDir: "desc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", tc.query, nil)

			params := ParseGin(c)

			if params.Page != tc.expectPage {
				t.Errorf("expected page %d, got %d", tc.expectPage, params.Page)
			}
			if params.Limit != tc.expectLimit {
				t.Errorf("expected limit %d, got %d", tc.expectLimit, params.Limit)
			}
			if params.Sort != tc.expectSort {
				t.Errorf("expected sort %s, got %s", tc.expectSort, params.Sort)
			}
			if params.SortDir != tc.expectSortDir {
				t.Errorf("expected sort_dir %s, got %s", tc.expectSortDir, params.SortDir)
			}
		})
	}
}
