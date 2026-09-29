package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/internalclient"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/dto"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/limiter"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/mailer"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/repository"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/token"
	"github.com/pkm-asia/pmb-backend/services/auth-service/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AdminService interface {
	GetRoles(ctx context.Context) ([]dto.RoleResponseData, error)
	ListUsers(ctx context.Context, query dto.AdminListUsersQuery) ([]dto.UserResponseData, int64, error)
	CreateUser(ctx context.Context, req dto.AdminCreateUserRequest, actorID uuid.UUID) (*dto.UserResponseData, error)
	GetUser(ctx context.Context, id uuid.UUID) (*dto.UserResponseData, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req dto.AdminUpdateUserRequest, actorID uuid.UUID) (*dto.UserResponseData, error)
	UpdateUserRoles(ctx context.Context, id uuid.UUID, req dto.AdminUpdateRolesRequest, actorID uuid.UUID) (*dto.UserResponseData, error)
	UpdateUserStatus(ctx context.Context, id uuid.UUID, req dto.AdminUpdateStatusRequest, actorID uuid.UUID) (*dto.UserResponseData, error)
	GetPermissions(ctx context.Context) ([]dto.PermissionResponseData, error)
	GetRolePermissions(ctx context.Context, roleCode string) (*dto.RolePermissionsResponseData, error)
	UpdateRolePermissions(ctx context.Context, roleCode string, permissions []string, actorID uuid.UUID) (*dto.RolePermissionsResponseData, error)

	// Internal Inter-Service Methods
	InternalGetUser(ctx context.Context, id uuid.UUID) (*dto.UserResponseData, error)
	InternalGetUsersBatch(ctx context.Context, ids []uuid.UUID) ([]dto.UserResponseData, error)
	InternalAddRole(ctx context.Context, id uuid.UUID, req dto.AddUserRoleRequest) (*dto.UserResponseData, error)
	InternalVerifyPassword(ctx context.Context, id uuid.UUID, password string) error
	InternalRemoveRole(ctx context.Context, id uuid.UUID, roleCode string) error
}

type adminService struct {
	cfg          *config.Config
	userRepo     repository.UserRepository
	roleRepo     repository.RoleRepository
	resetRepo    repository.PasswordResetRepository
	auditRepo    repository.AuditLogRepository
	sessionStore token.SessionStore
	rateLimiter  limiter.RateLimiter
	mailer       mailer.Mailer
	orgClient    internalclient.Client
	logger       *zap.Logger
}

func NewAdminService(
	cfg *config.Config,
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	resetRepo repository.PasswordResetRepository,
	auditRepo repository.AuditLogRepository,
	sessionStore token.SessionStore,
	rateLimiter limiter.RateLimiter,
	mailer mailer.Mailer,
	orgClient internalclient.Client,
	logger *zap.Logger,
) AdminService {
	return &adminService{
		cfg:          cfg,
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		resetRepo:    resetRepo,
		auditRepo:    auditRepo,
		sessionStore: sessionStore,
		rateLimiter:  rateLimiter,
		mailer:       mailer,
		orgClient:    orgClient,
		logger:       logger,
	}
}

func (s *adminService) GetRoles(ctx context.Context) ([]dto.RoleResponseData, error) {
	roles, err := s.roleRepo.FindAll(ctx)
	if err != nil {
		s.logger.Error("Gagal mengambil daftar role", zap.Error(err))
		return nil, err
	}

	result := make([]dto.RoleResponseData, 0, len(roles))
	for _, r := range roles {
		result = append(result, dto.RoleResponseData{
			ID:          r.ID,
			Code:        r.Code,
			Name:        r.Name,
			Description: r.Description,
		})
	}
	return result, nil
}

func (s *adminService) ListUsers(ctx context.Context, query dto.AdminListUsersQuery) ([]dto.UserResponseData, int64, error) {
	users, total, err := s.userRepo.FindUsers(ctx, query.Role, query.Status, query.Q, query.Page, query.PerPage)
	if err != nil {
		s.logger.Error("Gagal mengambil daftar pengguna", zap.Error(err))
		return nil, 0, err
	}

	result := make([]dto.UserResponseData, 0, len(users))
	for _, u := range users {
		result = append(result, toUserResponseData(&u))
	}
	return result, total, nil
}

