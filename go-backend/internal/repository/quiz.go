// Package repository implements data access for Quizzes, Questions, and Attempts.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxQuizRepository struct {
	pool *pgxpool.Pool
}

// NewQuizRepository creates a new QuizRepository.
func NewQuizRepository(pool *pgxpool.Pool) QuizRepository {
	return &pgxQuizRepository{pool: pool}
}

// Create stores a Quiz and its Questions in a single SQL transaction.
func (r *pgxQuizRepository) Create(ctx context.Context, quiz *model.Quiz, questions []model.QuizQuestion) (*model.Quiz, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting quiz creation transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	quiz.ID = uuid.New().String()
	quiz.CreatedAt = now

	// 1. Insert Quiz
	_, err = tx.Exec(ctx,
		`INSERT INTO "Quiz" (id, "userId", "subjectId", "topicId", "subtopicId", title, difficulty, "sourceDocumentIds", "createdAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		quiz.ID, quiz.UserID, quiz.SubjectID, quiz.TopicID, quiz.SubtopicID,
		quiz.Title, quiz.Difficulty, quiz.SourceDocumentIDs, now,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting quiz: %w", err)
	}

	// 2. Insert Questions
	for i := range questions {
		q := &questions[i]
		q.ID = uuid.New().String()
		q.QuizID = quiz.ID

		_, err = tx.Exec(ctx,
			`INSERT INTO "QuizQuestion" (id, "quizId", question, options, "correctAnswer", explanation, difficulty)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			q.ID, q.QuizID, q.Question, q.Options, q.CorrectAnswer, q.Explanation, q.Difficulty,
		)
		if err != nil {
			return nil, fmt.Errorf("inserting quiz question %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing quiz transaction: %w", err)
	}

	quiz.Questions = questions
	return quiz, nil
}

