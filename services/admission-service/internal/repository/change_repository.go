package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"gorm.io/gorm"
)

type ChangeRepository interface {
	// Hanya INSERT (Append-Only)
	CreateChange(ctx context.Context, tx *gorm.DB, change *model.ApplicationChange) error
	GetChangesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error)
	GetChangesByApplicationID(ctx context.Context, appID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error)

	// Path Change Requests
	CreatePathChangeRequest(ctx context.Context, tx *gorm.DB, req *model.PathChangeRequest) error
	GetPathChangeRequestByID(ctx context.Context, id uuid.UUID) (*model.PathChangeRequest, error)
	GetPendingPathChangeRequest(ctx context.Context, appID uuid.UUID) (*model.PathChangeRequest, error)
	UpdatePathChangeRequest(ctx context.Context, tx *gorm.DB, req *model.PathChangeRequest) error
	ListPathChangeRequests(ctx context.Context, status string, limit, offset int) ([]model.PathChangeRequest, int64, error)

	// Inisialisasi DB trigger & backfill
	InitApplicationChangesTriggersAndBackfill(db *gorm.DB) error
}

type changeRepository struct {
	db *gorm.DB
}

func NewChangeRepository(db *gorm.DB) ChangeRepository {
	return &changeRepository{db: db}
}

// CreateChange hanya melakukan INSERT ke tabel application_changes
func (r *changeRepository) CreateChange(ctx context.Context, tx *gorm.DB, change *model.ApplicationChange) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(change).Error
}

func (r *changeRepository) GetChangesByUserID(ctx context.Context, userID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error) {
	var changes []model.ApplicationChange
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ApplicationChange{}).Where("user_id = ?", userID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&changes).Error
	return changes, total, err
}

func (r *changeRepository) GetChangesByApplicationID(ctx context.Context, appID uuid.UUID, limit, offset int) ([]model.ApplicationChange, int64, error) {
	var changes []model.ApplicationChange
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ApplicationChange{}).Where("application_id = ?", appID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&changes).Error
	return changes, total, err
}

func (r *changeRepository) CreatePathChangeRequest(ctx context.Context, tx *gorm.DB, req *model.PathChangeRequest) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Create(req).Error
}

func (r *changeRepository) GetPathChangeRequestByID(ctx context.Context, id uuid.UUID) (*model.PathChangeRequest, error) {
	var req model.PathChangeRequest
	err := r.db.WithContext(ctx).
		Preload("Application").
		Preload("FromPath").
		Preload("ToPath").
		Preload("FromPeriod").
		Preload("ToPeriod").
		First(&req, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *changeRepository) GetPendingPathChangeRequest(ctx context.Context, appID uuid.UUID) (*model.PathChangeRequest, error) {
	var req model.PathChangeRequest
	err := r.db.WithContext(ctx).
		Where("application_id = ? AND status = ?", appID, model.PathChangePending).
		First(&req).Error
	if err != nil {
		return nil, err
	}
	return &req, nil
}

func (r *changeRepository) UpdatePathChangeRequest(ctx context.Context, tx *gorm.DB, req *model.PathChangeRequest) error {
	db := r.db
	if tx != nil {
		db = tx
	}
	return db.WithContext(ctx).Save(req).Error
}

func (r *changeRepository) ListPathChangeRequests(ctx context.Context, status string, limit, offset int) ([]model.PathChangeRequest, int64, error) {
	var list []model.PathChangeRequest
	var total int64

	query := r.db.WithContext(ctx).Model(&model.PathChangeRequest{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Preload("Application").
		Preload("FromPath").
		Preload("ToPath").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&list).Error
	return list, total, err
}

// InitApplicationChangesTriggersAndBackfill memasang trigger DB PostgreSQL untuk menjamin immutabilitas dan backfill
func (r *changeRepository) InitApplicationChangesTriggersAndBackfill(db *gorm.DB) error {
	// 1. Trigger PostgreSQL untuk menolak UPDATE dan DELETE
	triggerSQL := `
	DO $$
	BEGIN
		IF EXISTS (SELECT 1 FROM pg_tables WHERE tablename = 'application_changes') THEN
			CREATE OR REPLACE FUNCTION prevent_application_changes_modification()
			RETURNS TRIGGER AS $trg$
			BEGIN
				RAISE EXCEPTION 'application_changes table is append-only. UPDATE and DELETE are prohibited.';
			END;
			$trg$ LANGUAGE plpgsql;

			DROP TRIGGER IF EXISTS trg_prevent_application_changes_update ON application_changes;
			CREATE TRIGGER trg_prevent_application_changes_update
			BEFORE UPDATE ON application_changes
			FOR EACH ROW EXECUTE FUNCTION prevent_application_changes_modification();

			DROP TRIGGER IF EXISTS trg_prevent_application_changes_delete ON application_changes;
			CREATE TRIGGER trg_prevent_application_changes_delete
			BEFORE DELETE ON application_changes
			FOR EACH ROW EXECUTE FUNCTION prevent_application_changes_modification();
		END IF;
	END $$;
	`
	if err := db.Exec(triggerSQL).Error; err != nil {
		// Log warning jika di driver non-postgres (misal saat mock/sqlite test)
		fmt.Printf("[WARN] Tidak dapat memasang trigger PostgreSQL pada application_changes: %v\n", err)
	}

	// 2. Backfill dari application_status_logs & applications
	backfillCreatedSQL := `
	INSERT INTO application_changes (
		id, application_id, user_id, change_type, from_values, to_values,
		reason, fee_difference, actor_id, actor_role, source, created_at
	)
	SELECT 
		COALESCE(uuid_generate_v4(), a.id),
		a.id,
		a.user_id,
		'created',
		'{}'::jsonb,
		jsonb_build_object(
			'period_id', a.period_id,
			'path_id', a.admission_path_id,
			'choice_1_id', a.choice_1_id,
			'choice_2_id', a.choice_2_id
		),
		'Pendaftaran baru dibuat',
		0,
		a.user_id,
		'calon_mahasiswa',
		'applicant',
		a.created_at
	FROM applications a
	WHERE NOT EXISTS (
		SELECT 1 FROM application_changes ac 
		WHERE ac.application_id = a.id AND ac.change_type = 'created'
	);
	`
	_ = db.Exec(backfillCreatedSQL)

	backfillCancelledSQL := `
	INSERT INTO application_changes (
		id, application_id, user_id, change_type, from_values, to_values,
		reason, fee_difference, actor_id, actor_role, source, created_at
	)
	SELECT 
		COALESCE(uuid_generate_v4(), l.id),
		l.application_id,
		a.user_id,
		'cancelled',
		'{}'::jsonb,
		'{}'::jsonb,
		COALESCE(l.notes, 'Pendaftaran dibatalkan oleh calon mahasiswa'),
		0,
		COALESCE(l.changed_by, a.user_id),
		'calon_mahasiswa',
		'applicant',
		l.created_at
	FROM application_status_logs l
	JOIN applications a ON a.id = l.application_id
	WHERE l.to_status = 'cancelled'
	AND NOT EXISTS (
		SELECT 1 FROM application_changes ac 
		WHERE ac.application_id = l.application_id AND ac.change_type = 'cancelled'
	);
	`
	_ = db.Exec(backfillCancelledSQL)

	return nil
}
