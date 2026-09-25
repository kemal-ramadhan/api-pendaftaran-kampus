package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/response"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/model"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/service"
)

type CBTController struct {
	svc service.CBTService
}

func NewCBTController(svc service.CBTService) *CBTController {
	return &CBTController{svc: svc}
}

// GetServerTime mengembalikan waktu server untuk sinkronisasi timer frontend
func (ctrl *CBTController) GetServerTime(c *gin.Context) {
	response.Success(c, http.StatusOK, "Waktu server berhasil diambil", gin.H{
		"server_time": time.Now(),
		"timestamp":   time.Now().Unix(),
	})
}

// GetMyCBTExams mengembalikan daftar ujian CBT untuk peserta
func (ctrl *CBTController) GetMyCBTExams(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	exams, err := ctrl.svc.GetMyCBTExams(c.Request.Context(), userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil daftar ujian CBT: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Daftar ujian CBT berhasil diambil", exams)
}

// GetCBTExamDetail mengembalikan detail halaman CBT untuk peserta
func (ctrl *CBTController) GetCBTExamDetail(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	examID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID ujian tidak valid")
		return
	}

	detail, err := ctrl.svc.GetCBTExamDetail(c.Request.Context(), examID, userID)
	if err != nil {
		if errors.Is(err, service.ErrNotRegisteredForSession) {
			response.Forbidden(c, response.ErrCodeForbidden, err.Error())
			return
		}
		response.InternalServerError(c, "Gagal mengambil detail ujian CBT: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Detail ujian CBT berhasil diambil", detail)
}

// StartAttempt memulai pengerjaan ujian
func (ctrl *CBTController) StartAttempt(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	examID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID ujian tidak valid")
		return
	}

	var reqBody struct {
		AccessToken string `json:"access_token"`
	}
	_ = c.ShouldBindJSON(&reqBody)

	attempt, err := ctrl.svc.StartAttempt(c.Request.Context(), examID, userID, reqBody.AccessToken)
	if err != nil {
		if errors.Is(err, service.ErrExamNotStarted) {
			response.BadRequest(c, "EXAM_NOT_STARTED", err.Error())
			return
		}
		if errors.Is(err, service.ErrExamEnded) {
			response.BadRequest(c, "EXAM_ENDED", err.Error())
			return
		}
		if errors.Is(err, service.ErrExamAccessTokenInvalid) {
			response.BadRequest(c, "EXAM_ACCESS_TOKEN_INVALID", err.Error())
			return
		}
		if errors.Is(err, service.ErrAttemptAlreadySubmitted) {
			response.Conflict(c, "ATTEMPT_ALREADY_SUBMITTED", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal memulai sesi pengerjaan ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Sesi pengerjaan ujian dimulai", attempt)
}

// GetAttemptQuestions mengambil paket soal ujian untuk peserta (tanpa kunci jawaban!)
func (ctrl *CBTController) GetAttemptQuestions(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	attemptID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID attempt tidak valid")
		return
	}

	pkg, err := ctrl.svc.GetAttemptQuestions(c.Request.Context(), attemptID, userID)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak memiliki akses ke sesi ujian ini")
			return
		}
		response.InternalServerError(c, "Gagal mengambil lembar soal: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Lembar soal berhasil diambil", pkg)
}

// SaveAnswer menyimpan jawaban / flag satu soal (autosave idempoten)
func (ctrl *CBTController) SaveAnswer(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	attemptIDStr := c.Param("id")
	attemptID, err := uuid.Parse(attemptIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID attempt tidak valid")
		return
	}

	qIDStr := c.Param("question_id")
	questionID, err := uuid.Parse(qIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID question tidak valid")
		return
	}

	var reqBody struct {
		SelectedOptionIDs []uuid.UUID `json:"selected_option_ids"`
		EssayText         *string     `json:"essay_text"`
		IsFlagged         bool        `json:"is_flagged"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Payload jawaban tidak valid: "+err.Error())
		return
	}

	err = ctrl.svc.SaveAnswer(c.Request.Context(), attemptID, questionID, userID, reqBody.SelectedOptionIDs, reqBody.EssayText, reqBody.IsFlagged)
	if err != nil {
		if errors.Is(err, service.ErrAttemptTimeUp) {
			response.Conflict(c, "ATTEMPT_TIME_UP", err.Error())
			return
		}
		if errors.Is(err, service.ErrAttemptAlreadySubmitted) {
			response.Conflict(c, "ATTEMPT_ALREADY_SUBMITTED", err.Error())
			return
		}
		response.InternalServerError(c, "Gagal menyimpan jawaban: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Jawaban berhasil disimpan", gin.H{"saved": true})
}

// RecordEvent mencatat event aktivitas ujian (mis. fokus hilang, fullscreen exit)
func (ctrl *CBTController) RecordEvent(c *gin.Context) {
	attemptIDStr := c.Param("id")
	attemptID, err := uuid.Parse(attemptIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID attempt tidak valid")
		return
	}

	var reqBody struct {
		Type       string     `json:"type"`
		ClientTime *time.Time `json:"client_time"`
		Metadata   string     `json:"metadata"`
	}
	_ = c.ShouldBindJSON(&reqBody)

	_ = ctrl.svc.RecordEvent(c.Request.Context(), attemptID, reqBody.Type, reqBody.ClientTime, reqBody.Metadata)
	response.Success(c, http.StatusOK, "Event dicatat", gin.H{"recorded": true})
}

// SubmitAttempt mengumpulkan ujian
func (ctrl *CBTController) SubmitAttempt(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	attemptID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID attempt tidak valid")
		return
	}

	attempt, err := ctrl.svc.SubmitAttempt(c.Request.Context(), attemptID, userID)
	if err != nil {
		if errors.Is(err, service.ErrForbiddenAccess) {
			response.Forbidden(c, response.ErrCodeForbidden, "Anda tidak memiliki akses ke sesi ujian ini")
			return
		}
		response.InternalServerError(c, "Gagal mengumpulkan ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Ujian berhasil dikumpulkan", gin.H{
		"attempt_id":      attempt.ID,
		"status":          attempt.Status,
		"submitted_at":    attempt.SubmittedAt,
		"objective_score": attempt.ObjectiveScore,
	})
}

// GetAttemptResult mengambil hasil ujian jika diizinkan
func (ctrl *CBTController) GetAttemptResult(c *gin.Context) {
	userID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}

	idStr := c.Param("id")
	attemptID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID attempt tidak valid")
		return
	}

	res, err := ctrl.svc.GetAttemptResult(c.Request.Context(), attemptID, userID)
	if err != nil {
		response.InternalServerError(c, "Gagal mengambil hasil ujian: "+err.Error())
		return
	}

	response.Success(c, http.StatusOK, "Hasil ujian berhasil diambil", res)
}

// Admin endpoints
func (ctrl *CBTController) CreateQuestionBank(c *gin.Context) {
	var bank model.QuestionBank
	if err := c.ShouldBindJSON(&bank); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", err.Error())
		return
	}
	res, err := ctrl.svc.CreateQuestionBank(c.Request.Context(), &bank)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Bank soal berhasil dibuat", res)
}

func (ctrl *CBTController) ListQuestionBanks(c *gin.Context) {
	banks, err := ctrl.svc.ListQuestionBanks(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar bank soal", banks)
}

func (ctrl *CBTController) CreateQuestion(c *gin.Context) {
	var q model.Question
	if err := c.ShouldBindJSON(&q); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", err.Error())
		return
	}
	res, err := ctrl.svc.CreateQuestion(c.Request.Context(), &q)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Soal berhasil dibuat", res)
}

func (ctrl *CBTController) ListQuestions(c *gin.Context) {
	bankIDStr := c.Query("bank_id")
	bankID, err := uuid.Parse(bankIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_BANK_ID", "Format ID bank soal tidak valid")
		return
	}
	qs, err := ctrl.svc.ListQuestions(c.Request.Context(), bankID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar soal", qs)
}

func (ctrl *CBTController) CreateCBTExam(c *gin.Context) {
	var input struct {
		model.CBTExam
		AccessToken string `json:"access_token"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", err.Error())
		return
	}
	res, err := ctrl.svc.CreateCBTExam(c.Request.Context(), &input.CBTExam, input.AccessToken)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusCreated, "Ujian CBT berhasil dibuat", res)
}

