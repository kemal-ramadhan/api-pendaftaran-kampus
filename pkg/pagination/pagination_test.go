package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestPagination_FromContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Default values", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/items", nil)

		q := FromContext(c)
		if q.Page != 1 {
			t.Errorf("Expected page 1, got %d", q.Page)
		}
		if q.PerPage != 20 {
			t.Errorf("Expected per_page 20, got %d", q.PerPage)
		}
		if q.Order != "desc" {
			t.Errorf("Expected order desc, got %s", q.Order)
		}
		if q.Offset() != 0 {
			t.Errorf("Expected offset 0, got %d", q.Offset())
		}
		if q.Limit() != 20 {
			t.Errorf("Expected limit 20, got %d", q.Limit())
		}
	})

	t.Run("Custom values and bounds", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/items?page=3&per_page=500&sort_by=created_at&order=asc&search=kampus", nil)

		q := FromContext(c)
		if q.Page != 3 {
			t.Errorf("Expected page 3, got %d", q.Page)
		}
		if q.PerPage != 100 { // MaxPerPage capped
			t.Errorf("Expected per_page capped at 100, got %d", q.PerPage)
		}
		if q.SortBy != "created_at" {
			t.Errorf("Expected sort_by created_at, got %s", q.SortBy)
		}
		if q.Order != "asc" {
			t.Errorf("Expected order asc, got %s", q.Order)
		}
		if q.Search != "kampus" {
			t.Errorf("Expected search kampus, got %s", q.Search)
		}
		if q.Offset() != 200 { // (3 - 1) * 100
			t.Errorf("Expected offset 200, got %d", q.Offset())
		}

		meta := BuildMeta(250, q)
		if meta.Page != 3 || meta.PerPage != 100 || meta.Total != 250 {
			t.Errorf("BuildMeta mismatch: %+v", meta)
		}
	})

	t.Run("Unsafe sort_by sanitized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/items?sort_by=id;DROP%20TABLE%20users", nil)

		q := FromContext(c)
		if q.SortBy != "" {
			t.Errorf("Unsafe sort_by harus dikosongkan, got %s", q.SortBy)
		}
	})
}
