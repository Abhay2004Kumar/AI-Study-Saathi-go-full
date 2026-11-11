// Package repository contains all database access code.
//
// In Node.js with Prisma, your DB calls were inline in services:
//   prisma.user.findUnique({ where: { email } })
//
// In Go without an ORM, we write SQL explicitly in a repository layer.
// The repository pattern separates "how to talk to the database" from
// "what the business logic needs to happen". This makes testing easier
// (mock the repository), and it centralises all SQL in one place.
//
// The repository layer:
//   1. Accepts a pgxpool.Pool (the connection pool)
//   2. Runs SQL queries via pgx
//   3. Scans rows into model structs
//   4. Returns typed values + errors (never panics)
package repository

import (
	"context"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
)

// UserRepository defines all database operations for the User entity.
// By depending on an interface instead of the concrete implementation,
// tests can pass in a mock without needing a real database.
//
// Think of this as the "data access layer" contract — equivalent to
// Prisma's auto-generated methods, but explicit and hand-written.
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	FindByID(ctx context.Context, id string) (*model.User, error)
	Create(ctx context.Context, user *model.User) error
}

// ExamRepository defines all database operations for the Exam entity.
type ExamRepository interface {
	Create(ctx context.Context, exam *model.Exam, subjects []model.Subject) (*model.Exam, error)
	FindByID(ctx context.Context, id, userID string) (*model.Exam, error)
	FindAllByUser(ctx context.Context, userID string) ([]model.Exam, error)
	Update(ctx context.Context, id, userID string, updates map[string]interface{}) (*model.Exam, error)
	Delete(ctx context.Context, id, userID string) error
	ReplaceSyllabus(ctx context.Context, examID, userID string, subjects []model.Subject) (*model.Exam, error)
}

