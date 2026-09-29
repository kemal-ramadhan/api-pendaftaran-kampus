package rbac

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
)

const (
	HeaderUserPermissions = "X-User-Permissions"
	ErrCodePermissionDenied = "PERMISSION_DENIED"
)

// GetUserPermissions mengekstrak daftar kode permissions dari header X-User-Permissions
func GetUserPermissions(c *gin.Context) []string {
	raw := strings.TrimSpace(c.GetHeader(HeaderUserPermissions))
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	perms := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			perms = append(perms, trimmed)
		}
	}
	return perms
}

// HasPermission memeriksa apakah pengguna memiliki kode permission tertentu
func HasPermission(c *gin.Context, requiredPerm string) bool {
	for _, p := range GetUserPermissions(c) {
		if p == requiredPerm {
			return true
		}
	}
	return false
}

// HasAnyPermission memeriksa apakah pengguna memiliki setidaknya salah satu dari permission yang ditentukan
func HasAnyPermission(c *gin.Context, requiredPerms ...string) bool {
	userPerms := GetUserPermissions(c)
	for _, req := range requiredPerms {
		for _, up := range userPerms {
			if up == req {
				return true
			}
		}
	}
	return false
}

// RequirePermission adalah middleware Gin untuk memastikan user memiliki salah satu permission yang ditentukan
func RequirePermission(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !HasAnyPermission(c, permissions...) {
			response.Error(
				c,
				http.StatusForbidden,
				ErrCodePermissionDenied,
				"Akses ditolak: izin tidak mencukupi",
				map[string][]string{"required_permissions": permissions},
			)
			c.Abort()
			return
		}
		c.Next()
	}
}

// ScopeResult merepresentasikan hasil evaluasi scope akses pengguna
type ScopeResult struct {
	IsUnrestricted bool
	Role           string
	ScopeType      string
	ScopeID        string
	ProdiCode      string
	FacultyCode    string
}

// ScopeFilter mengekstrak konteks hak akses prodi/fakultas pengguna
func ScopeFilter(c *gin.Context) ScopeResult {
	roles := gatewayauth.GetUserRoles(c)
	scopes := gatewayauth.GetUserScopes(c)

	// Peran tingkat universitas/sistem memiliki akses bebas scope
	for _, r := range roles {
		if r == "superadmin" || r == "admin_pmb" || r == "pimpinan" || r == "keuangan" {
			return ScopeResult{IsUnrestricted: true, Role: r}
		}
	}

	res := ScopeResult{IsUnrestricted: false}
	for _, s := range scopes {
		if s.Role == "prodi" || s.Type == "prodi" {
			res.Role = "prodi"
			res.ScopeType = "prodi"
			res.ScopeID = s.ID
			res.ProdiCode = s.ID
			return res
		}
		if s.Role == "dekan" || s.Type == "faculty" {
			res.Role = "dekan"
			res.ScopeType = "faculty"
			res.ScopeID = s.ID
			res.FacultyCode = s.ID
			return res
		}
	}

	// Fallback jika role prodi atau dekan ada di roles tanpa scope eksplisit
	for _, r := range roles {
		if r == "prodi" {
			res.Role = "prodi"
			return res
		}
		if r == "dekan" {
			res.Role = "dekan"
			return res
		}
	}

	return res
}
