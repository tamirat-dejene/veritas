package pagination

import (
	"math"
	"strings"
)

// Params represents pagination parameters derived from a request.
type Params struct {
	Page    int    // Current page number (1-indexed)
	Limit   int    // Maximum number of items per page
	Sort    string // Field to sort by
	SortDir string // Sort direction ("asc" or "desc")
}

// GetOffset computes the offset for SQL queries based on Page and Limit.
func (p Params) GetOffset() int {
	return (p.GetPage() - 1) * p.GetLimit()
}

// GetLimit returns a safe limit value.
func (p Params) GetLimit() int {
	if p.Limit <= 0 {
		return 10
	}
	if p.Limit > 1000 {
		return 1000
	}
	return p.Limit
}

// GetPage returns a safe page value (1-indexed).
func (p Params) GetPage() int {
	if p.Page <= 0 {
		return 1
	}
	return p.Page
}

// GetSort returns the sort field, defaulting to "created_at" if empty.
func (p Params) GetSort() string {
	if strings.TrimSpace(p.Sort) == "" {
		return "created_at"
	}
	return p.Sort
}

// GetSortDir returns a valid SQL sort direction ("ASC" or "DESC").
func (p Params) GetSortDir() string {
	dir := strings.ToUpper(strings.TrimSpace(p.SortDir))
	if dir == "ASC" || dir == "DESC" {
		return dir
	}
	return "DESC" // Default to DESC
}

// Metadata provides pagination insights to the client.
type Metadata struct {
	CurrentPage   int   `json:"current_page"`
	PageSize      int   `json:"page_size"`
	TotalElements int64 `json:"total_elements"`
	TotalPages    int   `json:"total_pages"`
	HasNext       bool  `json:"has_next"`
	HasPrevious   bool  `json:"has_previous"`
}

// PaginatedResponse wraps generic data with pagination metadata.
type PaginatedResponse[T any] struct {
	Data     []T      `json:"data"`
	Metadata Metadata `json:"metadata"`
}

// NewPaginatedResponse returns a paginated response containing the provided data and constructed metadata.
func NewPaginatedResponse[T any](data []T, totalElements int64, params Params) PaginatedResponse[T] {
	page := params.GetPage()
	limit := params.GetLimit()

	totalPages := int(math.Ceil(float64(totalElements) / float64(limit)))

	hasPrevious := page > 1
	hasNext := page < totalPages

	metadata := Metadata{
		CurrentPage:   page,
		PageSize:      limit,
		TotalElements: totalElements,
		TotalPages:    totalPages,
		HasNext:       hasNext,
		HasPrevious:   hasPrevious,
	}

	if data == nil {
		data = make([]T, 0)
	}

	return PaginatedResponse[T]{
		Data:     data,
		Metadata: metadata,
	}
}
