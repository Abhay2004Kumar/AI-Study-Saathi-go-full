-- AI Study Saathi Database Schema
-- Idempotent DDL for PostgreSQL with pgvector

CREATE EXTENSION IF NOT EXISTS vector;

-- 1. User
CREATE TABLE IF NOT EXISTS "User" (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL,
    "passwordHash" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 2. Exam
CREATE TABLE IF NOT EXISTS "Exam" (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES "User"(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    "examDate" TIMESTAMP(3) WITH TIME ZONE,
    "availableHoursPerDay" DOUBLE PRECISION,
    metadata JSONB,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "Exam_userId_idx" ON "Exam"("userId");

-- 3. Subject
CREATE TABLE IF NOT EXISTS "Subject" (
    id TEXT PRIMARY KEY,
    "examId" TEXT NOT NULL REFERENCES "Exam"(id) ON DELETE CASCADE,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    name TEXT NOT NULL,
    description TEXT,
    "orderIndex" INTEGER NOT NULL DEFAULT 0,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "Subject_examId_name_key" UNIQUE ("examId", name)
);
CREATE INDEX IF NOT EXISTS "Subject_examId_idx" ON "Subject"("examId");

-- 4. Topic
CREATE TABLE IF NOT EXISTS "Topic" (
    id TEXT PRIMARY KEY,
    "subjectId" TEXT NOT NULL REFERENCES "Subject"(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    "orderIndex" INTEGER NOT NULL DEFAULT 0,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "Topic_subjectId_name_key" UNIQUE ("subjectId", name)
);
CREATE INDEX IF NOT EXISTS "Topic_subjectId_idx" ON "Topic"("subjectId");

-- 5. Subtopic
CREATE TABLE IF NOT EXISTS "Subtopic" (
    id TEXT PRIMARY KEY,
    "topicId" TEXT NOT NULL REFERENCES "Topic"(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    "orderIndex" INTEGER NOT NULL DEFAULT 0,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "Subtopic_topicId_name_key" UNIQUE ("topicId", name)
);
CREATE INDEX IF NOT EXISTS "Subtopic_topicId_idx" ON "Subtopic"("topicId");

-- 6. Document
CREATE TABLE IF NOT EXISTS "Document" (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    "examId" TEXT REFERENCES "Exam"(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    "fileName" TEXT NOT NULL,
    "filePath" TEXT NOT NULL,
    "fileType" TEXT NOT NULL,
    "fileSize" INTEGER NOT NULL,
    category TEXT NOT NULL DEFAULT 'GENERAL',
    "fileHash" TEXT,
    "pageCount" INTEGER,
    "processingStatus" TEXT NOT NULL DEFAULT 'UPLOADED',
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "aiCategory" TEXT,
    "classificationConfidence" DOUBLE PRECISION
);
CREATE INDEX IF NOT EXISTS "Document_userId_fileHash_idx" ON "Document"("userId", "fileHash");
CREATE INDEX IF NOT EXISTS "Document_examId_idx" ON "Document"("examId");

-- 7. DocumentChunk
CREATE TABLE IF NOT EXISTS "DocumentChunk" (
    id TEXT PRIMARY KEY,
    "documentId" TEXT NOT NULL REFERENCES "Document"(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    "chunkIndex" INTEGER NOT NULL,
    embedding vector(3072),
    metadata JSONB,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "DocumentChunk_documentId_idx" ON "DocumentChunk"("documentId");

-- 8. ResourceExtraction
CREATE TABLE IF NOT EXISTS "ResourceExtraction" (
    id TEXT PRIMARY KEY,
    "documentId" TEXT UNIQUE NOT NULL REFERENCES "Document"(id) ON DELETE CASCADE,
    "resourceType" TEXT NOT NULL,
    summary TEXT,
    topics JSONB,
    questions JSONB,
    confidence DOUBLE PRECISION,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 9. ResourceTopicMapping
CREATE TABLE IF NOT EXISTS "ResourceTopicMapping" (
    id TEXT PRIMARY KEY,
    "documentId" TEXT NOT NULL REFERENCES "Document"(id) ON DELETE CASCADE,
    "topicId" TEXT NOT NULL REFERENCES "Topic"(id) ON DELETE CASCADE,
    "subtopicId" TEXT REFERENCES "Subtopic"(id) ON DELETE SET NULL,
    "startPage" INTEGER,
    "endPage" INTEGER,
    confidence DOUBLE PRECISION,
    status TEXT NOT NULL DEFAULT 'CONFIRMED',
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "ResourceTopicMapping_documentId_idx" ON "ResourceTopicMapping"("documentId");
CREATE INDEX IF NOT EXISTS "ResourceTopicMapping_topicId_idx" ON "ResourceTopicMapping"("topicId");
CREATE INDEX IF NOT EXISTS "ResourceTopicMapping_subtopicId_idx" ON "ResourceTopicMapping"("subtopicId");

-- 10. PyqQuestion
CREATE TABLE IF NOT EXISTS "PyqQuestion" (
    id TEXT PRIMARY KEY,
    "documentId" TEXT NOT NULL REFERENCES "Document"(id) ON DELETE CASCADE,
    "topicId" TEXT REFERENCES "Topic"(id) ON DELETE SET NULL,
    "subtopicId" TEXT REFERENCES "Subtopic"(id) ON DELETE SET NULL,
    "questionText" TEXT NOT NULL,
    "normalizedText" TEXT NOT NULL,
    options JSONB,
    "correctAnswer" TEXT,
    year INTEGER,
    confidence DOUBLE PRECISION,
    status TEXT NOT NULL DEFAULT 'CONFIRMED',
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "PyqQuestion_documentId_idx" ON "PyqQuestion"("documentId");
CREATE INDEX IF NOT EXISTS "PyqQuestion_topicId_idx" ON "PyqQuestion"("topicId");

-- 11. StudyProgress
CREATE TABLE IF NOT EXISTS "StudyProgress" (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    "subjectId" TEXT NOT NULL REFERENCES "Subject"(id),
    topic TEXT NOT NULL,
    score DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    "lastStudiedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 12. Quiz
CREATE TABLE IF NOT EXISTS "Quiz" (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    "subjectId" TEXT NOT NULL REFERENCES "Subject"(id),
    "topicId" TEXT REFERENCES "Topic"(id) ON DELETE SET NULL,
    "subtopicId" TEXT REFERENCES "Subtopic"(id) ON DELETE SET NULL,
    title TEXT NOT NULL,
    difficulty TEXT NOT NULL DEFAULT 'medium',
    "sourceDocumentIds" JSONB,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "Quiz_topicId_idx" ON "Quiz"("topicId");
CREATE INDEX IF NOT EXISTS "Quiz_subjectId_idx" ON "Quiz"("subjectId");

-- 13. QuizQuestion
CREATE TABLE IF NOT EXISTS "QuizQuestion" (
    id TEXT PRIMARY KEY,
    "quizId" TEXT NOT NULL REFERENCES "Quiz"(id) ON DELETE CASCADE,
    question TEXT NOT NULL,
    options JSONB NOT NULL,
    "correctAnswer" TEXT NOT NULL,
    explanation TEXT,
    difficulty TEXT NOT NULL DEFAULT 'medium'
);
CREATE INDEX IF NOT EXISTS "QuizQuestion_quizId_idx" ON "QuizQuestion"("quizId");

-- 14. QuizAttempt
CREATE TABLE IF NOT EXISTS "QuizAttempt" (
    id TEXT PRIMARY KEY,
    "quizId" TEXT NOT NULL REFERENCES "Quiz"(id) ON DELETE CASCADE,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    score DOUBLE PRECISION NOT NULL,
    "totalQuestions" INTEGER NOT NULL,
    "correctCount" INTEGER NOT NULL,
    "startedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "completedAt" TIMESTAMP(3) WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS "QuizAttempt_quizId_idx" ON "QuizAttempt"("quizId");
CREATE INDEX IF NOT EXISTS "QuizAttempt_userId_idx" ON "QuizAttempt"("userId");

-- 15. QuizAttemptAnswer
CREATE TABLE IF NOT EXISTS "QuizAttemptAnswer" (
    id TEXT PRIMARY KEY,
    "attemptId" TEXT NOT NULL REFERENCES "QuizAttempt"(id) ON DELETE CASCADE,
    "questionId" TEXT NOT NULL REFERENCES "QuizQuestion"(id) ON DELETE CASCADE,
    "selectedAnswer" TEXT,
    "isCorrect" BOOLEAN NOT NULL,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "QuizAttemptAnswer_attemptId_idx" ON "QuizAttemptAnswer"("attemptId");
CREATE INDEX IF NOT EXISTS "QuizAttemptAnswer_questionId_idx" ON "QuizAttemptAnswer"("questionId");

-- 16. TutorSession
CREATE TABLE IF NOT EXISTS "TutorSession" (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    subject TEXT NOT NULL,
    topic TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE',
    "conversationHistory" JSONB NOT NULL DEFAULT '[]',
    "currentConcept" TEXT,
    "weakAreas" JSONB NOT NULL DEFAULT '[]',
    "questionCount" INTEGER NOT NULL DEFAULT 0,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 17. StudyPlan
CREATE TABLE IF NOT EXISTS "StudyPlan" (
    id TEXT PRIMARY KEY,
    "examId" TEXT NOT NULL REFERENCES "Exam"(id) ON DELETE CASCADE,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    "startDate" TIMESTAMP(3) WITH TIME ZONE NOT NULL,
    "endDate" TIMESTAMP(3) WITH TIME ZONE NOT NULL,
    "generatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "StudyPlan_examId_userId_key" UNIQUE ("examId", "userId")
);
CREATE INDEX IF NOT EXISTS "StudyPlan_userId_idx" ON "StudyPlan"("userId");

-- 18. StudyPlanSession
CREATE TABLE IF NOT EXISTS "StudyPlanSession" (
    id TEXT PRIMARY KEY,
    "planId" TEXT NOT NULL REFERENCES "StudyPlan"(id) ON DELETE CASCADE,
    date TIMESTAMP(3) WITH TIME ZONE NOT NULL,
    "topicId" TEXT NOT NULL REFERENCES "Topic"(id) ON DELETE CASCADE,
    "subtopicId" TEXT REFERENCES "Subtopic"(id) ON DELETE SET NULL,
    "durationMinutes" INTEGER NOT NULL,
    "priorityScore" DOUBLE PRECISION NOT NULL,
    reason TEXT NOT NULL,
    completed BOOLEAN NOT NULL DEFAULT false,
    "completedAt" TIMESTAMP(3) WITH TIME ZONE,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "StudyPlanSession_planId_idx" ON "StudyPlanSession"("planId");
CREATE INDEX IF NOT EXISTS "StudyPlanSession_topicId_idx" ON "StudyPlanSession"("topicId");
CREATE INDEX IF NOT EXISTS "StudyPlanSession_date_idx" ON "StudyPlanSession"("date");

-- 19. Flashcard
CREATE TABLE IF NOT EXISTS "Flashcard" (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    "subjectId" TEXT NOT NULL REFERENCES "Subject"(id),
    "topicId" TEXT REFERENCES "Topic"(id) ON DELETE SET NULL,
    "subtopicId" TEXT REFERENCES "Subtopic"(id) ON DELETE SET NULL,
    front TEXT NOT NULL,
    back TEXT NOT NULL,
    hint TEXT,
    "normalizedFront" TEXT NOT NULL,
    "sourceDocumentIds" JSONB,
    "easeFactor" DOUBLE PRECISION NOT NULL DEFAULT 2.5,
    "intervalDays" INTEGER NOT NULL DEFAULT 0,
    "repetitions" INTEGER NOT NULL DEFAULT 0,
    "dueAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "lastReviewedAt" TIMESTAMP(3) WITH TIME ZONE,
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "Flashcard_userId_dueAt_idx" ON "Flashcard"("userId", "dueAt");
CREATE INDEX IF NOT EXISTS "Flashcard_topicId_idx" ON "Flashcard"("topicId");

-- 20. AssistantSession
CREATE TABLE IF NOT EXISTS "AssistantSession" (
    id TEXT PRIMARY KEY,
    "userId" TEXT NOT NULL REFERENCES "User"(id),
    "examId" TEXT NOT NULL REFERENCES "Exam"(id) ON DELETE CASCADE,
    messages JSONB NOT NULL DEFAULT '[]',
    "createdAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT "AssistantSession_examId_userId_key" UNIQUE ("examId", "userId")
);
CREATE INDEX IF NOT EXISTS "AssistantSession_userId_idx" ON "AssistantSession"("userId");