func (s *adminService) CreateUser(ctx context.Context, req dto.AdminCreateUserRequest, actorID uuid.UUID) (*dto.UserResponseData, error) {
	cleanEmail := strings.ToLower(strings.TrimSpace(req.Email))
	if cleanEmail == "" {
		return nil, fmt.Errorf("%w: email wajib diisi", ErrValidation)
	}

	cleanName := strings.TrimSpace(req.FullName)
	if len(cleanName) < 2 {
		return nil, fmt.Errorf("%w: nama lengkap minimal 2 karakter", ErrValidation)
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, cleanEmail)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailTaken
	}

	var passwordHash string
	if req.Password != "" {
		if !validatePasswordPolicy(req.Password) {
			return nil, fmt.Errorf("%w: kata sandi minimal 8 karakter dan harus mengandung kombinasi huruf serta angka", ErrValidation)
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.cfg.BcryptCost)
		if err != nil {
			return nil, err
		}
		passwordHash = string(hash)
	} else if req.SendSetPasswordEmail {
		// Buat temporary password hash random sebelum diset oleh user
		randPass, _ := utils.GenerateResetToken()
		hash, err := bcrypt.GenerateFromPassword([]byte(randPass), s.cfg.BcryptCost)
		if err != nil {
			return nil, err
		}
		passwordHash = string(hash)
	} else {
		return nil, fmt.Errorf("%w: kata sandi awal atau opsi kirim tautan set-password wajib dipilih", ErrValidation)
	}

	// Validasi seluruh roles yang diminta
	var userRoles []model.UserRole
	now := time.Now()
	for _, rDTO := range req.Roles {
		roleRecord, err := s.roleRepo.FindByCode(ctx, rDTO.Code)
		if err != nil || roleRecord == nil {
			return nil, fmt.Errorf("%w: role %s tidak valid", ErrRoleNotFound, rDTO.Code)
		}

		cleanScopeType := strings.TrimSpace(rDTO.ScopeType)
		cleanScopeID := strings.TrimSpace(rDTO.ScopeID)

		if roleRecord.Code == model.RoleProdi {
			st, si, err := s.validateProdiScope(ctx, cleanScopeType, cleanScopeID)
			if err != nil {
				return nil, err
			}
			cleanScopeType = st
			cleanScopeID = si
		} else if roleRecord.Code == model.RoleDekan {
			st, si, err := s.validateDekanScope(ctx, cleanScopeType, cleanScopeID)
			if err != nil {
				return nil, err
			}
			cleanScopeType = st
			cleanScopeID = si
		}

		userRoles = append(userRoles, model.UserRole{
			RoleID:    roleRecord.ID,
			Role:      *roleRecord,
			ScopeType: &cleanScopeType,
			ScopeID:   &cleanScopeID,
			CreatedAt: now,
			CreatedBy: &actorID,
		})
	}

	phone := utils.NormalizePhone(req.Phone)
	userID := uuid.New()
	for i := range userRoles {
		userRoles[i].UserID = userID
	}

	newUser := &model.User{
		ID:              userID,
		Email:           cleanEmail,
		Phone:           phone,
		PasswordHash:    passwordHash,
		FullName:        cleanName,
		Status:          model.UserStatusActive,
		EmailVerifiedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
		UserRoles:       userRoles,
	}

	if err := s.userRepo.Create(ctx, newUser); err != nil {
		s.logger.Error("Gagal membuat user admin", zap.Error(err))
		return nil, err
	}

	// Jika opsi kirim email set-password dipilih
	if req.Password == "" && req.SendSetPasswordEmail {
		go func(u *model.User) {
			resetCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			rawToken, err := utils.GenerateResetToken()
			if err != nil {
				return
			}
			tokenHash := utils.HashToken(rawToken)
			resetRecord := &model.PasswordReset{
				ID:        uuid.New(),
				UserID:    u.ID,
				TokenHash: tokenHash,
				ExpiresAt: time.Now().Add(24 * time.Hour),
				CreatedAt: time.Now(),
			}
			if err := s.resetRepo.Create(resetCtx, resetRecord); err == nil {
				link := fmt.Sprintf("%s/reset-password?token=%s", s.cfg.FrontendURL, rawToken)
				_ = s.mailer.SendPasswordReset(resetCtx, u.Email, u.FullName, link)
			}
		}(newUser)
	}

	// Audit Log
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		UserID:    &actorID,
		Action:    "admin.user.created",
		Metadata:  fmt.Sprintf(`{"target_user_id":"%s","email":"%s"}`, userID.String(), cleanEmail),
		CreatedAt: time.Now(),
	})

	res := toUserResponseData(newUser)
	return &res, nil
}

