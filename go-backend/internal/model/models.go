// Package model defines the domain models for AI Study Saathi.
//
// In Node.js with Prisma, your schema.prisma file is the single source of truth.
// Prisma generates TypeScript types automatically from that schema.
//
// In Go WITHOUT an ORM, you define these structs yourself.
// They serve three purposes simultaneously:
//   1. Represent database rows when scanned from SQL results
//   2. Carry business logic data between service layers
//   3. Optionally, be serialized to JSON in API responses (via DTOs)
//
// Key Go concepts:
//   - Struct composition and embedding
//   - Pointer fields for nullable values (e.g., *string means nullable string)
//   - time.Time for dates (not JS Date — proper timezone handling)
//   - json struct tags for serialization
package model

import "time"

// User maps to the "User" table in schema.prisma.
//
// Concept: Nullable Fields with Pointers
//   In Go, all basic types (string, int, bool) have zero values ("", 0, false).
//   There's no built-in "null". To represent a nullable column:
//     - Use *string instead of string (pointer to string)
//     - nil pointer = SQL NULL
//     - non-nil pointer = has a value
//
//   Prisma/TypeScript uses:
//     email: string    // NOT NULL in SQL
//     examDate: Date | null  // NULLABLE in SQL
//
//   Go equivalent:
//     Email    string     // NOT NULL
//     ExamDate *time.Time // NULLABLE
type User struct {
	ID           string    `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"passwordHash"` // "-" means: never include in JSON
	CreatedAt    time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updatedAt"`
}

// Exam maps to the "Exam" table.
// Fields with * are nullable (optional in Prisma schema).
type Exam struct {
	ID                  string     `json:"id" db:"id"`
	UserID              string     `json:"userId" db:"userId"`
	Name                string     `json:"name" db:"name"`
	Description         *string    `json:"description" db:"description"`
	ExamDate            *time.Time `json:"examDate" db:"examDate"`
	AvailableHoursPerDay *float64  `json:"availableHoursPerDay" db:"availableHoursPerDay"`
	Metadata            *[]byte    `json:"metadata,omitempty" db:"metadata"` // JSON column
	CreatedAt           time.Time  `json:"createdAt" db:"createdAt"`
	UpdatedAt           time.Time  `json:"updatedAt" db:"updatedAt"`

	// Associations — populated by JOIN queries, not always present.
	// A nil slice means "not loaded", an empty slice means "loaded, no rows".
	Subjects []Subject `json:"subjects,omitempty" db:"-"` // db:"-" = not a direct column
}

// Subject maps to the "Subject" table.
type Subject struct {
	ID          string    `json:"id" db:"id"`
	ExamID      string    `json:"examId" db:"examId"`
	UserID      string    `json:"userId" db:"userId"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	OrderIndex  int       `json:"orderIndex" db:"orderIndex"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`

	Topics []Topic `json:"topics,omitempty" db:"-"`
}

// Topic maps to the "Topic" table.
type Topic struct {
	ID          string    `json:"id" db:"id"`
	SubjectID   string    `json:"subjectId" db:"subjectId"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	OrderIndex  int       `json:"orderIndex" db:"orderIndex"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`

	Subtopics []Subtopic `json:"subtopics,omitempty" db:"-"`
}

// Subtopic maps to the "Subtopic" table.
type Subtopic struct {
	ID          string    `json:"id" db:"id"`
	TopicID     string    `json:"topicId" db:"topicId"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	OrderIndex  int       `json:"orderIndex" db:"orderIndex"`
	CreatedAt   time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updatedAt"`
}

// Document maps to the "Document" table.
type Document struct {
	ID                      string    `json:"id" db:"id"`
	UserID                  string    `json:"userId" db:"userId"`
	ExamID                  *string   `json:"examId" db:"examId"`
	Title                   string    `json:"title" db:"title"`
	FileName                string    `json:"fileName" db:"fileName"`
	FilePath                string    `json:"filePath" db:"filePath"`
	FileType                string    `json:"fileType" db:"fileType"`
	FileSize                int64     `json:"fileSize" db:"fileSize"`
	FileHash                *string   `json:"fileHash,omitempty" db:"fileHash"`
	PageCount               *int      `json:"pageCount,omitempty" db:"pageCount"`
	Category                string    `json:"category" db:"category"` // SYLLABUS, NOTES, PYQ, GENERAL
	ProcessingStatus        string    `json:"processingStatus" db:"processingStatus"` // UPLOADED, PROCESSING, READY, FAILED
	AICategory              *string   `json:"aiCategory,omitempty" db:"aiCategory"`
	ClassificationConfidence *float64 `json:"classificationConfidence,omitempty" db:"classificationConfidence"`
	CreatedAt               time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt" db:"updatedAt"`
}

