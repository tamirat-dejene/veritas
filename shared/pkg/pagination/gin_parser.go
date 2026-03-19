package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

// ParseGin extracts pagination parameters from the Gin context query string.
func ParseGin(c *gin.Context) Params {
	pageStr := c.Query("page")
	limitStr := c.Query("limit")
	sort := c.Query("sort")
	sortDir := c.Query("sort_dir")

	page := 1
	if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
		page = p
	}

	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	if sort == "" {
		sort = "created_at"
	}

	if sortDir == "" {
		sortDir = "desc"
	}

	return Params{
		Page:    page,
		Limit:   limit,
		Sort:    sort,
		SortDir: sortDir,
	}
}
