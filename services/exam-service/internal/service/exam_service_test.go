package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/config"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"github.com/stretchr/testify/assert"
)

type mockSessionRepo struct {
	sessions  map[uuid.UUID]*model.ExamSession
	examiners map[uuid.UUID][]uuid.UUID
}

func newMockSessionRepo() *mockSessionRepo {
	return &mockSessionRepo{
		sessions:  make(map[uuid.UUID]*model.ExamSession),
		examiners: make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *mockSessionRepo) Create(ctx context.Context, session *model.ExamSession) error {
	if session.ID == uuid.Nil {
		session.ID = uuid.New()
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepo) Update(ctx context.Context, session *model.ExamSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.sessions, id)
	return nil
}

func (m *mockSessionRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.ExamSession, error) {
	s, ok := m.sessions[id]
	if !ok {
		return nil, nil
	}
	return s, nil
}

func (m *mockSessionRepo) List(ctx context.Context, periodID, pathID *uuid.UUID, sessionType *string, isActive *bool) ([]model.ExamSession, error) {
	var list []model.ExamSession
	for _, s := range m.sessions {
		list = append(list, *s)
	}
	return list, nil
}

func (m *mockSessionRepo) AddExaminer(ctx context.Context, examiner *model.SessionExaminer) error {
	m.examiners[examiner.SessionID] = append(m.examiners[examiner.SessionID], examiner.ExaminerUserID)
	return nil
}

func (m *mockSessionRepo) RemoveExaminer(ctx context.Context, sessionID, examinerUserID uuid.UUID) error {
	exList := m.examiners[sessionID]
	var updated []uuid.UUID
	for _, u := range exList {
		if u != examinerUserID {
			updated = append(updated, u)
		}
	}
	m.examiners[sessionID] = updated
	return nil
}

func (m *mockSessionRepo) GetExaminersBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.SessionExaminer, error) {
	var list []model.SessionExaminer
	for _, u := range m.examiners[sessionID] {
		list = append(list, model.SessionExaminer{
			SessionID:      sessionID,
			ExaminerUserID: u,
		})
	}
	return list, nil
}

func (m *mockSessionRepo) GetSessionsByExaminerID(ctx context.Context, examinerUserID uuid.UUID) ([]model.ExamSession, error) {
	var list []model.ExamSession
	for sessID, exList := range m.examiners {
		for _, u := range exList {
			if u == examinerUserID {
				if s, ok := m.sessions[sessID]; ok {
					list = append(list, *s)
				}
			}
		}
	}
	return list, nil
}

func (m *mockSessionRepo) IsExaminerAssigned(ctx context.Context, sessionID, examinerUserID uuid.UUID) (bool, error) {
	for _, u := range m.examiners[sessionID] {
		if u == examinerUserID {
			return true, nil
		}
	}
	return false, nil
}

type mockParticipantRepo struct {
	participants map[uuid.UUID]*model.Participant
	cards        map[uuid.UUID]*model.ExamCard
	seq          int64
}

func newMockParticipantRepo() *mockParticipantRepo {
	return &mockParticipantRepo{
		participants: make(map[uuid.UUID]*model.Participant),
		cards:        make(map[uuid.UUID]*model.ExamCard),
	}
}

func (m *mockParticipantRepo) AddParticipant(ctx context.Context, p *model.Participant) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	m.participants[p.ID] = p
	return nil
}

func (m *mockParticipantRepo) FindByID(ctx context.Context, id uuid.UUID) (*model.Participant, error) {
	p, ok := m.participants[id]
	if !ok {
		return nil, nil
	}
	return p, nil
}