// DocumentRepository defines all database operations for Documents.
type DocumentRepository interface {
	Create(ctx context.Context, doc *model.Document) error
	FindByID(ctx context.Context, id string) (*model.Document, error)
	FindAllByUser(ctx context.Context, userID string, examID *string) ([]model.Document, error)
	FindByHash(ctx context.Context, userID, hash string) (*model.Document, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdatePageCount(ctx context.Context, id string, count int) error
	Update(ctx context.Context, id string, updates map[string]interface{}) (*model.Document, error)
	Delete(ctx context.Context, id string) error
}

// DocumentChunkRepository handles chunk storage and pgvector similarity search.
type DocumentChunkRepository interface {
	// InsertWithEmbedding stores a chunk and its vector using raw SQL (pgvector requires it)
	InsertWithEmbedding(ctx context.Context, chunk *model.DocumentChunk, embedding []float32) error
	// InsertWithoutEmbedding stores a chunk with no vector (when embedding fails)
	InsertWithoutEmbedding(ctx context.Context, chunk *model.DocumentChunk) error
	// SimilaritySearch performs cosine distance search via pgvector
	SimilaritySearch(ctx context.Context, query SimilaritySearchParams) ([]SimilarityResult, error)
	// FindByDocumentID returns all chunks for a document (for topic mapping)
	FindByDocumentID(ctx context.Context, documentID string) ([]model.DocumentChunk, error)
	// DeleteByDocumentID removes all chunks when a document is deleted
	DeleteByDocumentID(ctx context.Context, documentID string) error
}

// SimilaritySearchParams encapsulates all parameters for a pgvector search.
// Using a struct instead of many function arguments is idiomatic Go.
type SimilaritySearchParams struct {
	UserID      string
	QueryVector []float32
	TopK        int
	ExamID      *string   // nil = no filter
	Category    *string   // nil = no filter
	DocumentIDs []string  // empty = no filter
}

// SimilarityResult is a single row from a similarity search.
type SimilarityResult struct {
	Content    string
	Metadata   []byte  // JSONB
	DocumentID string
	Title      string
	ExamID     *string
	Category   string
	Distance   float64 // cosine distance (lower = more similar)
}

// QuizRepository handles all quiz-related database operations.
type QuizRepository interface {
	Create(ctx context.Context, quiz *model.Quiz, questions []model.QuizQuestion) (*model.Quiz, error)
	FindByID(ctx context.Context, id, userID string) (*model.Quiz, error)
	FindByTopic(ctx context.Context, topicID, userID string) ([]model.Quiz, error)
	CreateAttempt(ctx context.Context, attempt *model.QuizAttempt, answers []model.QuizAttemptAnswer) (*model.QuizAttempt, error)
	FindAttemptByID(ctx context.Context, attemptID, quizID, userID string) (*model.QuizAttempt, error)
	FindAttemptsByQuiz(ctx context.Context, quizID, userID string) ([]model.QuizAttempt, error)
}

// FlashcardRepository handles all flashcard-related database operations.
type FlashcardRepository interface {
	CreateMany(ctx context.Context, cards []model.Flashcard) error
	FindByTopic(ctx context.Context, topicID, userID string) ([]model.Flashcard, error)
	FindDue(ctx context.Context, userID, examID string, limit int) ([]model.Flashcard, error)
	FindByID(ctx context.Context, id, userID string) (*model.Flashcard, error)
	UpdateReview(ctx context.Context, id string, easeFactor float64, intervalDays, repetitions int, dueAt, lastReviewedAt time.Time) (*model.Flashcard, error)
	FindNormalizedFrontsByTopic(ctx context.Context, topicID, userID string) ([]string, error)
}

// AssistantSessionRepository handles assistant conversation persistence.
type AssistantSessionRepository interface {
	FindByExamAndUser(ctx context.Context, examID, userID string) (*model.AssistantSession, error)
	Upsert(ctx context.Context, examID, userID string, messages []byte) (*model.AssistantSession, error)
}

// StudyPlanRepository handles study plan operations.
type StudyPlanRepository interface {
	Upsert(ctx context.Context, plan *model.StudyPlan, sessions []model.StudyPlanSession) (*model.StudyPlan, error)
	FindByExamAndUser(ctx context.Context, examID, userID string) (*model.StudyPlan, error)
	UpdateSessionCompletion(ctx context.Context, sessionID, userID string, completed bool) (*model.StudyPlanSession, error)
}

// MasteryRepository fetches quiz attempt answers for mastery computation.
type MasteryRepository interface {
	FindAnswersByTopic(ctx context.Context, userID, topicID string) ([]MasteryAnswer, error)
	FindAnswersByExam(ctx context.Context, userID, examID string) (map[string][]MasteryAnswer, error)
}

// MasteryAnswer is a flattened row from QuizAttemptAnswer for mastery scoring.
type MasteryAnswer struct {
	IsCorrect   bool
	CompletedAt time.Time
	TopicID     string // only for exam-wide queries
}

// TopicResourceItem holds a ResourceTopicMapping joined with Document, Subtopic, and Topic information.
type TopicResourceItem struct {
	ID                  string
	DocumentID          string
	TopicID             string
	SubtopicID          *string
	StartPage           *int
	EndPage             *int
	Confidence          *float64
	Status              string
	CreatedAt           time.Time
	DocID               string
	DocTitle            string
	DocCategory         string
	DocFileName         string
	DocProcessingStatus string
	SubtopicName        *string
	TopicName           string
}

// TopicMappingRepository handles ResourceTopicMapping operations.
type TopicMappingRepository interface {
	ReplaceForDocument(ctx context.Context, documentID string, mappings []model.ResourceTopicMapping) error
	FindByDocument(ctx context.Context, documentID string) ([]TopicResourceItem, error)
	FindByTopicWithRelations(ctx context.Context, topicID, userID string) ([]TopicResourceItem, error)
	FindDocumentIDsByTopic(ctx context.Context, topicID string, excludeRejected bool) ([]string, error)
	FindByID(ctx context.Context, id, userID string) (*model.ResourceTopicMapping, error)
	UpdateStatus(ctx context.Context, id, status string) (*model.ResourceTopicMapping, error)
}

// PyqWithRelations holds a PYQ Question along with its joined document and subtopic info.
type PyqWithRelations struct {
	Question      model.PyqQuestion
	DocumentID    string
	DocumentTitle string
	SubtopicID    *string
	SubtopicName  *string
}

// PyqRepository handles PyqQuestion operations.
type PyqRepository interface {
	ReplaceForDocument(ctx context.Context, documentID string, questions []model.PyqQuestion) error
	FindByDocument(ctx context.Context, documentID string) ([]model.PyqQuestion, error)
	FindByTopic(ctx context.Context, topicID, userID string, excludeRejected bool) ([]PyqWithRelations, error)
	FindByExam(ctx context.Context, examID string, excludeRejected bool) (map[string][]model.PyqQuestion, error)
	FindByID(ctx context.Context, id, userID string) (*model.PyqQuestion, error)
	UpdateStatus(ctx context.Context, id, status string) (*model.PyqQuestion, error)
}

// ResourceExtractionRepository handles ingestion pipeline output.
type ResourceExtractionRepository interface {
	Upsert(ctx context.Context, extraction *model.ResourceExtraction) error
}

// TutorSessionRepository handles tutoring session persistence.
type TutorSessionRepository interface {
	Create(ctx context.Context, session *model.TutorSession) error
	FindByID(ctx context.Context, id, userID string) (*model.TutorSession, error)
	Update(ctx context.Context, id string, updates map[string]interface{}) (*model.TutorSession, error)
}

