// Package dto contains Data Transfer Objects — the shapes of data that
// cross the HTTP boundary (request bodies and response bodies).
//
// Why DTOs separate from models?
//   - Models represent the DATABASE shape (all columns, internal fields like passwordHash)
//   - DTOs represent the API WIRE shape (what the client sends/receives)
//   - Separating them prevents accidentally leaking internal fields
//     (e.g., passwordHash must NEVER appear in an API response)
//
// In Node.js, you managed this by selecting specific Prisma fields:
//   prisma.user.findUnique({ select: { id: true, name: true, email: true } })
//
// In Go, we have separate struct types for each concern.
// The compile-time type system makes it impossible to accidentally include
// a field that isn't in the DTO struct.
//
// validate struct tags:
//   required       → field must be present and non-zero
//   email         → must be a valid email address
//   min=N         → string minimum length N
//   oneof=a b c   → must be one of these values
//
// These are interpreted by go-playground/validator/v10
// (bundled with Gin, same library your Joi/Zod replace)
package dto

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// =============================================================================
// Auth DTOs
// =============================================================================

// RegisterRequest is the body for POST /api/auth/register
// Equivalent to your Joi registerSchema in auth.validator.js
type RegisterRequest struct {
	Name     string `json:"name" binding:"required,min=2,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest is the body for POST /api/auth/login
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is the success response body for register and login.
// Notice: no passwordHash. The DTO enforces this at the type level.
type AuthResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Token string `json:"token"`
}

// UserResponse is the response for GET /api/auth/me
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// =============================================================================
// Exam DTOs
// =============================================================================

func parseFlexibleDate(val interface{}) *time.Time {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" || s == "null" {
			return nil
		}
		// Clean up any double hyphens if user or date picker produced "2027-02--24"
		for strings.Contains(s, "--") {
			s = strings.ReplaceAll(s, "--", "-")
		}
		layouts := []string{
			time.RFC3339,
			"2006-01-02T15:04:05.000Z07:00",
			"2006-01-02T15:04:05.000Z",
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
			"2006-01-02",
			"02-01-2006",
			"2006/01/02",
			"02/01/2006",
		}
		for _, layout := range layouts {
			if t, err := time.Parse(layout, s); err == nil {
				return &t
			}
		}
	}
	return nil
}

func parseFlexibleFloat(val interface{}) *float64 {
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case float64:
		return &v
	case int:
		f := float64(v)
		return &f
	case string:
		s := strings.TrimSpace(v)
		if s == "" || s == "null" {
			return nil
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return &f
		}
	}
	return nil
}

// SubtopicInput is used inside CreateExamRequest for creating subtopics inline.
type SubtopicInput struct {
	ID          *string `json:"id,omitempty"`
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description"`
	OrderIndex  int     `json:"orderIndex"`
}

// TopicInput is used inside CreateExamRequest.
type TopicInput struct {
	ID          *string         `json:"id,omitempty"`
	Name        string          `json:"name" binding:"required"`
	Description *string         `json:"description"`
	OrderIndex  int             `json:"orderIndex"`
	Subtopics   []SubtopicInput `json:"subtopics"`
}

// SubjectInput is used inside CreateExamRequest.
type SubjectInput struct {
	ID          *string      `json:"id,omitempty"`
	Name        string       `json:"name" binding:"required"`
	Description *string      `json:"description"`
	OrderIndex  int          `json:"orderIndex"`
	Topics      []TopicInput `json:"topics"`
}

// CreateExamRequest is the body for POST /api/exams
type CreateExamRequest struct {
	Name                 string          `json:"name" binding:"required,min=1,max=200"`
	Description          *string         `json:"description"`
	ExamDate             *time.Time      `json:"examDate"`
	AvailableHoursPerDay *float64        `json:"availableHoursPerDay"`
	Metadata             json.RawMessage `json:"metadata,omitempty"`
	Subjects             []SubjectInput  `json:"subjects"`
}

