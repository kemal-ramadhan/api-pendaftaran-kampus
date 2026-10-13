package model

import (
	"time"

	"github.com/google/uuid"
)

// Role Codes standar PMB sesuai PRD Bagian 4.1
const (
	RoleCalonMahasiswa = "calon_mahasiswa"
	RoleMahasiswa      = "mahasiswa"
	RoleAdminPMB       = "admin_pmb"
	RoleKeuangan       = "keuangan"
	RoleDosen          = "dosen"
	RoleProdi          = "prodi"
	RoleDekan          = "dekan"
	RolePimpinan       = "pimpinan"
	RoleSuperadmin     = "superadmin"
)

type Role struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

type Permission struct {
	Code        string    `gorm:"primaryKey;type:varchar(100)" json:"code"`
	Module      string    `gorm:"type:varchar(50);not null" json:"module"`
	Description string    `gorm:"type:text" json:"description"`
	CreatedAt   time.Time `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
}

type RolePermission struct {
	RoleID         uint       `gorm:"primaryKey" json:"role_id"`
	PermissionCode string     `gorm:"primaryKey;type:varchar(100)" json:"permission_code"`
	CreatedAt      time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	Role           Role       `gorm:"foreignKey:RoleID" json:"role"`
	Permission     Permission `gorm:"foreignKey:PermissionCode" json:"permission"`
}

type UserRole struct {
	UserID    uuid.UUID  `gorm:"type:uuid;primaryKey" json:"user_id"`
	RoleID    uint       `gorm:"primaryKey" json:"role_id"`
	ScopeType *string    `gorm:"type:varchar(30);primaryKey;default:''" json:"scope_type,omitempty"`
	ScopeID   *string    `gorm:"type:varchar(50);primaryKey;default:''" json:"scope_id,omitempty"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`

	Role Role `gorm:"foreignKey:RoleID" json:"role"`
}
