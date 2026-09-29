package repository_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/admission-service/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestApplicationChangeModel(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()

	change := &model.ApplicationChange{
		ApplicationID: appID,
		UserID:        userID,
		ChangeType:    model.ChangeTypeCreated,
		Reason:        "Pendaftaran baru dibuat",
		FeeDifference: 0,
		Source:        model.ChangeSourceApplicant,
		ActorRole:     "calon_mahasiswa",
		CreatedAt:     time.Now(),
	}

	fromData := map[string]interface{}{"status": "draft"}
	toData := map[string]interface{}{"status": "awaiting_payment"}

	err := change.SetFromValues(fromData)
	assert.NoError(t, err)
	assert.Contains(t, change.FromValues, "draft")

	err = change.SetToValues(toData)
	assert.NoError(t, err)
	assert.Contains(t, change.ToValues, "awaiting_payment")

	assert.Equal(t, "application_changes", change.TableName())
}

func TestPathChangeRequestModel(t *testing.T) {
	appID := uuid.New()
	userID := uuid.New()
	choice1 := uuid.New()
	choice2 := uuid.New()

	req := &model.PathChangeRequest{
		ApplicationID: appID,
		UserID:        userID,
		FromPeriodID:  1,
		FromPathID:    1,
		FromChoice1ID: choice1,
		ToPeriodID:    1,
		ToPathID:      2,
		ToChoice1ID:   choice2,
		Reason:        "Pindah ke jalur prestasi karena ada sertifikat",
		FeeDifference: 50000,
	}

	assert.Equal(t, "path_change_requests", req.TableName())
	assert.Equal(t, model.PathChangePending, model.PathChangeStatus("pending"))
}