// DocumentChunk maps to the "DocumentChunk" table.
// The embedding field is a pgvector type — we store/retrieve it as a string.
type DocumentChunk struct {
	ID         string    `json:"id" db:"id"`
	DocumentID string    `json:"documentId" db:"documentId"`
	Content    string    `json:"content" db:"content"`
	ChunkIndex int       `json:"chunkIndex" db:"chunkIndex"`
	Metadata   []byte    `json:"metadata,omitempty" db:"metadata"` // JSONB
	CreatedAt  time.Time `json:"createdAt" db:"createdAt"`
	// Embedding is NOT included here — we never fetch the full 3072-dim
	// vector back to Go. We only store it via raw SQL and query it via
	// pgvector's <=> operator in SQL.
}

// Quiz maps to the "Quiz" table.
type Quiz struct {
	ID               string     `json:"id" db:"id"`
	UserID           string     `json:"userId" db:"userId"`
	SubjectID        string     `json:"subjectId" db:"subjectId"`
	TopicID          *string    `json:"topicId" db:"topicId"`
	SubtopicID       *string    `json:"subtopicId" db:"subtopicId"`
	Title            string     `json:"title" db:"title"`
	Difficulty       string     `json:"difficulty" db:"difficulty"` // easy, medium, hard
	SourceDocumentIDs []byte    `json:"sourceDocumentIds,omitempty" db:"sourceDocumentIds"` // JSON array
	CreatedAt        time.Time  `json:"createdAt" db:"createdAt"`

	Questions []QuizQuestion `json:"questions,omitempty" db:"-"`
	Attempts  []QuizAttempt  `json:"attempts,omitempty" db:"-"`
}

// QuizQuestion maps to the "QuizQuestion" table.
type QuizQuestion struct {
	ID            string  `json:"id" db:"id"`
	QuizID        string  `json:"quizId" db:"quizId"`
	Question      string  `json:"question" db:"question"`
	Options       []byte  `json:"options" db:"options"`       // JSON array of strings
	CorrectAnswer string  `json:"correctAnswer" db:"correctAnswer"`
	Explanation   *string `json:"explanation,omitempty" db:"explanation"`
	Difficulty    string  `json:"difficulty" db:"difficulty"`
}

// QuizAttempt maps to the "QuizAttempt" table.
type QuizAttempt struct {
	ID            string     `json:"id" db:"id"`
	QuizID        string     `json:"quizId" db:"quizId"`
	UserID        string     `json:"userId" db:"userId"`
	Score         float64    `json:"score" db:"score"`
	TotalQuestions int       `json:"totalQuestions" db:"totalQuestions"`
	CorrectCount  int        `json:"correctCount" db:"correctCount"`
	StartedAt     time.Time  `json:"startedAt" db:"startedAt"`
	CompletedAt   *time.Time `json:"completedAt" db:"completedAt"`

	Answers []QuizAttemptAnswer `json:"answers,omitempty" db:"-"`
}

// QuizAttemptAnswer maps to the "QuizAttemptAnswer" table.
type QuizAttemptAnswer struct {
	ID             string    `json:"id" db:"id"`
	AttemptID      string    `json:"attemptId" db:"attemptId"`
	QuestionID     string    `json:"questionId" db:"questionId"`
	SelectedAnswer *string   `json:"selectedAnswer" db:"selectedAnswer"`
	IsCorrect      bool      `json:"isCorrect" db:"isCorrect"`
	CreatedAt      time.Time `json:"createdAt" db:"createdAt"`
}

// Flashcard maps to the "Flashcard" table with SM-2 state.
type Flashcard struct {
	ID               string     `json:"id" db:"id"`
	UserID           string     `json:"userId" db:"userId"`
	SubjectID        string     `json:"subjectId" db:"subjectId"`
	TopicID          *string    `json:"topicId" db:"topicId"`
	SubtopicID       *string    `json:"subtopicId" db:"subtopicId"`
	Front            string     `json:"front" db:"front"`
	Back             string     `json:"back" db:"back"`
	Hint             *string    `json:"hint,omitempty" db:"hint"`
	NormalizedFront  string     `json:"-" db:"normalizedFront"`
	SourceDocumentIDs []byte    `json:"sourceDocumentIds,omitempty" db:"sourceDocumentIds"`
	// SM-2 state
	EaseFactor    float64    `json:"easeFactor" db:"easeFactor"`
	IntervalDays  int        `json:"intervalDays" db:"intervalDays"`
	Repetitions   int        `json:"repetitions" db:"repetitions"`
	DueAt         time.Time  `json:"dueAt" db:"dueAt"`
	LastReviewedAt *time.Time `json:"lastReviewedAt" db:"lastReviewedAt"`
	CreatedAt     time.Time  `json:"createdAt" db:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updatedAt"`
}

