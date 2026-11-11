// Package routes registers all application HTTP routes onto the Gin engine.
//
// In Node.js Express, you split routes into separate files:
//   app.use('/api/auth', authRoutes)
//   app.use('/api/exams', examRoutes)
//   app.use('/api/documents', documentRoutes)
//   app.use('/api/ai', aiRoutes)
//
// In Go with Gin, we do the exact same thing using Router Groups (r.Group).
// The structure below maps 1:1 to your Express route tree.
//
// Clean Architecture Concept:
//   Routes only wire URLs to Handler methods.
//   They contain NO business logic, NO database queries, NO response formatting.
//   All dependency injection happens at the top (main.go), passed down here.
package routes

import (
	"github.com/aiStudySaathi/backend/internal/config"
	"github.com/aiStudySaathi/backend/internal/handler"
	"github.com/aiStudySaathi/backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// Setup registers all application routes onto the Gin engine.
// All handlers and middleware are passed in as arguments — no globals.
func Setup(
	r *gin.Engine,
	cfg *config.Config,
	authHandler *handler.AuthHandler,
	examHandler *handler.ExamHandler,
	docHandler *handler.DocumentHandler,
	aiHandler *handler.AIHandler,
	quizHandler *handler.QuizHandler,
	masteryHandler *handler.MasteryHandler,
	pyqHandler *handler.PyqHandler,
	mappingHandler *handler.MappingHandler,
	cardHandler *handler.FlashcardHandler,
	planHandler *handler.StudyPlanHandler,
	assistantHandler *handler.AssistantHandler,
	tutoringHandler *handler.TutoringHandler,
) {
	// /api is the common prefix — identical to your Node.js app.use('/api/...')
	api := r.Group("/api")

	// --- Health Check ---
	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "ai-study-saathi-go"})
	})

	// --- Auth Routes (public) ---
	auth := api.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.GET("/me", middleware.Protect(cfg.JWTSecret), authHandler.GetMe)
	}

	// --- Exam Routes (protected) ---
	exams := api.Group("/exams", middleware.Protect(cfg.JWTSecret))
	{
		exams.POST("/syllabus/extract", examHandler.ExtractSyllabus)
		exams.POST("", examHandler.Create)
		exams.GET("", examHandler.List)
		exams.GET("/:id", examHandler.GetByID)
		exams.PATCH("/:id", examHandler.Update)
		exams.DELETE("/:id", examHandler.Delete)
		exams.PUT("/:id/syllabus", examHandler.ReplaceSyllabus)
		exams.GET("/:id/topics/:topicId/resources", mappingHandler.GetTopicResources)
		exams.GET("/:id/topics/:topicId/quizzes", quizHandler.ListTopicQuizzes)
		exams.POST("/:id/topics/:topicId/quizzes", quizHandler.GenerateTopicQuiz)
		exams.GET("/:id/mastery", masteryHandler.GetExamMastery)
		exams.GET("/:id/topics/:topicId/mastery", masteryHandler.GetTopicMastery)
		exams.GET("/:id/topics/:topicId/pyq-analysis", pyqHandler.GetTopicPyqAnalysis)
		exams.GET("/:id/topics/:topicId/flashcards", cardHandler.ListTopicFlashcards)
		exams.POST("/:id/topics/:topicId/flashcards", cardHandler.GenerateTopicFlashcards)
		exams.GET("/:id/flashcards/due", cardHandler.GetDueFlashcards)
		exams.GET("/:id/study-plan", planHandler.GetStudyPlan)
		exams.POST("/:id/study-plan", planHandler.GenerateStudyPlan)
		exams.GET("/:id/assistant", assistantHandler.GetAssistantSession)
		exams.POST("/:id/assistant/message", assistantHandler.SendAssistantMessage)
	}

	// --- Quiz Routes (protected) ---
	quizzes := api.Group("/quizzes", middleware.Protect(cfg.JWTSecret))
	{
		quizzes.GET("/:quizId", quizHandler.GetQuiz)
		quizzes.POST("/:quizId/submit", quizHandler.SubmitAttempt)
		quizzes.GET("/:quizId/attempts", quizHandler.ListQuizAttempts)
		quizzes.GET("/:quizId/attempts/:attemptId", quizHandler.GetQuizAttempt)
	}

	// --- Document Routes (protected) ---
	docs := api.Group("/documents", middleware.Protect(cfg.JWTSecret))
	{
		docs.POST("", docHandler.Upload)
		docs.GET("", docHandler.List)
		docs.GET("/:id", docHandler.GetByID)
		docs.PATCH("/:id", docHandler.Update)
		docs.DELETE("/:id", docHandler.Delete)
		docs.GET("/:id/status", docHandler.GetStatus)
		docs.POST("/:id/retry", docHandler.Retry)
		docs.GET("/:id/mappings", mappingHandler.GetDocumentMappings)
		docs.POST("/:id/map-topics", mappingHandler.MapDocumentTopics)
		docs.GET("/:id/questions", pyqHandler.GetDocumentQuestions)
		docs.POST("/:id/analyze-pyq", pyqHandler.AnalyzePyqDocument)
	}

	// --- Topic Mapping Review Routes (protected) ---
	mappings := api.Group("/mappings", middleware.Protect(cfg.JWTSecret))
	{
		mappings.PATCH("/:mappingId", mappingHandler.UpdateMappingStatus)
	}

	// --- PYQ Question Routes (protected) ---
	pyqQuestions := api.Group("/pyq-questions", middleware.Protect(cfg.JWTSecret))
	{
		pyqQuestions.PATCH("/:questionId", pyqHandler.UpdateQuestionStatus)
	}

	// --- Flashcard Review Routes (protected) ---
	flashcards := api.Group("/flashcards", middleware.Protect(cfg.JWTSecret))
	{
		flashcards.POST("/:id/review", cardHandler.ReviewFlashcard)
	}

	// --- Study Plan Session Routes (protected) ---
	sessions := api.Group("/study-plan-sessions", middleware.Protect(cfg.JWTSecret))
	{
		sessions.PATCH("/:sessionId", planHandler.UpdateSessionCompletion)
	}

	// --- AI / RAG Routes (protected) ---
	aiGroup := api.Group("/ai", middleware.Protect(cfg.JWTSecret))
	{
		aiGroup.POST("/ask", aiHandler.AskQuestion)
	}

	// --- Socratic Tutoring Routes (protected) ---
	tutoring := api.Group("/tutoring", middleware.Protect(cfg.JWTSecret))
	{
		tutoring.POST("/session/start", tutoringHandler.StartSession)
		tutoring.POST("/session/:id/respond", tutoringHandler.RespondToTutor)
		tutoring.GET("/session/:id", tutoringHandler.GetSession)
	}
}