func (m *mockParticipantRepo) FindByApplicationID(ctx context.Context, appID uuid.UUID) (*model.Participant, error) {
	for _, p := range m.participants {
		if p.ApplicationID == appID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockParticipantRepo) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Participant, error) {
	var list []model.Participant
	for _, p := range m.participants {
		if p.UserID == userID {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockParticipantRepo) FindBySessionAndUser(ctx context.Context, sessionID, userID uuid.UUID) (*model.Participant, error) {
	for _, p := range m.participants {
		if p.SessionID == sessionID && p.UserID == userID {
			return p, nil
		}
	}
	return nil, nil
}

func (m *mockParticipantRepo) FindBySessionID(ctx context.Context, sessionID uuid.UUID) ([]model.Participant, error) {
	var list []model.Participant
	for _, p := range m.participants {
		if p.SessionID == sessionID {
			list = append(list, *p)
		}
	}
	return list, nil
}

func (m *mockParticipantRepo) CountBySessionID(ctx context.Context, sessionID uuid.UUID) (int64, error) {
	var count int64
	for _, p := range m.participants {
		if p.SessionID == sessionID {
			count++
		}
	}
	return count, nil
}

func (m *mockParticipantRepo) UpdateAttendance(ctx context.Context, participantID uuid.UUID, attendance string) error {
	p, ok := m.participants[participantID]
	if !ok {
		return ErrParticipantNotFound
	}
	p.Attendance = &attendance
	return nil
}

func (m *mockParticipantRepo) GetNextParticipantSequence(ctx context.Context, year int) (int64, error) {
	m.seq++
	return m.seq, nil
}

func (m *mockParticipantRepo) SaveExamCard(ctx context.Context, card *model.ExamCard) error {
	if card.ID == uuid.Nil {
		card.ID = uuid.New()
	}
	m.cards[card.ParticipantID] = card
	return nil
}

func (m *mockParticipantRepo) FindCardByParticipantID(ctx context.Context, participantID uuid.UUID) (*model.ExamCard, error) {
	c, ok := m.cards[participantID]
	if !ok {
		return nil, nil
	}
	return c, nil
}

func TestExamService_CreateSession(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	partRepo := newMockParticipantRepo()
	cfg := &config.Config{ExamCardBucket: "pmb-exam-cards"}

	svc := NewExamService(cfg, sessionRepo, partRepo, nil, nil, nil, nil)
	ctx := context.Background()

	// 1. Sesi dengan waktu valid
	start := time.Now().Add(24 * time.Hour)
	end := start.Add(2 * time.Hour)
	sess, err := svc.CreateSession(ctx, &model.ExamSession{
		Title:    "Sesi Tulis 1",
		StartAt:  start,
		EndAt:    end,
		Capacity: 30,
	})
	assert.NoError(t, err)
	assert.NotNil(t, sess)
	assert.Equal(t, model.SessionTypeWritten, sess.Type)

	// 2. Sesi dengan waktu invalid (end <= start)
	_, err = svc.CreateSession(ctx, &model.ExamSession{
		Title:   "Sesi Salah",
		StartAt: end,
		EndAt:   start,
	})
	assert.ErrorIs(t, err, ErrInvalidSessionTime)
}

func TestExamService_AssignParticipantFlow(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	partRepo := newMockParticipantRepo()
	cfg := &config.Config{ExamCardBucket: "pmb-exam-cards"}

	svc := NewExamService(cfg, sessionRepo, partRepo, nil, nil, nil, nil)
	ctx := context.Background()

	// Buat sesi dengan kapasitas 2
	sessionID := uuid.New()
	sess := &model.ExamSession{
		ID:       sessionID,
		Title:    "Ujian Masuk Kampus",
		Type:     model.SessionTypeWritten,
		StartAt:  time.Now().Add(24 * time.Hour),
		EndAt:    time.Now().Add(26 * time.Hour),
		Capacity: 2,
	}
	_ = sessionRepo.Create(ctx, sess)

	// 1. Assign peserta 1 berhasil
	appID1 := uuid.New()
	p1, err := svc.AssignParticipantManual(ctx, sessionID, appID1)
	assert.NoError(t, err)
	assert.NotNil(t, p1)
	assert.Contains(t, p1.ParticipantNumber, "EXAM-")
	assert.NotNil(t, p1.ExamCard)

	// 2. Assign peserta duplikat ditolak
	_, err = svc.AssignParticipantManual(ctx, sessionID, appID1)
	assert.ErrorIs(t, err, ErrDuplicateParticipant)

	// 3. Assign peserta 2 berhasil (kapasitas menjadi penuh: 2/2)
	appID2 := uuid.New()
	p2, err := svc.AssignParticipantManual(ctx, sessionID, appID2)
	assert.NoError(t, err)
	assert.NotNil(t, p2)

	// 4. Assign peserta 3 ditolak karena kapasitas penuh
	appID3 := uuid.New()
	_, err = svc.AssignParticipantManual(ctx, sessionID, appID3)
	assert.ErrorIs(t, err, ErrSessionCapacityExceeded)
}

func TestExamService_AttendanceAndExaminerIsolation(t *testing.T) {
	sessionRepo := newMockSessionRepo()
	partRepo := newMockParticipantRepo()
	cfg := &config.Config{ExamCardBucket: "pmb-exam-cards"}

	svc := NewExamService(cfg, sessionRepo, partRepo, nil, nil, nil, nil)
	ctx := context.Background()

	sessID := uuid.New()
	_ = sessionRepo.Create(ctx, &model.ExamSession{
		ID:       sessID,
		Title:    "Sesi Wawancara",
		Type:     model.SessionTypeInterview,
		StartAt:  time.Now().Add(24 * time.Hour),
		EndAt:    time.Now().Add(26 * time.Hour),
		Capacity: 10,
	})

	dosenA := uuid.New()
	dosenB := uuid.New()

	// Tugaskan dosen A ke sesi
	_, err := svc.AssignExaminer(ctx, sessID, dosenA)
	assert.NoError(t, err)

	// Tambah peserta
	p, err := svc.AssignParticipantManual(ctx, sessID, uuid.New())
	assert.NoError(t, err)

	// 1. Dosen B mencoba update presensi peserta sesi dosen A -> Ditolak FORBIDDEN
	err = svc.UpdateAttendance(ctx, p.ID, "present", dosenB, "dosen")
	assert.ErrorIs(t, err, ErrForbiddenExaminer)

	// 2. Dosen A update presensi peserta -> Berhasil
	err = svc.UpdateAttendance(ctx, p.ID, "present", dosenA, "dosen")
	assert.NoError(t, err)

	updatedP, _ := partRepo.FindByID(ctx, p.ID)
	assert.NotNil(t, updatedP.Attendance)
	assert.Equal(t, "present", *updatedP.Attendance)

	// 3. Admin PMB update presensi peserta -> Berhasil
	err = svc.UpdateAttendance(ctx, p.ID, "absent", uuid.New(), "admin_pmb")
	assert.NoError(t, err)
	assert.Equal(t, "absent", *updatedP.Attendance)

	// 4. Update presensi dengan nilai invalid -> Ditolak
	err = svc.UpdateAttendance(ctx, p.ID, "bolos", dosenA, "dosen")
	assert.ErrorIs(t, err, ErrInvalidAttendance)

	// 5. Cek isolasi sesi: Dosen A melihat sesi, Dosen B tidak melihat
	sessA, err := svc.GetExaminerSessions(ctx, dosenA)
	assert.NoError(t, err)
	assert.Len(t, sessA, 1)

	sessB, err := svc.GetExaminerSessions(ctx, dosenB)
	assert.NoError(t, err)
	assert.Len(t, sessB, 0)
}
