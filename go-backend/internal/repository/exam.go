// Package repository implements data access for the Exam & Syllabus hierarchy.
//
// Key Go concepts demonstrated:
//   - Database transactions with pgx.Tx: Begin, Rollback, Commit
//   - Safe dynamic SQL query building with parameterized values ($1, $2, ...)
//   - Efficient in-memory tree assembly using maps and slices
//   - Multi-row queries with rows.Next() and defer rows.Close()
package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgxExamRepository struct {
	pool *pgxpool.Pool
}

// NewExamRepository creates a new pgx-backed ExamRepository.
func NewExamRepository(pool *pgxpool.Pool) ExamRepository {
	return &pgxExamRepository{pool: pool}
}

// Create inserts a new Exam record.
func (r *pgxExamRepository) Create(ctx context.Context, exam *model.Exam, subjects []model.Subject) (*model.Exam, error) {
	now := time.Now().UTC()
	exam.ID = uuid.New().String()
	exam.CreatedAt = now
	exam.UpdatedAt = now

	err := r.pool.QueryRow(ctx,
		`INSERT INTO "Exam" (id, "userId", name, description, "examDate", "availableHoursPerDay", metadata, "createdAt", "updatedAt")
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING "createdAt", "updatedAt"`,
		exam.ID,
		exam.UserID,
		exam.Name,
		exam.Description,
		exam.ExamDate,
		exam.AvailableHoursPerDay,
		exam.Metadata,
		now,
		now,
	).Scan(&exam.CreatedAt, &exam.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("inserting exam: %w", err)
	}

	return exam, nil
}