// TutorSession maps to the "TutorSession" table.
type TutorSession struct {
	ID                  string    `json:"id" db:"id"`
	UserID              string    `json:"userId" db:"userId"`
	Subject             string    `json:"subject" db:"subject"`
	Topic               string    `json:"topic" db:"topic"`
	Status              string    `json:"status" db:"status"` // ACTIVE, COMPLETED
	ConversationHistory []byte    `json:"conversationHistory" db:"conversationHistory"` // JSON
	CurrentConcept      *string   `json:"currentConcept" db:"currentConcept"`
	WeakAreas           []byte    `json:"weakAreas" db:"weakAreas"` // JSON array of strings
	QuestionCount       int       `json:"questionCount" db:"questionCount"`
	CreatedAt           time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt" db:"updatedAt"`
}

// StudyPlan maps to the "StudyPlan" table.
type StudyPlan struct {
	ID          string    `json:"id" db:"id"`
	ExamID      string    `json:"examId" db:"examId"`
	UserID      string    `json:"userId" db:"userId"`
	StartDate   time.Time `json:"startDate" db:"startDate"`
	EndDate     time.Time `json:"endDate" db:"endDate"`
	GeneratedAt time.Time `json:"generatedAt" db:"generatedAt"`

	Sessions []StudyPlanSession `json:"sessions,omitempty" db:"-"`
}

// StudyPlanSession maps to the "StudyPlanSession" table.
type StudyPlanSession struct {
	ID              string     `json:"id" db:"id"`
	PlanID          string     `json:"planId" db:"planId"`
	Date            time.Time  `json:"date" db:"date"`
	TopicID         string     `json:"topicId" db:"topicId"`
	SubtopicID      *string    `json:"subtopicId" db:"subtopicId"`
	DurationMinutes int        `json:"durationMinutes" db:"durationMinutes"`
	PriorityScore   float64    `json:"priorityScore" db:"priorityScore"`
	Reason          string     `json:"reason" db:"reason"`
	Completed       bool       `json:"completed" db:"completed"`
	CompletedAt     *time.Time `json:"completedAt" db:"completedAt"`
	CreatedAt       time.Time  `json:"createdAt" db:"createdAt"`

	Topic *Topic `json:"topic,omitempty" db:"-"`
}

// AssistantSession maps to the "AssistantSession" table.
type AssistantSession struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"userId" db:"userId"`
	ExamID    string    `json:"examId" db:"examId"`
	Messages  []byte    `json:"messages" db:"messages"` // JSON array of message objects
	CreatedAt time.Time `json:"createdAt" db:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt" db:"updatedAt"`
}

// ResourceTopicMapping maps to the "ResourceTopicMapping" table.
type ResourceTopicMapping struct {
	ID         string   `json:"id" db:"id"`
	DocumentID string   `json:"documentId" db:"documentId"`
	TopicID    string   `json:"topicId" db:"topicId"`
	SubtopicID *string  `json:"subtopicId" db:"subtopicId"`
	StartPage  *int     `json:"startPage" db:"startPage"`
	EndPage    *int     `json:"endPage" db:"endPage"`
	Confidence *float64 `json:"confidence" db:"confidence"`
	Status     string   `json:"status" db:"status"` // CONFIRMED, PENDING_REVIEW, REJECTED
}

// PyqQuestion maps to the "PyqQuestion" table.
type PyqQuestion struct {
	ID            string   `json:"id" db:"id"`
	DocumentID    string   `json:"documentId" db:"documentId"`
	TopicID       *string  `json:"topicId" db:"topicId"`
	SubtopicID    *string  `json:"subtopicId" db:"subtopicId"`
	QuestionText  string   `json:"questionText" db:"questionText"`
	NormalizedText string  `json:"-" db:"normalizedText"`
	Options       *[]byte  `json:"options,omitempty" db:"options"`
	CorrectAnswer *string  `json:"correctAnswer,omitempty" db:"correctAnswer"`
	Year          *int     `json:"year,omitempty" db:"year"`
	Confidence    *float64 `json:"confidence,omitempty" db:"confidence"`
	Status        string   `json:"status" db:"status"` // CONFIRMED, PENDING_REVIEW, REJECTED
}

// ResourceExtraction maps to the "ResourceExtraction" table.
type ResourceExtraction struct {
	ID           string   `json:"id" db:"id"`
	DocumentID   string   `json:"documentId" db:"documentId"`
	ResourceType string   `json:"resourceType" db:"resourceType"`
	Summary      *string  `json:"summary,omitempty" db:"summary"`
	Topics       *[]byte  `json:"topics,omitempty" db:"topics"`
	Questions    *[]byte  `json:"questions,omitempty" db:"questions"`
	Confidence   *float64 `json:"confidence,omitempty" db:"confidence"`
}

