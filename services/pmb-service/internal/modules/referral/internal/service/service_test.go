package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/model"
	"github.com/pkm-asia/pmb-backend/services/pmb-service/internal/modules/referral/internal/repository"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockReferralRepo struct {
	repository.ReferralRepository
	codes map[string]*model.ReferralCode
}

func (m *mockReferralRepo) GetCodeByCode(_ context.Context, code string) (*model.ReferralCode, error) {
	if c, ok := m.codes[code]; ok {
		return c, nil
	}
	return nil, nil
}

func TestReferralService_CheckCode(t *testing.T) {
	mockRepo := &mockReferralRepo{
		codes: map[string]*model.ReferralCode{
			"BKSMAN1": {
				ID:              uuid.New(),
				Code:            "BKSMAN1",
				InstitutionName: "SMAN 1 Tasikmalaya",
				Status:          "active",
				CreatedAt:       time.Now(),
			},
			"INACTIVE": {
				ID:              uuid.New(),
				Code:            "INACTIVE",
				InstitutionName: "Inactive School",
				Status:          "inactive",
			},
		},
	}

	logger := zap.NewNop()
	svc := NewReferralService(mockRepo, nil, logger)

	t.Run("Valid active code", func(t *testing.T) {
		res, err := svc.CheckCode(context.Background(), "BKSMAN1")
		assert.NoError(t, err)
		assert.True(t, res.Valid)
		assert.Equal(t, "SMAN 1 Tasikmalaya", res.InstitutionName)
	})

	t.Run("Inactive code returns invalid", func(t *testing.T) {
		res, err := svc.CheckCode(context.Background(), "INACTIVE")
		assert.NoError(t, err)
		assert.False(t, res.Valid)
	})

	t.Run("Not found code returns invalid", func(t *testing.T) {
		res, err := svc.CheckCode(context.Background(), "UNKNOWN")
		assert.NoError(t, err)
		assert.False(t, res.Valid)
	})
}

func TestMaskNIK(t *testing.T) {
	assert.Equal(t, "3204********0002", MaskNIK("3204010101060002"))
	assert.Equal(t, "3204********0002", MaskNIK("short"))
}
