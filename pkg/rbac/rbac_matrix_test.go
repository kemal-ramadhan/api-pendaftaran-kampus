package rbac_test

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/rbac"
)

type EndpointRBACRule struct {
	No             string
	Service        string
	Method         string
	Path           string
	Permission     string
	ScopeChecked   string
	AuthorizedRoles []string
}

// findDocsRBAC mencari file docs/RBAC.md dari direktori pengujian
func findDocsRBAC() (string, error) {
	candidates := []string{
		"../../docs/RBAC.md",
		"../docs/RBAC.md",
		"docs/RBAC.md",
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs, nil
			}
		}
	}
	// Fallback cari ke parent directories
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for i := 0; i < 5; i++ {
		target := filepath.Join(dir, "docs", "RBAC.md")
		if _, err := os.Stat(target); err == nil {
			return target, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", os.ErrNotExist
}

// parseRBACMatrix membaca docs/RBAC.md dan mem-parsing tabel Section 3
func parseRBACMatrix(filePath string) ([]EndpointRBACRule, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var rules []EndpointRBACRule
	scanner := bufio.NewScanner(file)
	inTable := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "| No | Service | Method |") {
			inTable = true
			continue
		}
		if inTable && strings.HasPrefix(line, "|---|") {
			continue
		}
		if inTable {
			if !strings.HasPrefix(line, "|") {
				// Akhir tabel
				break
			}
			parts := strings.Split(line, "|")
			if len(parts) >= 8 {
				no := strings.TrimSpace(parts[1])
				service := strings.TrimSpace(parts[2])
				method := strings.TrimSpace(parts[3])
				path := strings.TrimSpace(parts[4])
				perm := strings.TrimSpace(parts[5])
				scope := strings.TrimSpace(parts[6])
				rolesRaw := strings.TrimSpace(parts[7])

				// Hilangkan backticks
				perm = strings.ReplaceAll(perm, "`", "")
				path = strings.ReplaceAll(path, "`", "")

				// Parse roles
				rawRoles := strings.Split(rolesRaw, ",")
				var roles []string
				for _, r := range rawRoles {
					cleaned := strings.TrimSpace(strings.ReplaceAll(r, "`", ""))
					if cleaned != "" {
						roles = append(roles, cleaned)
					}
				}

				if no != "" && perm != "" {
					rules = append(rules, EndpointRBACRule{
						No:             no,
						Service:        service,
						Method:         method,
						Path:           path,
						Permission:     perm,
						ScopeChecked:   scope,
						AuthorizedRoles: roles,
					})
				}
			}
		}
	}

	return rules, scanner.Err()
}

func TestRBACMatrixFromDocs(t *testing.T) {
	path, err := findDocsRBAC()
	if err != nil {
		t.Fatalf("docs/RBAC.md tidak ditemukan: %v", err)
	}

	rules, err := parseRBACMatrix(path)
	if err != nil {
		t.Fatalf("Gagal mem-parsing docs/RBAC.md: %v", err)
	}

	if len(rules) == 0 {
		t.Fatalf("Tabel RBAC di docs/RBAC.md kosong atau tidak ter-parsing")
	}

	t.Logf("Berhasil mem-parsing %d endpoint admin dari docs/RBAC.md", len(rules))

	allRoles := []string{"superadmin", "admin_pmb", "keuangan", "prodi", "dekan", "pimpinan", "dosen", "calon_mahasiswa"}

	for _, rule := range rules {
		// Endpoint publik/hanya authenticated umum dilewati
		if strings.Contains(rule.Permission, "Authenticated") {
			continue
		}

		perm := rule.Permission
		authMap := make(map[string]bool)
		for _, r := range rule.AuthorizedRoles {
			authMap[r] = true
		}

		// Siapkan router Gin dengan middleware RequirePermission(perm)
		r := gin.New()
		r.Use(rbac.RequirePermission(perm))
		r.Handle(rule.Method, "/test-endpoint", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"authorized": true})
		})

		// 1. Uji Penerimaan: Role yang berhak (dengan header X-User-Permissions) harus diterima (200 OK)
		for _, role := range rule.AuthorizedRoles {
			if role == "Semua Role Login" {
				continue
			}
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(rule.Method, "/test-endpoint", nil)
			req.Header.Set("X-User-Permissions", perm+",other.perm")

			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("[Rule #%s %s %s] Role berhak '%s' ditolak; status = %d, body: %s",
					rule.No, rule.Method, rule.Path, role, w.Code, w.Body.String())
			}
		}

		// 2. Uji Penolakan: Role yang tidak berhak (atau tanpa permission) harus ditolak 403 PERMISSION_DENIED
		for _, role := range allRoles {
			if !authMap[role] {
				w := httptest.NewRecorder()
				req, _ := http.NewRequest(rule.Method, "/test-endpoint", nil)
				// Kirim permissions lain yang tidak mencakup perm yang dibutuhkan
				req.Header.Set("X-User-Permissions", "unrelated.permission,dummy.read")

				r.ServeHTTP(w, req)
				if w.Code != http.StatusForbidden {
					t.Errorf("[Rule #%s %s %s] Role tidak berhak '%s' diizinkan (harus 403); didapat %d",
						rule.No, rule.Method, rule.Path, role, w.Code)
				}

				var resp struct {
					Success   bool   `json:"success"`
					ErrorCode string `json:"error_code"`
				}
				if err := json.Unmarshal(w.Body.Bytes(), &resp); err == nil {
					if resp.ErrorCode != rbac.ErrCodePermissionDenied {
						t.Errorf("[Rule #%s] Error code = %s; expected %s", rule.No, resp.ErrorCode, rbac.ErrCodePermissionDenied)
					}
				}
			}
		}
	}
}