// FindByID retrieves a quiz with its questions.
func (r *pgxQuizRepository) FindByID(ctx context.Context, id, userID string) (*model.Quiz, error) {
	// 1. Fetch Quiz
	quiz := &model.Quiz{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, "userId", "subjectId", "topicId", "subtopicId", title, difficulty, "sourceDocumentIds", "createdAt"
		 FROM "Quiz"
		 WHERE id = $1 AND "userId" = $2`,
		id, userID,
	).Scan(
		&quiz.ID, &quiz.UserID, &quiz.SubjectID, &quiz.TopicID, &quiz.SubtopicID,
		&quiz.Title, &quiz.Difficulty, &quiz.SourceDocumentIDs, &quiz.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching quiz: %w", err)
	}

	// 2. Fetch Questions
	rows, err := r.pool.Query(ctx,
		`SELECT id, "quizId", question, options, "correctAnswer", explanation, difficulty
		 FROM "QuizQuestion"
		 WHERE "quizId" = $1`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching quiz questions: %w", err)
	}
	defer rows.Close()

	var questions []model.QuizQuestion
	for rows.Next() {
		var q model.QuizQuestion
		err := rows.Scan(
			&q.ID, &q.QuizID, &q.Question, &q.Options, &q.CorrectAnswer, &q.Explanation, &q.Difficulty,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning quiz question: %w", err)
		}
		questions = append(questions, q)
	}

	quiz.Questions = questions
	return quiz, nil
}

// FindByTopic retrieves all quizzes for a topic, with questions count and the latest attempt.
func (r *pgxQuizRepository) FindByTopic(ctx context.Context, topicID, userID string) ([]model.Quiz, error) {
	// Query quizzes for this topic
	query := `
		SELECT q.id, q."userId", q."subjectId", q."topicId", q."subtopicId", q.title, q.difficulty, q."createdAt",
		       COUNT(qq.id) as question_count
		FROM "Quiz" q
		LEFT JOIN "QuizQuestion" qq ON qq."quizId" = q.id
		WHERE q."topicId" = $1 AND q."userId" = $2
		GROUP BY q.id
		ORDER BY q."createdAt" DESC
	`
	rows, err := r.pool.Query(ctx, query, topicID, userID)
	if err != nil {
		return nil, fmt.Errorf("querying quizzes by topic: %w", err)
	}
	defer rows.Close()

	var quizzes []model.Quiz
	for rows.Next() {
		var q model.Quiz
		var qCount int
		err := rows.Scan(
			&q.ID, &q.UserID, &q.SubjectID, &q.TopicID, &q.SubtopicID,
			&q.Title, &q.Difficulty, &q.CreatedAt, &qCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning topic quiz row: %w", err)
		}
		if qCount > 0 {
			q.Questions = make([]model.QuizQuestion, qCount)
		}
		quizzes = append(quizzes, q)
	}

	// For each quiz, fetch latest attempt if any
	for i := range quizzes {
		attempt := &model.QuizAttempt{}
		err := r.pool.QueryRow(ctx,
			`SELECT id, score, "completedAt"
			 FROM "QuizAttempt"
			 WHERE "quizId" = $1 AND "userId" = $2
			 ORDER BY "completedAt" DESC NULLS LAST
			 LIMIT 1`,
			quizzes[i].ID, userID,
		).Scan(&attempt.ID, &attempt.Score, &attempt.CompletedAt)
		if err == nil {
			quizzes[i].Attempts = []model.QuizAttempt{*attempt}
		}
	}

	return quizzes, nil
}

// CreateAttempt records a student's attempt and answers atomically.
func (r *pgxQuizRepository) CreateAttempt(ctx context.Context, attempt *model.QuizAttempt, answers []model.QuizAttemptAnswer) (*model.QuizAttempt, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting attempt transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	attempt.ID = uuid.New().String()
	now := time.Now().UTC()
	attempt.StartedAt = now
	attempt.CompletedAt = &now

	// 1. Insert Attempt
	_, err = tx.Exec(ctx,
		`INSERT INTO "QuizAttempt" (id, "quizId", "userId", score, "totalQuestions", "correctCount", "startedAt", "completedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		attempt.ID, attempt.QuizID, attempt.UserID, attempt.Score, attempt.TotalQuestions,
		attempt.CorrectCount, attempt.StartedAt, attempt.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting quiz attempt: %w", err)
	}

	// 2. Insert Answers
	for i := range answers {
		a := &answers[i]
		a.ID = uuid.New().String()
		a.AttemptID = attempt.ID
		a.CreatedAt = now

		_, err = tx.Exec(ctx,
			`INSERT INTO "QuizAttemptAnswer" (id, "attemptId", "questionId", "selectedAnswer", "isCorrect", "createdAt")
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			a.ID, a.AttemptID, a.QuestionID, a.SelectedAnswer, a.IsCorrect, a.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("inserting attempt answer %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing attempt transaction: %w", err)
	}

	attempt.Answers = answers
	return attempt, nil
}

// FindAttemptsByQuiz returns attempt history for a quiz.
func (r *pgxQuizRepository) FindAttemptsByQuiz(ctx context.Context, quizID, userID string) ([]model.QuizAttempt, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, "quizId", "userId", score, "totalQuestions", "correctCount", "startedAt", "completedAt"
		 FROM "QuizAttempt"
		 WHERE "quizId" = $1 AND "userId" = $2
		 ORDER BY "completedAt" DESC`,
		quizID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("querying quiz attempts: %w", err)
	}
	defer rows.Close()

	var attempts []model.QuizAttempt
	for rows.Next() {
		var a model.QuizAttempt
		err := rows.Scan(
			&a.ID, &a.QuizID, &a.UserID, &a.Score, &a.TotalQuestions,
			&a.CorrectCount, &a.StartedAt, &a.CompletedAt,
		)
		if err != nil {
			return nil, err
		}
		attempts = append(attempts, a)
	}
	return attempts, nil
}

// FindAttemptByID fetches a single attempt with answers.
func (r *pgxQuizRepository) FindAttemptByID(ctx context.Context, attemptID, quizID, userID string) (*model.QuizAttempt, error) {
	attempt := &model.QuizAttempt{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, "quizId", "userId", score, "totalQuestions", "correctCount", "startedAt", "completedAt"
		 FROM "QuizAttempt"
		 WHERE id = $1 AND "quizId" = $2 AND "userId" = $3`,
		attemptID, quizID, userID,
	).Scan(
		&attempt.ID, &attempt.QuizID, &attempt.UserID, &attempt.Score, &attempt.TotalQuestions,
		&attempt.CorrectCount, &attempt.StartedAt, &attempt.CompletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching attempt: %w", err)
	}

	// Fetch answers
	rows, err := r.pool.Query(ctx,
		`SELECT id, "attemptId", "questionId", "selectedAnswer", "isCorrect", "createdAt"
		 FROM "QuizAttemptAnswer"
		 WHERE "attemptId" = $1`,
		attemptID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching attempt answers: %w", err)
	}
	defer rows.Close()

	var answers []model.QuizAttemptAnswer
	for rows.Next() {
		var a model.QuizAttemptAnswer
		if err := rows.Scan(&a.ID, &a.AttemptID, &a.QuestionID, &a.SelectedAnswer, &a.IsCorrect, &a.CreatedAt); err != nil {
			return nil, err
		}
		answers = append(answers, a)
	}
	attempt.Answers = answers

	return attempt, nil
}