func (s *adminService) GetUser(ctx context.Context, id uuid.UUID) (*dto.UserResponseData, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}
	res := toUserResponseData(u)
	return &res, nil
}

func (s *adminService) UpdateUser(ctx context.Context, id uuid.UUID, req dto.AdminUpdateUserRequest, actorID uuid.UUID) (*dto.UserResponseData, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	cleanName := strings.TrimSpace(req.FullName)
	if len(cleanName) < 2 {
		return nil, fmt.Errorf("%w: nama lengkap minimal 2 karakter", ErrValidation)
	}

	u.FullName = cleanName
	if req.Phone != "" {
		u.Phone = utils.NormalizePhone(req.Phone)
	}
	u.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		UserID:    &actorID,
		Action:    "admin.user.updated",
		Metadata:  fmt.Sprintf(`{"target_user_id":"%s"}`, id.String()),
		CreatedAt: time.Now(),
	})

	res := toUserResponseData(u)
	return &res, nil
}

func (s *adminService) UpdateUserRoles(ctx context.Context, id uuid.UUID, req dto.AdminUpdateRolesRequest, actorID uuid.UUID) (*dto.UserResponseData, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	// Cek apakah target user saat ini adalah superadmin
	targetWasSuperadmin := false
	for _, ur := range u.UserRoles {
		if ur.Role.Code == model.RoleSuperadmin {
			targetWasSuperadmin = true
			break
		}
	}

	// Cek apakah role baru memuat superadmin
	hasNewSuperadmin := false
	for _, rDTO := range req.Roles {
		if rDTO.Code == model.RoleSuperadmin {
			hasNewSuperadmin = true
			break
		}
	}

	// GUARD ATURAN SUPERADMIN:
	// 1. Superadmin tidak boleh mencabut role superadmin dari dirinya sendiri
	if id == actorID && targetWasSuperadmin && !hasNewSuperadmin {
		return nil, ErrSelfDemoteSuperadmin
	}

	// 2. Jika mencabut superadmin dari user lain, pastikan tidak meninggalkan sistem dengan 0 superadmin aktif
	if targetWasSuperadmin && !hasNewSuperadmin {
		activeCount, err := s.roleRepo.CountActiveSuperadmins(ctx)
		if err != nil {
			return nil, err
		}
		if activeCount <= 1 {
			return nil, ErrLastSuperadmin
		}
	}

	var newRoles []model.UserRole
	now := time.Now()
	for _, rDTO := range req.Roles {
		roleRecord, err := s.roleRepo.FindByCode(ctx, rDTO.Code)
		if err != nil || roleRecord == nil {
			return nil, fmt.Errorf("%w: role %s tidak valid", ErrRoleNotFound, rDTO.Code)
		}

		cleanScopeType := strings.TrimSpace(rDTO.ScopeType)
		cleanScopeID := strings.TrimSpace(rDTO.ScopeID)
		if roleRecord.Code == model.RoleProdi {
			st, si, err := s.validateProdiScope(ctx, cleanScopeType, cleanScopeID)
			if err != nil {
				return nil, err
			}
			cleanScopeType = st
			cleanScopeID = si
		} else if roleRecord.Code == model.RoleDekan {
			st, si, err := s.validateDekanScope(ctx, cleanScopeType, cleanScopeID)
			if err != nil {
				return nil, err
			}
			cleanScopeType = st
			cleanScopeID = si
		}

		newRoles = append(newRoles, model.UserRole{
			UserID:    id,
			RoleID:    roleRecord.ID,
			Role:      *roleRecord,
			ScopeType: &cleanScopeType,
			ScopeID:   &cleanScopeID,
			CreatedAt: now,
			CreatedBy: &actorID,
		})
	}

	if err := s.roleRepo.ReplaceUserRoles(ctx, id, newRoles); err != nil {
		return nil, err
	}

	// Cabut seluruh sesi user agar token lama tidak membawa role usang
	_ = s.sessionStore.RevokeAllUserSessions(ctx, id)

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		UserID:    &actorID,
		Action:    "admin.user.roles_updated",
		Metadata:  fmt.Sprintf(`{"target_user_id":"%s"}`, id.String()),
		CreatedAt: time.Now(),
	})

	// Refresh user object
	updatedUser, _ := s.userRepo.FindByID(ctx, id)
	if updatedUser != nil && len(updatedUser.UserRoles) == 0 {
		userRoles, _ := s.roleRepo.GetUserRoles(ctx, id)
		updatedUser.UserRoles = userRoles
	}
	res := toUserResponseData(updatedUser)
	return &res, nil
}

