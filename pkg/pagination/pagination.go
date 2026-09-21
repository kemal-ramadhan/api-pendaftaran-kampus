package pagination

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"gorm.io/gorm"
)

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

var safeSortRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{1,50}$`)

// Query memuat parameter paginasi dan filter pencarian
type Query struct {
	Page    int    `json:"page"`
	PerPage int    `json:"per_page"`
	SortBy  string `json:"sort_by,omitempty"`
	Order   string `json:"order,omitempty"`
	Search  string `json:"search,omitempty"`
}

// FromContext membaca dan membersihkan parameter query string dari request Gin
func FromContext(c *gin.Context) Query {
	page := DefaultPage
	if pStr := strings.TrimSpace(c.Query("page")); pStr != "" {
		if p, err := strconv.Atoi(pStr); err == nil && p > 0 {
			page = p
		}
	}

	perPage := DefaultPerPage
	if ppStr := strings.TrimSpace(c.Query("per_page")); ppStr != "" {
		if pp, err := strconv.Atoi(ppStr); err == nil && pp > 0 {
			if pp > MaxPerPage {
				perPage = MaxPerPage
			} else {
				perPage = pp
			}
		}
	}

	order := strings.ToLower(strings.TrimSpace(c.Query("order")))
	if order != "asc" && order != "desc" {
		order = "desc"
	}

	sortBy := strings.TrimSpace(c.Query("sort_by"))
	if !safeSortRegex.MatchString(sortBy) {
		sortBy = ""
	}

	search := strings.TrimSpace(c.Query("search"))
	if search == "" {
		search = strings.TrimSpace(c.Query("q"))
	}

	return Query{
		Page:    page,
		PerPage: perPage,
		SortBy:  sortBy,
		Order:   order,
		Search:  search,
	}
}

// Offset menghitung offset database
func (q Query) Offset() int {
	if q.Page <= 1 {
		return 0
	}
	return (q.Page - 1) * q.PerPage
}

// Limit menghitung limit database
func (q Query) Limit() int {
	if q.PerPage <= 0 {
		return DefaultPerPage
	}
	if q.PerPage > MaxPerPage {
		return MaxPerPage
	}
	return q.PerPage
}

// Scope mengembalikan GORM scope untuk limit, offset, dan sorting
func Scope(q Query) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		tx := db.Offset(q.Offset()).Limit(q.Limit())
		if q.SortBy != "" && safeSortRegex.MatchString(q.SortBy) {
			orderClause := q.SortBy + " " + q.Order
			tx = tx.Order(orderClause)
		}
		return tx
	}
}

// BuildMeta membentuk metadata pagination standar response.Meta
func BuildMeta(total int64, q Query) *response.Meta {
	return &response.Meta{
		Page:    q.Page,
		PerPage: q.Limit(),
		Total:   total,
	}
}