func (r *CreateExamRequest) UnmarshalJSON(data []byte) error {
	type Alias CreateExamRequest
	aux := &struct {
		RawExamDate interface{} `json:"examDate"`
		RawHours    interface{} `json:"availableHoursPerDay"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.RawExamDate != nil {
		r.ExamDate = parseFlexibleDate(aux.RawExamDate)
	}
	if aux.RawHours != nil {
		r.AvailableHoursPerDay = parseFlexibleFloat(aux.RawHours)
	}
	return nil
}

// ReplaceSyllabusRequest is the body for PUT /api/exams/:id/syllabus
type ReplaceSyllabusRequest struct {
	Subjects []SubjectInput `json:"subjects" binding:"required"`
}

// ExtractSyllabusRequest is the body or form payload for POST /api/exams/syllabus/extract
type ExtractSyllabusRequest struct {
	Text string `json:"text" form:"text"`
}

// ExtractedSubtopic is a subtopic parsed by Gemini during syllabus extraction
type ExtractedSubtopic struct {
	Name string `json:"name"`
}

// ExtractedTopic is a topic with nested subtopics parsed by Gemini
type ExtractedTopic struct {
	Name      string              `json:"name"`
	Subtopics []ExtractedSubtopic `json:"subtopics"`
}

// ExtractedSubject is a subject with nested topics parsed by Gemini
type ExtractedSubject struct {
	Name   string           `json:"name"`
	Topics []ExtractedTopic `json:"topics"`
}

// ExtractSyllabusResponse is returned by POST /api/exams/syllabus/extract
type ExtractSyllabusResponse struct {
	Subjects []ExtractedSubject `json:"subjects"`
}

// ExamCountInfo holds relation counts for list responses.
// Matches Node.js: { _count: { subjects: 3 } }
type ExamCountInfo struct {
	Subjects int `json:"subjects"`
}

// ExamListItemResponse represents an exam in GET /api/exams
type ExamListItemResponse struct {
	ID                   string         `json:"id"`
	UserID               string         `json:"userId"`
	Name                 string         `json:"name"`
	Description          *string        `json:"description"`
	ExamDate             *time.Time     `json:"examDate"`
	AvailableHoursPerDay *float64       `json:"availableHoursPerDay"`
	Metadata             *[]byte        `json:"metadata,omitempty"`
	CreatedAt            time.Time      `json:"createdAt"`
	UpdatedAt            time.Time      `json:"updatedAt"`
	Count                ExamCountInfo  `json:"_count"`
}

// SubtopicResponse represents a subtopic in syllabus details
type SubtopicResponse struct {
	ID          string    `json:"id"`
	TopicID     string    `json:"topicId"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	OrderIndex  int       `json:"orderIndex"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TopicResponse represents a topic with its subtopics
type TopicResponse struct {
	ID          string             `json:"id"`
	SubjectID   string             `json:"subjectId"`
	Name        string             `json:"name"`
	Description *string            `json:"description"`
	OrderIndex  int                `json:"orderIndex"`
	CreatedAt   time.Time          `json:"createdAt"`
	UpdatedAt   time.Time          `json:"updatedAt"`
	Subtopics   []SubtopicResponse `json:"subtopics"`
}

// SubjectResponse represents a subject with its topics and subtopics
type SubjectResponse struct {
	ID          string          `json:"id"`
	ExamID      string          `json:"examId"`
	UserID      string          `json:"userId"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	OrderIndex  int             `json:"orderIndex"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	Topics      []TopicResponse `json:"topics"`
}

// ExamDetailResponse represents the complete exam with full syllabus tree
type ExamDetailResponse struct {
	ID                   string            `json:"id"`
	UserID               string            `json:"userId"`
	Name                 string            `json:"name"`
	Description          *string           `json:"description"`
	ExamDate             *time.Time        `json:"examDate"`
	AvailableHoursPerDay *float64          `json:"availableHoursPerDay"`
	Metadata             *[]byte           `json:"metadata,omitempty"`
	CreatedAt            time.Time         `json:"createdAt"`
	UpdatedAt            time.Time         `json:"updatedAt"`
	Subjects             []SubjectResponse `json:"subjects"`
}

// UpdateExamRequest is the body for PATCH /api/exams/:id
// All fields are optional (pointer types), so the client only sends what changed.
type UpdateExamRequest struct {
	Name                 *string    `json:"name"`
	Description          *string    `json:"description"`
	ExamDate             *time.Time `json:"examDate"`
	AvailableHoursPerDay *float64   `json:"availableHoursPerDay"`
}

func (r *UpdateExamRequest) UnmarshalJSON(data []byte) error {
	type Alias UpdateExamRequest
	aux := &struct {
		RawExamDate interface{} `json:"examDate"`
		RawHours    interface{} `json:"availableHoursPerDay"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.RawExamDate != nil {
		r.ExamDate = parseFlexibleDate(aux.RawExamDate)
	}
	if aux.RawHours != nil {
		r.AvailableHoursPerDay = parseFlexibleFloat(aux.RawHours)
	}
	return nil
}

// =============================================================================
// Document DTOs
// =============================================================================

// DocumentSummary is returned in list/upload responses (no filePath for security).
type DocumentSummary struct {
	ID                       string     `json:"id"`
	Title                    string     `json:"title"`
	FileName                 string     `json:"fileName"`
	FileType                 string     `json:"fileType"`
	FileSize                 int64      `json:"fileSize"`
	ExamID                   *string    `json:"examId"`
	Category                 string     `json:"category"`
	AICategory               *string    `json:"aiCategory"`
	ClassificationConfidence *float64   `json:"classificationConfidence"`
	PageCount                *int       `json:"pageCount"`
	ProcessingStatus         string     `json:"processingStatus"`
	CreatedAt                time.Time  `json:"createdAt"`
	UpdatedAt                time.Time  `json:"updatedAt"`
}

// UpdateDocumentRequest is the body for PATCH /api/documents/:id
type UpdateDocumentRequest struct {
	Category string `json:"category" binding:"required,oneof=SYLLABUS NOTES PYQ GENERAL"`
}

// DocumentStatusResponse is the response for GET /api/documents/:id/status
type DocumentStatusResponse struct {
	Status string `json:"status"`
}

// DocumentDetailResponse is the response for GET /api/documents/:id
type DocumentDetailResponse struct {
	ID                       string      `json:"id"`
	UserID                   string      `json:"userId"`
	ExamID                   *string     `json:"examId"`
	Title                    string      `json:"title"`
	FileName                 string      `json:"fileName"`
	FileType                 string      `json:"fileType"`
	FileSize                 int64       `json:"fileSize"`
	Category                 string      `json:"category"`
	AICategory               *string     `json:"aiCategory"`
	ClassificationConfidence *float64    `json:"classificationConfidence"`
	PageCount                *int        `json:"pageCount"`
	ProcessingStatus         string      `json:"processingStatus"`
	CreatedAt                time.Time   `json:"createdAt"`
	UpdatedAt                time.Time   `json:"updatedAt"`
	Extraction               interface{} `json:"extraction,omitempty"`
}

// =============================================================================
// Quiz DTOs
// =============================================================================

// GenerateQuizRequest is the body for POST /api/exams/:id/topics/:topicId/quizzes
type GenerateQuizRequest struct {
	NumberOfQuestions int    `json:"numberOfQuestions"`
	Difficulty        string `json:"difficulty" binding:"omitempty,oneof=easy medium hard"`
	SubtopicID        string `json:"subtopicId"`
}

// TakeableQuestion strips the correctAnswer from QuizQuestion (sent to client pre-attempt)
type TakeableQuestion struct {
	ID         string   `json:"id"`
	Question   string   `json:"question"`
	Options    []string `json:"options"`
	Difficulty string   `json:"difficulty"`
}

// SubmitAnswerItem is one answer in a quiz submission
type SubmitAnswerItem struct {
	QuestionID     string  `json:"questionId" binding:"required"`
	SelectedAnswer *string `json:"selectedAnswer"`
}

// SubmitQuizRequest is the body for POST /api/quizzes/:quizId/submit
type SubmitQuizRequest struct {
	Answers []SubmitAnswerItem `json:"answers" binding:"required"`
}

// TakeableQuizResponse is returned when getting a quiz to take (no answers)
type TakeableQuizResponse struct {
	ID                string             `json:"id"`
	Title             string             `json:"title"`
	Difficulty        string             `json:"difficulty"`
	TopicID           *string            `json:"topicId"`
	SubtopicID        *string            `json:"subtopicId"`
	SourceDocumentIDs interface{}        `json:"sourceDocumentIds,omitempty"`
	CreatedAt         time.Time          `json:"createdAt"`
	Questions         []TakeableQuestion `json:"questions"`
}

// LastAttemptSummary holds the score of the most recent attempt for topic listing
type LastAttemptSummary struct {
	ID          string     `json:"id"`
	Score       float64    `json:"score"`
	CompletedAt *time.Time `json:"completedAt"`
}

// TopicQuizListItemResponse represents an item in GET /api/exams/:id/topics/:topicId/quizzes
type TopicQuizListItemResponse struct {
	ID            string              `json:"id"`
	Title         string              `json:"title"`
	Difficulty    string              `json:"difficulty"`
	SubtopicID    *string             `json:"subtopicId"`
	QuestionCount int                 `json:"questionCount"`
	CreatedAt     time.Time           `json:"createdAt"`
	LastAttempt   *LastAttemptSummary `json:"lastAttempt"`
}

// GradedAnswerDetail represents a graded answer with full breakdown
type GradedAnswerDetail struct {
	QuestionID     string   `json:"questionId"`
	Question       string   `json:"question"`
	Options        []string `json:"options"`
	SelectedAnswer *string  `json:"selectedAnswer"`
	CorrectAnswer  string   `json:"correctAnswer"`
	Explanation    *string  `json:"explanation"`
	IsCorrect      bool     `json:"isCorrect"`
}

// QuizAttemptDetailResponse represents an attempt with per-question graded results
type QuizAttemptDetailResponse struct {
	ID             string               `json:"id"`
	QuizID         string               `json:"quizId"`
	Score          float64              `json:"score"`
	CorrectCount   int                  `json:"correctCount"`
	TotalQuestions int                  `json:"totalQuestions"`
	CompletedAt    *time.Time           `json:"completedAt"`
	Answers        []GradedAnswerDetail `json:"answers"`
}

// QuizAttemptListItem represents a row in GET /api/quizzes/:quizId/attempts
type QuizAttemptListItem struct {
	ID             string     `json:"id"`
	Score          float64    `json:"score"`
	CorrectCount   int        `json:"correctCount"`
	TotalQuestions int        `json:"totalQuestions"`
	StartedAt      time.Time  `json:"startedAt"`
	CompletedAt    *time.Time `json:"completedAt"`
}

// =============================================================================
// Flashcard DTOs
// =============================================================================

// GenerateFlashcardsRequest is the body for POST /api/exams/:id/topics/:topicId/flashcards
type GenerateFlashcardsRequest struct {
	NumberOfCards int    `json:"numberOfCards"`
	SubtopicID    string `json:"subtopicId"`
}

// ReviewFlashcardRequest is the body for POST /api/flashcards/:id/review
type ReviewFlashcardRequest struct {
	Grade string `json:"grade" binding:"required,oneof=AGAIN HARD GOOD EASY"`
}

// =============================================================================
// Study Plan DTOs
// =============================================================================

// GenerateStudyPlanRequest is the body for POST /api/exams/:id/study-plan
type GenerateStudyPlanRequest struct{}

// GenerateStudyPlanResponse is the payload for POST /api/exams/:id/study-plan
type GenerateStudyPlanResponse struct {
	Plan            interface{} `json:"plan"`
	SkippedTopicIDs []string    `json:"skippedTopicIds"`
}

// UpdateSessionCompletionRequest is the payload for PATCH /api/study-plan-sessions/:sessionId
type UpdateSessionCompletionRequest struct {
	Completed *bool `json:"completed" binding:"required"`
}

// =============================================================================
// AI / RAG DTOs
// =============================================================================

// AskQuestionRequest is the body for POST /api/ai/ask
type AskQuestionRequest struct {
	Question string `json:"question" binding:"required"`
	ExamID   string `json:"examId"`
	Category string `json:"category"`
}

// =============================================================================
// Tutoring DTOs
// =============================================================================

// StartTutoringRequest is the body for POST /api/tutoring/session/start
type StartTutoringRequest struct {
	Subject string `json:"subject" binding:"required"`
	Topic   string `json:"topic" binding:"required"`
}

// StartTutoringResponse is returned when a new tutor session starts
type StartTutoringResponse struct {
	SessionID     string `json:"sessionId"`
	AIMessage     string `json:"aiMessage"`
	QuestionCount int    `json:"questionCount"`
	SessionStatus string `json:"sessionStatus"`
}

// RespondToTutorRequest is the body for POST /api/tutoring/session/:id/respond
type RespondToTutorRequest struct {
	Answer string `json:"answer" binding:"required"`
}

// RespondToTutorResponse is returned on each tutoring turn
type RespondToTutorResponse struct {
	AIMessage     string   `json:"aiMessage"`
	QuestionCount int      `json:"questionCount"`
	SessionStatus string   `json:"sessionStatus"`
	WeakAreas     []string `json:"weakAreas"`
}

// =============================================================================
// Assistant DTOs
// =============================================================================

// SendAssistantMessageRequest is the body for POST /api/exams/:id/assistant/message
type SendAssistantMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

// SendAssistantMessageResponse is returned on each assistant turn
type SendAssistantMessageResponse struct {
	Reply     string   `json:"reply"`
	ToolsUsed []string `json:"toolsUsed"`
	SessionID string   `json:"sessionId"`
}

// =============================================================================
// Mastery & PYQ DTOs
// =============================================================================

// TopicReference represents a minimal topic identifier and name
type TopicReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SubtopicReference represents a minimal subtopic identifier and name
type SubtopicReference struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// DocumentReference represents a minimal document identifier and title
type DocumentReference struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// MasteryResult is the recency-weighted accuracy output for a topic
type MasteryResult struct {
	Score         float64 `json:"score"`
	Level         string  `json:"level"` // NOT_STARTED, WEAK, DEVELOPING, STRONG
	Accuracy      float64 `json:"accuracy"`
	AttemptCount  int     `json:"attemptCount"`
	LowConfidence bool    `json:"lowConfidence"`
}

// TopicMasteryResponse is the payload for GET /api/exams/:id/topics/:topicId/mastery
type TopicMasteryResponse struct {
	Topic   TopicReference `json:"topic"`
	Mastery MasteryResult  `json:"mastery"`
}

// ExamMasteryItem is one entry in GET /api/exams/:id/mastery
type ExamMasteryItem struct {
	Topic   TopicReference `json:"topic"`
	Mastery MasteryResult  `json:"mastery"`
}

// PyqFrequencyResult is the historical importance output for a topic
type PyqFrequencyResult struct {
	Score         float64 `json:"score"`
	Level         string  `json:"level"` // NONE, LOW, MEDIUM, HIGH
	YearsSeen     []int   `json:"yearsSeen"`
	QuestionCount int     `json:"questionCount"`
}

// TopicPyqQuestionItem represents a question in GET /api/exams/:id/topics/:topicId/pyq-analysis
type TopicPyqQuestionItem struct {
	ID            string             `json:"id"`
	QuestionText  string             `json:"questionText"`
	Options       interface{}        `json:"options,omitempty"`
	CorrectAnswer *string            `json:"correctAnswer,omitempty"`
	Year          *int               `json:"year,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
	Status        string             `json:"status"`
	Subtopic      *SubtopicReference `json:"subtopic,omitempty"`
	Document      *DocumentReference `json:"document,omitempty"`
}

// TopicPyqAnalysisResponse is the payload for GET /api/exams/:id/topics/:topicId/pyq-analysis
type TopicPyqAnalysisResponse struct {
	Topic     TopicReference         `json:"topic"`
	Frequency PyqFrequencyResult     `json:"frequency"`
	Questions []TopicPyqQuestionItem `json:"questions"`
}

// UpdateQuestionStatusRequest is the payload for PATCH /api/pyq-questions/:questionId
type UpdateQuestionStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=CONFIRMED REJECTED PENDING_REVIEW"`
}

