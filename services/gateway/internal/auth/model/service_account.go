package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ServiceAccount mewakili katalog layanan/produk kampus (CR-012)
type ServiceAccount struct {
	ID                     uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	Code                   string         `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Name                   string         `gorm:"type:varchar(100);not null" json:"name"`
	Description            string         `gorm:"type:text" json:"description"`
	ProductGroup           string         `gorm:"type:varchar(50);not null;default:'pmb'" json:"product_group"`
	BaseURL                string         `gorm:"type:varchar(255)" json:"base_url"`
	Audience               string         `gorm:"type:varchar(100);not null;default:'pmb-api'" json:"audience"`
	ClientType             string         `gorm:"type:varchar(30);not null;default:'public'" json:"client_type"` // public / confidential
	ClientSecretHash       *string        `gorm:"type:varchar(255)" json:"-"`
	RedirectURIs           string         `gorm:"type:text" json:"redirect_uris"`                       // Disimpan comma-separated atau JSON
	PostLogoutRedirectURIs string         `gorm:"type:text" json:"post_logout_redirect_uris"`           // Disimpan comma-separated atau JSON
	AllowedScopes          string         `gorm:"type:varchar(255);default:'openid profile email'" json:"allowed_scopes"`
	FirstParty             bool           `gorm:"default:true" json:"first_party"`
	IsActive               bool           `gorm:"default:true" json:"is_active"`
	CreatedAt              time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt              time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt              gorm.DeletedAt `gorm:"index" json:"-"`
}

func (sa *ServiceAccount) BeforeCreate(tx *gorm.DB) error {
	if sa.ID == uuid.Nil {
		sa.ID = uuid.New()
	}
	return nil
}

const (
	UserAccountStatusActive    = "active"
	UserAccountStatusSuspended = "suspended"
	UserAccountStatusRevoked   = "revoked"
)

// UserAccount mewakili relasi otorisasi user ke suatu service account
type UserAccount struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	UserID           uuid.UUID      `gorm:"type:uuid;index;not null" json:"user_id"`
	ServiceAccountID uuid.UUID      `gorm:"type:uuid;index;not null" json:"service_account_id"`
	Status           string         `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	GrantedBy        *uuid.UUID     `gorm:"type:uuid" json:"granted_by,omitempty"`
	GrantedAt        time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"granted_at"`
	LastAccessAt     *time.Time     `gorm:"type:timestamptz" json:"last_access_at,omitempty"`
	CreatedAt        time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`

	ServiceAccount   ServiceAccount    `gorm:"foreignKey:ServiceAccountID" json:"service_account,omitempty"`
	UserAccountRoles []UserAccountRole `gorm:"foreignKey:UserAccountID" json:"user_account_roles,omitempty"`
}

func (ua *UserAccount) BeforeCreate(tx *gorm.DB) error {
	if ua.ID == uuid.Nil {
		ua.ID = uuid.New()
	}
	return nil
}

// UserAccountRole mewakili role spesifik user pada service account tertentu
type UserAccountRole struct {
	UserAccountID uuid.UUID  `gorm:"type:uuid;primaryKey" json:"user_account_id"`
	RoleID        uint       `gorm:"primaryKey" json:"role_id"`
	ScopeType     *string    `gorm:"type:varchar(30);primaryKey;default:''" json:"scope_type,omitempty"`
	ScopeID       *string    `gorm:"type:varchar(50);primaryKey;default:''" json:"scope_id,omitempty"`
	CreatedAt     time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	CreatedBy     *uuid.UUID `gorm:"type:uuid" json:"created_by,omitempty"`

	Role Role `gorm:"foreignKey:RoleID" json:"role"`
}

// SigningKey mewakili kunci penandatanganan JWT / JWKS untuk rotasi kunci
type SigningKey struct {
	Kid       string     `gorm:"type:varchar(50);primaryKey" json:"kid"`
	PublicKey string     `gorm:"type:text;not null" json:"public_key"`
	Algorithm string     `gorm:"type:varchar(20);not null;default:'RS256'" json:"algorithm"`
	IsActive  bool       `gorm:"default:true" json:"is_active"`
	CreatedAt time.Time  `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	ExpiresAt *time.Time `gorm:"type:timestamptz" json:"expires_at,omitempty"`
}