func (s *adminService) UpdateUserStatus(ctx context.Context, id uuid.UUID, req dto.AdminUpdateStatusRequest, actorID uuid.UUID) (*dto.UserResponseData, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	if req.Status == model.UserStatusInactive {
		// GUARD ATURAN SUPERADMIN:
		// 1. Superadmin tidak boleh menonaktifkan akun sendiri
		if id == actorID {
			return nil, ErrSelfDeactivateSuperadmin
		}

		// 2. Superadmin tidak boleh menonaktifkan superadmin terakhir
		isSuperadmin := false
		for _, ur := range u.UserRoles {
			if ur.Role.Code == model.RoleSuperadmin {
				isSuperadmin = true
				break
			}
		}

		if isSuperadmin {
			activeCount, err := s.roleRepo.CountActiveSuperadmins(ctx)
			if err != nil {
				return nil, err
			}
			if activeCount <= 1 {
				return nil, ErrLastSuperadmin
			}
		}
	}

	if err := s.userRepo.UpdateStatus(ctx, id, req.Status); err != nil {
		return nil, err
	}

	// Jika dinonaktifkan, cabut seluruh sesi di Redis
	if req.Status == model.UserStatusInactive {
		_ = s.sessionStore.RevokeAllUserSessions(ctx, id)
	}

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		UserID:    &actorID,
		Action:    "admin.user.status_updated",
		Metadata:  fmt.Sprintf(`{"target_user_id":"%s","status":"%s"}`, id.String(), req.Status),
		CreatedAt: time.Now(),
	})

	u.Status = req.Status
	res := toUserResponseData(u)
	return &res, nil
}

// --- Internal API Implementation ---

func (s *adminService) InternalGetUser(ctx context.Context, id uuid.UUID) (*dto.UserResponseData, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}
	res := toUserResponseData(u)
	return &res, nil
}

func (s *adminService) InternalGetUsersBatch(ctx context.Context, ids []uuid.UUID) ([]dto.UserResponseData, error) {
	users, err := s.userRepo.FindBatch(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]dto.UserResponseData, 0, len(users))
	for _, u := range users {
		result = append(result, toUserResponseData(&u))
	}
	return result, nil
}

func (s *adminService) InternalAddRole(ctx context.Context, id uuid.UUID, req dto.AddUserRoleRequest) (*dto.UserResponseData, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil || u == nil {
		return nil, ErrUserNotFound
	}

	roleRecord, err := s.roleRepo.FindByCode(ctx, req.Role)
	if err != nil || roleRecord == nil {
		return nil, fmt.Errorf("%w: role %s tidak valid", ErrRoleNotFound, req.Role)
	}

	cleanScopeType := strings.TrimSpace(req.ScopeType)
	cleanScopeID := strings.TrimSpace(req.ScopeID)
	if roleRecord.Code == model.RoleProdi {
		if cleanScopeType == "" {
			cleanScopeType = "prodi"
		}
		if !validateScopeID(cleanScopeID) {
			return nil, fmt.Errorf("%w: scope_id prodi tidak valid", ErrValidation)
		}
	} else if roleRecord.Code == model.RoleDekan {
		if cleanScopeType == "" {
			cleanScopeType = "faculty"
		}
		if !validateScopeID(cleanScopeID) {
			return nil, fmt.Errorf("%w: scope_id dekan tidak valid", ErrValidation)
		}
	}

	ur := &model.UserRole{
		UserID:    id,
		RoleID:    roleRecord.ID,
		Role:      *roleRecord,
		ScopeType: &cleanScopeType,
		ScopeID:   &cleanScopeID,
		CreatedAt: time.Now(),
	}

	if err := s.roleRepo.AssignRole(ctx, ur); err != nil {
		// Abaikan jika sudah ada (idempotent)
		s.logger.Warn("Role mungkin sudah ada saat assign internal", zap.Error(err))
	}

	// Cabut sesi agar klaim token user diperbarui
	_ = s.sessionStore.RevokeAllUserSessions(ctx, id)

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Action:    "internal.user.role_added",
		Metadata:  fmt.Sprintf(`{"user_id":"%s","role":"%s"}`, id.String(), req.Role),
		CreatedAt: time.Now(),
	})

	updatedUser, _ := s.userRepo.FindByID(ctx, id)
	if updatedUser != nil && len(updatedUser.UserRoles) == 0 {
		userRoles, _ := s.roleRepo.GetUserRoles(ctx, id)
		updatedUser.UserRoles = userRoles
	}
	res := toUserResponseData(updatedUser)
	return &res, nil
}