// FindAllByUser returns all exams for a user with the count of subjects.
// Matches Prisma: findMany({ where: { userId }, orderBy: { createdAt: 'desc' }, include: { _count: { select: { subjects: true } } } })
func (r *pgxExamRepository) FindAllByUser(ctx context.Context, userID string) ([]model.Exam, error) {
	// A LEFT JOIN with COUNT allows us to fetch the subject counts in a single query!
	query := `
		SELECT 
			e.id, e."userId", e.name, e.description, e."examDate", 
			e."availableHoursPerDay", e.metadata, e."createdAt", e."updatedAt",
			COUNT(s.id) AS subject_count
		FROM "Exam" e
		LEFT JOIN "Subject" s ON s."examId" = e.id
		WHERE e."userId" = $1
		GROUP BY e.id
		ORDER BY e."createdAt" DESC
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("querying exams: %w", err)
	}
	defer rows.Close()

	var exams []model.Exam
	for rows.Next() {
		var e model.Exam
		var count int
		err := rows.Scan(
			&e.ID,
			&e.UserID,
			&e.Name,
			&e.Description,
			&e.ExamDate,
			&e.AvailableHoursPerDay,
			&e.Metadata,
			&e.CreatedAt,
			&e.UpdatedAt,
			&count,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning exam row: %w", err)
		}

		// Store subject count temporarily in dummy slice length for caller mapping
		if count > 0 {
			e.Subjects = make([]model.Subject, count)
		} else {
			e.Subjects = []model.Subject{}
		}

		exams = append(exams, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("reading exam rows: %w", err)
	}

	return exams, nil
}

// FindByID retrieves an exam and its full syllabus hierarchy (Subjects -> Topics -> Subtopics).
func (r *pgxExamRepository) FindByID(ctx context.Context, id, userID string) (*model.Exam, error) {
	// 1. Fetch Exam
	exam := &model.Exam{}
	err := r.pool.QueryRow(ctx,
		`SELECT id, "userId", name, description, "examDate", "availableHoursPerDay", metadata, "createdAt", "updatedAt"
		 FROM "Exam"
		 WHERE id = $1 AND "userId" = $2`,
		id, userID,
	).Scan(
		&exam.ID,
		&exam.UserID,
		&exam.Name,
		&exam.Description,
		&exam.ExamDate,
		&exam.AvailableHoursPerDay,
		&exam.Metadata,
		&exam.CreatedAt,
		&exam.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, fmt.Errorf("fetching exam: %w", err)
	}

	// 2. Fetch all subjects for this exam
	sRows, err := r.pool.Query(ctx,
		`SELECT id, "examId", "userId", name, description, "orderIndex", "createdAt", "updatedAt"
		 FROM "Subject"
		 WHERE "examId" = $1
		 ORDER BY "orderIndex" ASC`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching subjects: %w", err)
	}
	defer sRows.Close()

	var subjects []model.Subject
	var subjectIDs []string
	subjectIndexMap := make(map[string]int)

	for sRows.Next() {
		var s model.Subject
		err := sRows.Scan(
			&s.ID,
			&s.ExamID,
			&s.UserID,
			&s.Name,
			&s.Description,
			&s.OrderIndex,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning subject: %w", err)
		}
		s.Topics = []model.Topic{}
		subjectIndexMap[s.ID] = len(subjects)
		subjectIDs = append(subjectIDs, s.ID)
		subjects = append(subjects, s)
	}

	if len(subjects) == 0 {
		exam.Subjects = []model.Subject{}
		return exam, nil
	}

	// 3. Fetch all topics for these subjects
	tRows, err := r.pool.Query(ctx,
		`SELECT id, "subjectId", name, description, "orderIndex", "createdAt", "updatedAt"
		 FROM "Topic"
		 WHERE "subjectId" = ANY($1)
		 ORDER BY "orderIndex" ASC`,
		subjectIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching topics: %w", err)
	}
	defer tRows.Close()

	var topicIDs []string
	// Key: topicID -> pointer to Topic inside subjects[sIdx].Topics
	type topicRef struct {
		subjectIdx int
		topicIdx   int
	}
	topicLocMap := make(map[string]topicRef)

	for tRows.Next() {
		var t model.Topic
		err := tRows.Scan(
			&t.ID,
			&t.SubjectID,
			&t.Name,
			&t.Description,
			&t.OrderIndex,
			&t.CreatedAt,
			&t.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning topic: %w", err)
		}
		t.Subtopics = []model.Subtopic{}
		topicIDs = append(topicIDs, t.ID)

		sIdx := subjectIndexMap[t.SubjectID]
		subjects[sIdx].Topics = append(subjects[sIdx].Topics, t)
		tIdx := len(subjects[sIdx].Topics) - 1
		topicLocMap[t.ID] = topicRef{subjectIdx: sIdx, topicIdx: tIdx}
	}

	// 4. Fetch all subtopics for these topics if any topics exist
	if len(topicIDs) > 0 {
		stRows, err := r.pool.Query(ctx,
			`SELECT id, "topicId", name, description, "orderIndex", "createdAt", "updatedAt"
			 FROM "Subtopic"
			 WHERE "topicId" = ANY($1)
			 ORDER BY "orderIndex" ASC`,
			topicIDs,
		)
		if err != nil {
			return nil, fmt.Errorf("fetching subtopics: %w", err)
		}
		defer stRows.Close()

		for stRows.Next() {
			var st model.Subtopic
			err := stRows.Scan(
				&st.ID,
				&st.TopicID,
				&st.Name,
				&st.Description,
				&st.OrderIndex,
				&st.CreatedAt,
				&st.UpdatedAt,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning subtopic: %w", err)
			}

			if loc, ok := topicLocMap[st.TopicID]; ok {
				subjects[loc.subjectIdx].Topics[loc.topicIdx].Subtopics = append(
					subjects[loc.subjectIdx].Topics[loc.topicIdx].Subtopics,
					st,
				)
			}
		}
	}

	exam.Subjects = subjects
	return exam, nil
}

// Update updates selective fields on the Exam table.
func (r *pgxExamRepository) Update(ctx context.Context, id, userID string, updates map[string]interface{}) (*model.Exam, error) {
	if len(updates) == 0 {
		return r.FindByID(ctx, id, userID)
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for col, val := range updates {
		setClauses = append(setClauses, fmt.Sprintf(`"%s" = $%d`, col, argIdx))
		args = append(args, val)
		argIdx++
	}

	setClauses = append(setClauses, fmt.Sprintf(`"updatedAt" = $%d`, argIdx))
	args = append(args, time.Now().UTC())
	argIdx++

	args = append(args, id, userID)
	query := fmt.Sprintf(
		`UPDATE "Exam" SET %s WHERE id = $%d AND "userId" = $%d`,
		strings.Join(setClauses, ", "),
		argIdx,
		argIdx+1,
	)

	cmd, err := r.pool.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("updating exam: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return nil, nil // Not found or not owned by user
	}

	return r.FindByID(ctx, id, userID)
}

// Delete removes an Exam by ID. Foreign keys with ON DELETE CASCADE remove children.
func (r *pgxExamRepository) Delete(ctx context.Context, id, userID string) error {
	cmd, err := r.pool.Exec(ctx,
		`DELETE FROM "Exam" WHERE id = $1 AND "userId" = $2`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("deleting exam: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows // Signals 404
	}
	return nil
}

// ReplaceSyllabus reconciles the entire subject/topic/subtopic tree inside an ACID transaction.
// Matches the exact algorithm in Node.js exam.service.js:
//   - Identifies existing nodes by ID (or normalized name fallback)
//   - Updates existing nodes with new orderIndex / description
//   - Inserts newly added nodes
//   - Deletes removed nodes (cascading children)
func (r *pgxExamRepository) ReplaceSyllabus(ctx context.Context, examID, userID string, subjects []model.Subject) (*model.Exam, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting syllabus transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Safe: no-op if tx.Commit() is reached

	now := time.Now().UTC()

	// 1. Fetch all existing subjects for this exam inside the transaction
	sRows, err := tx.Query(ctx,
		`SELECT id, name FROM "Subject" WHERE "examId" = $1`,
		examID,
	)
	if err != nil {
		return nil, fmt.Errorf("fetching existing subjects: %w", err)
	}
	defer sRows.Close()

	existingSubjects := make(map[string]string) // normalizedName -> id
	existingSubjectIDs := make(map[string]bool) // id -> true
	for sRows.Next() {
		var sID, sName string
		if err := sRows.Scan(&sID, &sName); err != nil {
			return nil, err
		}
		existingSubjects[strings.ToLower(strings.TrimSpace(sName))] = sID
		existingSubjectIDs[sID] = true
	}
	sRows.Close()

	keepSubjectIDs := make(map[string]bool)

	// 2. Iterate and upsert Subjects
	for i, sub := range subjects {
		var subjectID string
		normName := strings.ToLower(strings.TrimSpace(sub.Name))

		if id, exists := existingSubjects[normName]; exists {
			subjectID = id
			// Update in place
			_, err = tx.Exec(ctx,
				`UPDATE "Subject" SET name = $1, description = $2, "orderIndex" = $3, "updatedAt" = $4 WHERE id = $5`,
				sub.Name, sub.Description, i, now, subjectID,
			)
			if err != nil {
				return nil, fmt.Errorf("updating subject %q: %w", sub.Name, err)
			}
		} else {
			// Insert new
			subjectID = uuid.New().String()
			_, err = tx.Exec(ctx,
				`INSERT INTO "Subject" (id, "examId", "userId", name, description, "orderIndex", "createdAt", "updatedAt")
				 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
				subjectID, examID, userID, sub.Name, sub.Description, i, now, now,
			)
			if err != nil {
				return nil, fmt.Errorf("inserting subject %q: %w", sub.Name, err)
			}
		}
		keepSubjectIDs[subjectID] = true

		// 3. Reconcile Topics for this subject
		tRows, err := tx.Query(ctx,
			`SELECT id, name FROM "Topic" WHERE "subjectId" = $1`,
			subjectID,
		)
		if err != nil {
			return nil, fmt.Errorf("fetching existing topics: %w", err)
		}
		existingTopics := make(map[string]string)
		for tRows.Next() {
			var tID, tName string
			if err := tRows.Scan(&tID, &tName); err != nil {
				tRows.Close()
				return nil, err
			}
			existingTopics[strings.ToLower(strings.TrimSpace(tName))] = tID
		}
		tRows.Close()

		keepTopicIDs := make(map[string]bool)

		for j, top := range sub.Topics {
			var topicID string
			normTopName := strings.ToLower(strings.TrimSpace(top.Name))

			if id, exists := existingTopics[normTopName]; exists {
				topicID = id
				_, err = tx.Exec(ctx,
					`UPDATE "Topic" SET name = $1, description = $2, "orderIndex" = $3, "updatedAt" = $4 WHERE id = $5`,
					top.Name, top.Description, j, now, topicID,
				)
				if err != nil {
					return nil, fmt.Errorf("updating topic %q: %w", top.Name, err)
				}
			} else {
				topicID = uuid.New().String()
				_, err = tx.Exec(ctx,
					`INSERT INTO "Topic" (id, "subjectId", name, description, "orderIndex", "createdAt", "updatedAt")
					 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
					topicID, subjectID, top.Name, top.Description, j, now, now,
				)
				if err != nil {
					return nil, fmt.Errorf("inserting topic %q: %w", top.Name, err)
				}
			}
			keepTopicIDs[topicID] = true

			// 4. Reconcile Subtopics for this topic
			stRows, err := tx.Query(ctx,
				`SELECT id, name FROM "Subtopic" WHERE "topicId" = $1`,
				topicID,
			)
			if err != nil {
				return nil, fmt.Errorf("fetching existing subtopics: %w", err)
			}
			existingSubtopics := make(map[string]string)
			for stRows.Next() {
				var stID, stName string
				if err := stRows.Scan(&stID, &stName); err != nil {
					stRows.Close()
					return nil, err
				}
				existingSubtopics[strings.ToLower(strings.TrimSpace(stName))] = stID
			}
			stRows.Close()

			keepSubtopicIDs := make(map[string]bool)

			for k, subtop := range top.Subtopics {
				var subtopicID string
				normSubtopName := strings.ToLower(strings.TrimSpace(subtop.Name))

				if id, exists := existingSubtopics[normSubtopName]; exists {
					subtopicID = id
					_, err = tx.Exec(ctx,
						`UPDATE "Subtopic" SET name = $1, description = $2, "orderIndex" = $3, "updatedAt" = $4 WHERE id = $5`,
						subtop.Name, subtop.Description, k, now, subtopicID,
					)
					if err != nil {
						return nil, fmt.Errorf("updating subtopic %q: %w", subtop.Name, err)
					}
				} else {
					subtopicID = uuid.New().String()
					_, err = tx.Exec(ctx,
						`INSERT INTO "Subtopic" (id, "topicId", name, description, "orderIndex", "createdAt", "updatedAt")
						 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
						subtopicID, topicID, subtop.Name, subtop.Description, k, now, now,
					)
					if err != nil {
						return nil, fmt.Errorf("inserting subtopic %q: %w", subtop.Name, err)
					}
				}
				keepSubtopicIDs[subtopicID] = true
			}

			// Delete subtopics not in payload
			for oldStID := range existingSubtopics {
				if !keepSubtopicIDs[oldStID] {
					if _, err := tx.Exec(ctx, `DELETE FROM "Subtopic" WHERE id = $1`, oldStID); err != nil {
						return nil, fmt.Errorf("deleting removed subtopic: %w", err)
					}
				}
			}
		}

		// Delete topics not in payload
		for oldTID := range existingTopics {
			if !keepTopicIDs[oldTID] {
				if _, err := tx.Exec(ctx, `DELETE FROM "Topic" WHERE id = $1`, oldTID); err != nil {
					return nil, fmt.Errorf("deleting removed topic: %w", err)
				}
			}
		}
	}

	// Delete subjects not in payload
	for oldSID := range existingSubjectIDs {
		if !keepSubjectIDs[oldSID] {
			if _, err := tx.Exec(ctx, `DELETE FROM "Subject" WHERE id = $1`, oldSID); err != nil {
				return nil, fmt.Errorf("deleting removed subject: %w", err)
			}
		}
	}

	// Commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing syllabus transaction: %w", err)
	}

	// Return the newly reconciled full tree
	return r.FindByID(ctx, examID, userID)
}
