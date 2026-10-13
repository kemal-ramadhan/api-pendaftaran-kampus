package seeder

import (
	"testing"

	"github.com/pkm-asia/pmb-backend/services/gateway/internal/auth/model"
)

func TestDefaultRolesList(t *testing.T) {
	// PRD Bagian 4.1 mewajibkan tepat 8 roles standar
	expectedRoles := map[string]string{
		model.RoleCalonMahasiswa: "Calon Mahasiswa",
		model.RoleMahasiswa:      "Mahasiswa",
		model.RoleAdminPMB:       "Admin PMB",
		model.RoleKeuangan:       "Keuangan",
		model.RoleDosen:          "Dosen",
		model.RoleProdi:          "Prodi",
		model.RoleDekan:          "Dekan",
		model.RolePimpinan:       "Pimpinan Universitas",
		model.RoleSuperadmin:     "Superadmin",
	}

	if len(defaultRoles) != 9 {
		t.Fatalf("Jumlah role bawaan harus tepat 9, didapat %d", len(defaultRoles))
	}

	for _, role := range defaultRoles {
		expectedName, exists := expectedRoles[role.Code]
		if !exists {
			t.Errorf("Role code tidak terdaftar di ekspektasi PRD: %s", role.Code)
		}
		if role.Name != expectedName {
			t.Errorf("Nama role untuk %s = %s; expected %s", role.Code, role.Name, expectedName)
		}
	}
}