func (s *adminService) InternalVerifyPassword(ctx context.Context, id uuid.UUID, password string) error {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil || u == nil {
		return ErrUserNotFound
	}

	if s.rateLimiter != nil {
		isLocked, _, err := s.rateLimiter.IsLoginLocked(ctx, u.Email)
		if err == nil && isLocked {
			return ErrTooManyAttempts
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		if s.rateLimiter != nil {
			locked, _, _ := s.rateLimiter.RecordLoginFailure(ctx, u.Email)
			if locked {
				return ErrTooManyAttempts
			}
		}
		return ErrInvalidCredentials
	}

	if s.rateLimiter != nil {
		_ = s.rateLimiter.ResetLoginFailure(ctx, u.Email)
	}

	return nil
}

func (s *adminService) InternalRemoveRole(ctx context.Context, id uuid.UUID, roleCode string) error {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil || u == nil {
		return ErrUserNotFound
	}

	if err := s.roleRepo.RemoveUserRoleByCode(ctx, id, roleCode); err != nil {
		return err
	}

	_ = s.sessionStore.RevokeAllUserSessions(ctx, id)

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Action:    "internal.user.role_removed",
		Metadata:  fmt.Sprintf(`{"user_id":"%s","role":"%s"}`, id.String(), roleCode),
		CreatedAt: time.Now(),
	})

	return nil
}

func toUserResponseData(user *model.User) dto.UserResponseData {
	roles := make([]string, 0)
	scopes := make([]token.ScopeClaim, 0)
	for _, ur := range user.UserRoles {
		if ur.Role.Code != "" {
			roles = append(roles, ur.Role.Code)
		}
		st := ""
		if ur.ScopeType != nil {
			st = *ur.ScopeType
		}
		si := ""
		if ur.ScopeID != nil {
			si = *ur.ScopeID
		}
		if st != "" && si != "" {
			scopes = append(scopes, token.ScopeClaim{
				Role: ur.Role.Code,
				Type: st,
				ID:   si,
			})
		}
	}
	return dto.UserResponseData{
		ID:       user.ID,
		Email:    user.Email,
		FullName: user.FullName,
		Phone:    user.Phone,
		Roles:    roles,
		Scopes:   scopes,
		Status:   user.Status,
	}
}

