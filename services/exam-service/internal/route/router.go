package route

import (
	"github.com/gin-gonic/gin"
	"github.com/pkm-asia/pmb-backend/pkg/gatewayauth"
	"github.com/pkm-asia/pmb-backend/pkg/internalauth"
	"github.com/pkm-asia/pmb-backend/services/exam-service/internal/controller"
)

type RouterConfig struct {
	GatewaySecret string
	InternalToken string
	IsDevelopment bool
}

func SetupRouter(
	cfg RouterConfig,
	healthCtrl *controller.HealthController,
	sessionCtrl *controller.SessionController,
	participantCtrl *controller.ParticipantController,
	studentCtrl *controller.StudentController,
	examinerCtrl *controller.ExaminerController,
	internalCtrl *controller.InternalController,
	cbtCtrl *controller.CBTController,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	// 1. Health check publik & sinkronisasi waktu server
	r.GET("/health", healthCtrl.HealthCheck)
	r.GET("/health/ready", healthCtrl.ReadinessCheck)
	r.GET("/time", cbtCtrl.GetServerTime)
	r.GET("/exam/time", cbtCtrl.GetServerTime)

	// 2. Internal endpoints (dilindungi X-Internal-Token)
	internalGroup := r.Group("/internal")
	internalGroup.Use(internalauth.Middleware(cfg.InternalToken))
	{
		internalGroup.GET("/participants/:application_id", internalCtrl.GetParticipantByApplicationID)
		internalGroup.GET("/sessions/:id", internalCtrl.GetSession)
		internalGroup.POST("/participants/remove", internalCtrl.RemoveParticipant)
	}

	// 3. Downstream endpoints dari API Gateway (dilindungi signature gateway)
	authGroup := r.Group("")
	authGroup.Use(gatewayauth.Middleware(cfg.GatewaySecret))
	{
		// Sesi & Kartu Ujian Calon Mahasiswa
		meGroup := authGroup.Group("/exam/me")
		{
			meGroup.GET("/schedule", studentCtrl.GetMySchedule)
			meGroup.GET("/card", studentCtrl.GetMyCard)
		}

		// CBT Peserta (mendukung prefix /exam/cbt/me maupun /cbt/me)
		registerCBTParticipantRoutes := func(prefix string) {
			cbtGroup := authGroup.Group(prefix)
			{
				cbtGroup.GET("/exams", cbtCtrl.GetMyCBTExams)
				cbtGroup.GET("/exams/:id", cbtCtrl.GetCBTExamDetail)
				cbtGroup.POST("/exams/:id/start", cbtCtrl.StartAttempt)
				cbtGroup.GET("/attempts/:id", cbtCtrl.GetAttemptQuestions)
				cbtGroup.PUT("/attempts/:id/answers/:question_id", cbtCtrl.SaveAnswer)
				cbtGroup.POST("/attempts/:id/events", cbtCtrl.RecordEvent)
				cbtGroup.POST("/attempts/:id/submit", cbtCtrl.SubmitAttempt)
				cbtGroup.GET("/attempts/:id/result", cbtCtrl.GetAttemptResult)
			}
		}
		registerCBTParticipantRoutes("/exam/cbt/me")
		registerCBTParticipantRoutes("/cbt/me")

		// Endpoint Dosen Penguji
		examinerGroup := authGroup.Group("/exam/examiner")
		examinerGroup.Use(gatewayauth.RequireRole("dosen", "admin_pmb", "superadmin"))
		{
			examinerGroup.GET("/sessions", examinerCtrl.GetMyAssignedSessions)
			examinerGroup.GET("/sessions/:id/participants", examinerCtrl.GetMySessionParticipants)
			examinerGroup.GET("/cbt/essays", cbtCtrl.ListUngradedEssays)
			examinerGroup.PUT("/cbt/answers/:id/grade", cbtCtrl.GradeEssay)
		}

		// Presensi Kehadiran (Dapat diakses oleh Dosen bertugas atau Admin PMB)
		authGroup.PATCH("/exam/admin/participants/:id/attendance",
			gatewayauth.RequireRole("admin_pmb", "superadmin", "dosen"),
			participantCtrl.UpdateAttendance,
		)

		// Endpoint Admin PMB & Superadmin
		adminGroup := authGroup.Group("/exam/admin")
		adminGroup.Use(gatewayauth.RequireRole("admin_pmb", "superadmin"))
		{
			adminGroup.POST("/sessions", sessionCtrl.CreateSession)
			adminGroup.GET("/sessions", sessionCtrl.ListSessions)
			adminGroup.GET("/sessions/:id", sessionCtrl.GetSession)
			adminGroup.PUT("/sessions/:id", sessionCtrl.UpdateSession)
			adminGroup.DELETE("/sessions/:id", sessionCtrl.DeleteSession)

			adminGroup.POST("/sessions/:id/examiners", sessionCtrl.AssignExaminer)
			adminGroup.DELETE("/sessions/:id/examiners/:user_id", sessionCtrl.RemoveExaminer)

			adminGroup.POST("/sessions/:id/participants", participantCtrl.AddParticipant)
			adminGroup.POST("/sessions/:id/auto-assign", participantCtrl.AutoAssignParticipants)
			adminGroup.GET("/sessions/:id/participants", participantCtrl.GetSessionParticipants)

			// Admin CBT
			adminGroup.POST("/cbt/banks", cbtCtrl.CreateQuestionBank)
			adminGroup.GET("/cbt/banks", cbtCtrl.ListQuestionBanks)
			adminGroup.POST("/cbt/questions", cbtCtrl.CreateQuestion)
			adminGroup.GET("/cbt/questions", cbtCtrl.ListQuestions)
			adminGroup.POST("/cbt/exams", cbtCtrl.CreateCBTExam)
			adminGroup.POST("/cbt/exams/:id/publish", cbtCtrl.PublishCBTExam)
			adminGroup.POST("/cbt/attempts/:id/reset", cbtCtrl.ResetAttempt)
			adminGroup.POST("/cbt/attempts/:id/extend", cbtCtrl.ExtendAttempt)
		}
	}

	return r
}
