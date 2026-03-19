package pagination

import (
	"testing"
)

func TestParamsVariablesAndDefaults(t *testing.T) {
	tests := []struct {
		name          string
		params        Params
		expectOffset  int
		expectLimit   int
		expectPage    int
		expectSort    string
		expectSortDir string
	}{
		{
			name:   "Defaults on zero values",
			params: Params{},
			expectOffset:  0,
			expectLimit:   10,
			expectPage:    1,
			expectSort:    "created_at",
			expectSortDir: "DESC",
		},
		{
			name: "Valid values",
			params: Params{
				Page:    3,
				Limit:   20,
				Sort:    "updated_at",
				SortDir: "asc",
			},
			expectOffset:  40,
			expectLimit:   20,
			expectPage:    3,
			expectSort:    "updated_at",
			expectSortDir: "ASC",
		},
		{
			name: "Negative limits and pages",
			params: Params{
				Page:    -1,
				Limit:   -5,
				Sort:    "   ",
				SortDir: "invalid",
			},
			expectOffset:  0,
			expectLimit:   10,
			expectPage:    1,
			expectSort:    "created_at",
			expectSortDir: "DESC",
		},
		{
			name: "High limits restricted to max 1000",
			params: Params{
				Page:    2,
				Limit:   5000,
			},
			expectOffset:  1000,
			expectLimit:   1000,
			expectPage:    2,
			expectSort:    "created_at",
			expectSortDir: "DESC",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.params.GetOffset(); got != tc.expectOffset {
				t.Errorf("expected offset %d, got %d", tc.expectOffset, got)
			}
			if got := tc.params.GetLimit(); got != tc.expectLimit {
				t.Errorf("expected limit %d, got %d", tc.expectLimit, got)
			}
			if got := tc.params.GetPage(); got != tc.expectPage {
				t.Errorf("expected page %d, got %d", tc.expectPage, got)
			}
			if got := tc.params.GetSort(); got != tc.expectSort {
				t.Errorf("expected sort %s, got %s", tc.expectSort, got)
			}
			if got := tc.params.GetSortDir(); got != tc.expectSortDir {
				t.Errorf("expected sort_dir %s, got %s", tc.expectSortDir, got)
			}
		})
	}
}

func TestNewPaginatedResponse(t *testing.T) {
	data := []string{"item1", "item2"}
	params := Params{Page: 2, Limit: 2}
	totalElements := int64(5) // (item1, item2), (item3, item4), (item5) -> total 3 pages

	resp := NewPaginatedResponse(data, totalElements, params)

	if len(resp.Data) != 2 {
		t.Errorf("expected 2 items in data, got %d", len(resp.Data))
	}

	meta := resp.Metadata
	if meta.CurrentPage != 2 {
		t.Errorf("expected current_page 2, got %d", meta.CurrentPage)
	}
	if meta.PageSize != 2 {
		t.Errorf("expected page_size 2, got %d", meta.PageSize)
	}
	if meta.TotalElements != 5 {
		t.Errorf("expected total_elements 5, got %d", meta.TotalElements)
	}
	if meta.TotalPages != 3 {
		t.Errorf("expected total_pages 3, got %d", meta.TotalPages)
	}
	if meta.HasNext != true {
		t.Errorf("expected has_next true, got %v", meta.HasNext)
	}
	if meta.HasPrevious != true {
		t.Errorf("expected has_previous true, got %v", meta.HasPrevious)
	}

	// Test case where nil data is passed
	nilResp := NewPaginatedResponse[string](nil, 0, Params{})
	if nilResp.Data == nil {
		t.Errorf("expected data to be initialized to empty slice, got nil")
	}
	if len(nilResp.Data) != 0 {
		t.Errorf("expected empty data slice, got len %d", len(nilResp.Data))
	}
}