func (ctrl *CBTController) PublishCBTExam(c *gin.Context) {
	idStr := c.Param("id")
	examID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID ujian tidak valid")
		return
	}
	err = ctrl.svc.PublishCBTExam(c.Request.Context(), examID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Ujian CBT dipublikasikan", gin.H{"published": true})
}

func (ctrl *CBTController) ResetAttempt(c *gin.Context) {
	idStr := c.Param("id")
	attemptID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID attempt tidak valid")
		return
	}
	var reqBody struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Alasan reset wajib diisi")
		return
	}
	err = ctrl.svc.ResetAttempt(c.Request.Context(), attemptID, reqBody.Reason)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Attempt berhasil direset", gin.H{"reset": true})
}

func (ctrl *CBTController) ExtendAttempt(c *gin.Context) {
	idStr := c.Param("id")
	attemptID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID attempt tidak valid")
		return
	}
	var reqBody struct {
		Minutes int    `json:"minutes" binding:"required"`
		Reason  string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Menit tambahan dan alasan wajib diisi")
		return
	}
	err = ctrl.svc.ExtendAttempt(c.Request.Context(), attemptID, reqBody.Minutes, reqBody.Reason)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Waktu pengerjaan berhasil ditambah", gin.H{"extended": true})
}

// Examiner endpoints
func (ctrl *CBTController) ListUngradedEssays(c *gin.Context) {
	examinerID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}
	examIDStr := c.Query("exam_id")
	examID, err := uuid.Parse(examIDStr)
	if err != nil {
		response.BadRequest(c, "INVALID_EXAM_ID", "Format ID ujian tidak valid")
		return
	}
	essays, err := ctrl.svc.ListUngradedEssays(c.Request.Context(), examID, examinerID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Daftar esai belum dinilai", essays)
}

func (ctrl *CBTController) GradeEssay(c *gin.Context) {
	examinerID, err := gatewayauth.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, response.ErrCodeTokenMissing, "Identitas pengguna tidak ditemukan")
		return
	}
	idStr := c.Param("id")
	answerID, err := uuid.Parse(idStr)
	if err != nil {
		response.BadRequest(c, "INVALID_ID", "Format ID jawaban tidak valid")
		return
	}
	var reqBody struct {
		Score float64 `json:"score"`
	}
	if err := c.ShouldBindJSON(&reqBody); err != nil {
		response.BadRequest(c, "VALIDATION_ERROR", "Nilai esai tidak valid")
		return
	}
	err = ctrl.svc.GradeEssay(c.Request.Context(), answerID, reqBody.Score, examinerID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}
	response.Success(c, http.StatusOK, "Nilai esai berhasil disimpan", gin.H{"graded": true})
}