func validateScopeID(scopeID string) bool {
	if len(scopeID) < 2 || len(scopeID) > 20 {
		return false
	}
	for _, ch := range scopeID {
		if !((ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
}

func (s *adminService) validateProdiScope(ctx context.Context, scopeType, scopeID string) (string, string, error) {
	cleanScopeType := strings.TrimSpace(scopeType)
	if cleanScopeType == "" {
		cleanScopeType = "prodi"
	}
	cleanScopeID := strings.ToUpper(strings.TrimSpace(scopeID))
	if !validateScopeID(cleanScopeID) {
		return "", "", fmt.Errorf("%w: role prodi wajib memiliki scope_id valid (2-20 karakter alfanumerik/underscore kapital)", ErrValidation)
	}

	if s.orgClient != nil {
		var orgResp any
		if err := s.orgClient.Get(ctx, "/internal/study-programs/"+cleanScopeID, nil, &orgResp); err != nil {
			var clientErr *internalclient.Error
			if errors.As(err, &clientErr) && clientErr.StatusCode == 404 {
				return "", "", fmt.Errorf("%w: program studi dengan kode '%s' tidak ditemukan", ErrValidation, cleanScopeID)
			}
			s.logger.Error("Gagal menghubungi organization service untuk validasi scope prodi", zap.Error(err), zap.String("scope_id", cleanScopeID))
			return "", "", fmt.Errorf("%w: layanan organization tidak dapat dihubungi untuk validasi scope prodi", ErrUpstreamUnavailable)
		}
	}

	return cleanScopeType, cleanScopeID, nil
}

func (s *adminService) validateDekanScope(ctx context.Context, scopeType, scopeID string) (string, string, error) {
	cleanScopeType := strings.TrimSpace(scopeType)
	if cleanScopeType == "" {
		cleanScopeType = "faculty"
	}
	cleanScopeID := strings.ToUpper(strings.TrimSpace(scopeID))
	if !validateScopeID(cleanScopeID) {
		return "", "", fmt.Errorf("%w: role dekan wajib memiliki scope_id valid (2-20 karakter alfanumerik/underscore kapital)", ErrValidation)
	}

	if s.orgClient != nil {
		var orgResp any
		if err := s.orgClient.Get(ctx, "/internal/faculties/"+cleanScopeID, nil, &orgResp); err != nil {
			var clientErr *internalclient.Error
			if errors.As(err, &clientErr) && clientErr.StatusCode == 404 {
				return "", "", fmt.Errorf("%w: fakultas dengan kode '%s' tidak ditemukan", ErrValidation, cleanScopeID)
			}
			s.logger.Warn("Gagal menghubungi organization service untuk validasi scope dekan", zap.Error(err), zap.String("scope_id", cleanScopeID))
		}
	}

	return cleanScopeType, cleanScopeID, nil
}

func (s *adminService) GetPermissions(ctx context.Context) ([]dto.PermissionResponseData, error) {
	perms, err := s.roleRepo.GetAllPermissions(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]dto.PermissionResponseData, len(perms))
	for i, p := range perms {
		res[i] = dto.PermissionResponseData{
			Code:        p.Code,
			Module:      p.Module,
			Description: p.Description,
		}
	}
	return res, nil
}

func (s *adminService) GetRolePermissions(ctx context.Context, roleCode string) (*dto.RolePermissionsResponseData, error) {
	role, err := s.roleRepo.FindByCode(ctx, roleCode)
	if err != nil || role == nil {
		return nil, ErrRoleNotFound
	}
	perms, err := s.roleRepo.GetRolePermissions(ctx, role.Code)
	if err != nil {
		return nil, err
	}
	return &dto.RolePermissionsResponseData{
		RoleCode:    roleCode,
		Permissions: perms,
	}, nil
}

func (s *adminService) UpdateRolePermissions(ctx context.Context, roleCode string, permissions []string, actorID uuid.UUID) (*dto.RolePermissionsResponseData, error) {
	role, err := s.roleRepo.FindByCode(ctx, roleCode)
	if err != nil || role == nil {
		return nil, ErrRoleNotFound
	}

	// PRD 1.3: tidak boleh mengurangi permission superadmin
	if roleCode == model.RoleSuperadmin {
		existing, _ := s.roleRepo.GetRolePermissions(ctx, role.Code)
		existingMap := make(map[string]bool)
		for _, p := range existing {
			existingMap[p] = true
		}
		for _, p := range permissions {
			delete(existingMap, p)
		}
		if len(existingMap) > 0 {
			return nil, fmt.Errorf("%w: tidak boleh mengurangi permission superadmin", ErrValidation)
		}
	}

	if err := s.roleRepo.UpdateRolePermissions(ctx, role.Code, permissions); err != nil {
		return nil, err
	}

	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		UserID:    &actorID,
		Action:    "admin.role_permissions.updated",
		Metadata:  fmt.Sprintf(`{"role_code":"%s","permissions_count":%d}`, roleCode, len(permissions)),
		CreatedAt: time.Now(),
	})

	return &dto.RolePermissionsResponseData{
		RoleCode:    roleCode,
		Permissions: permissions,
	}, nil
}
