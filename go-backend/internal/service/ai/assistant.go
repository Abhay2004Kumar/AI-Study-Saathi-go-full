// Package ai implements the ReAct AI Study Assistant with server-side scoped tools.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/internal/service/scoring"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/gemini"
)

const (
	MaxHistoryMessages = 20
	AssistantSystemPrompt = `You are the AI Study Assistant inside a student's exam-prep app, helping them prepare for one specific exam.

You have access to tools that read the student's OWN real data — their quiz performance, their study plan, how often topics have appeared in past papers they uploaded, and their own uploaded notes. ALWAYS rely on tool data instead of guessing when the student asks about their progress, their plan, a topic's importance, or anything that should come from their materials. Never invent facts, scores, or study-plan details.

Keep answers concise and encouraging, and ground every claim about the student's performance or plan in what the tools return.

Write in plain text only — no LaTeX and no markdown bold or backticks.`
)

// AssistantService defines operations for the exam-scoped AI Study Assistant.
type AssistantService interface {
	GetSession(ctx context.Context, userID, examID string) (*model.AssistantSession, error)
	SendMessage(ctx context.Context, userID, examID, message string) (*dto.SendAssistantMessageResponse, error)
}

type assistantService struct {
	assistantRepo repository.AssistantSessionRepository
	examRepo      repository.ExamRepository
	pyqRepo       repository.PyqRepository
	ragService    RAGService
	masterySvc    service.MasteryService
	planSvc       service.StudyPlanService
	cardSvc       service.FlashcardService
	geminiClient  *gemini.Client
}

// NewAssistantService creates a new AssistantService.
func NewAssistantService(
	assistantRepo repository.AssistantSessionRepository,
	examRepo repository.ExamRepository,
	pyqRepo repository.PyqRepository,
	ragService RAGService,
	masterySvc service.MasteryService,
	planSvc service.StudyPlanService,
	cardSvc service.FlashcardService,
	geminiClient *gemini.Client,
) AssistantService {
	return &assistantService{
		assistantRepo: assistantRepo,
		examRepo:      examRepo,
		pyqRepo:       pyqRepo,
		ragService:    ragService,
		masterySvc:    masterySvc,
		planSvc:       planSvc,
		cardSvc:       cardSvc,
		geminiClient:  geminiClient,
	}
}

// SessionMessage represents a message stored in AssistantSession.
type SessionMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	ToolsUsed []string  `json:"toolsUsed,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

func (s *assistantService) GetSession(ctx context.Context, userID, examID string) (*model.AssistantSession, error) {
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	return s.assistantRepo.FindByExamAndUser(ctx, examID, userID)
}

func (s *assistantService) SendMessage(ctx context.Context, userID, examID, message string) (*dto.SendAssistantMessageResponse, error) {
	exam, err := s.examRepo.FindByID(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying exam: %w", err)
	}
	if exam == nil {
		return nil, apperror.NotFound("Exam not found")
	}

	// 1. Fetch prior messages
	existing, err := s.assistantRepo.FindByExamAndUser(ctx, examID, userID)
	if err != nil {
		return nil, fmt.Errorf("fetching assistant session: %w", err)
	}

	var priorMessages []SessionMessage
	if existing != nil && len(existing.Messages) > 0 {
		_ = json.Unmarshal(existing.Messages, &priorMessages)
	}

	// 2. Determine necessary tools based on student query intent
	lowerMsg := strings.ToLower(message)
	var toolsUsed []string
	var toolContext strings.Builder

	// Tool 1: get_todays_study_plan
	if strings.Contains(lowerMsg, "today") || strings.Contains(lowerMsg, "plan") || strings.Contains(lowerMsg, "schedule") {
		toolsUsed = append(toolsUsed, "get_todays_study_plan")
		activePlan, _ := s.planSvc.GetActivePlan(ctx, userID, examID)
		if activePlan == nil {
			toolContext.WriteString("[Tool get_todays_study_plan]: No study plan has been generated for this exam yet.\n\n")
		} else {
			todayKey := time.Now().UTC().Format("2006-01-02")
			var todaysSessions []model.StudyPlanSession
			for _, sess := range activePlan.Sessions {
				if sess.Date.Format("2006-01-02") == todayKey {
					todaysSessions = append(todaysSessions, sess)
				}
			}
			if len(todaysSessions) == 0 {
				toolContext.WriteString("[Tool get_todays_study_plan]: Nothing is scheduled for today in the student's study plan.\n\n")
			} else {
				data, _ := json.Marshal(todaysSessions)
				toolContext.WriteString(fmt.Sprintf("[Tool get_todays_study_plan]: %s\n\n", string(data)))
			}
		}
	}

	// Tool 2: get_mastery_overview
	if strings.Contains(lowerMsg, "mastery") || strings.Contains(lowerMsg, "weak") || strings.Contains(lowerMsg, "strong") || strings.Contains(lowerMsg, "progress") || strings.Contains(lowerMsg, "readiness") {
		toolsUsed = append(toolsUsed, "get_mastery_overview")
		masteryList, _ := s.masterySvc.GetExamMastery(ctx, userID, examID)
		if len(masteryList) == 0 {
			toolContext.WriteString("[Tool get_mastery_overview]: This exam has no syllabus topics yet.\n\n")
		} else {
			data, _ := json.Marshal(masteryList)
			toolContext.WriteString(fmt.Sprintf("[Tool get_mastery_overview]: %s\n\n", string(data)))
		}
	}

	// Tool 3: get_due_flashcard_count
	if strings.Contains(lowerMsg, "card") || strings.Contains(lowerMsg, "flashcard") || strings.Contains(lowerMsg, "due") || strings.Contains(lowerMsg, "review") {
		toolsUsed = append(toolsUsed, "get_due_flashcard_count")
		dueCards, _ := s.cardSvc.GetDueFlashcards(ctx, userID, examID, 200)
		toolContext.WriteString(fmt.Sprintf("[Tool get_due_flashcard_count]: %d flashcard(s) are due for revision right now.\n\n", len(dueCards)))
	}

	// Tool 4: get_topic_pyq_importance
	if strings.Contains(lowerMsg, "pyq") || strings.Contains(lowerMsg, "previous year") || strings.Contains(lowerMsg, "importance") || strings.Contains(lowerMsg, "often") {
		toolsUsed = append(toolsUsed, "get_topic_pyq_importance")
		for _, sub := range exam.Subjects {
			for _, top := range sub.Topics {
				if strings.Contains(lowerMsg, strings.ToLower(top.Name)) {
					pyqs, _ := s.pyqRepo.FindByTopic(ctx, top.ID, userID, true)
					var years []int
					for _, q := range pyqs {
						if q.Question.Year != nil {
							years = append(years, *q.Question.Year)
						}
					}
					freq := scoring.ComputeTopicFrequency(years, time.Now().Year(), nil)
					toolContext.WriteString(fmt.Sprintf("[Tool get_topic_pyq_importance on '%s']: Level=%s, Score=%.2f, YearsSeen=%v, QuestionCount=%d\n\n",
						top.Name, freq.Level, freq.Score, freq.YearsSeen, freq.QuestionCount))
					break
				}
			}
		}
	}

	// Tool 5: search_study_material (fallback or explicit concept inquiry)
	if toolContext.Len() == 0 || strings.Contains(lowerMsg, "explain") || strings.Contains(lowerMsg, "what is") || strings.Contains(lowerMsg, "how to") {
		toolsUsed = append(toolsUsed, "search_study_material")
		ragResult, _ := s.ragService.AskQuestion(ctx, userID, message, RAGFilters{ExamID: &examID})
		if ragResult != nil && ragResult.FoundInMaterials {
			toolContext.WriteString(fmt.Sprintf("[Tool search_study_material]: %s\n\n", ragResult.Answer))
		} else {
			toolContext.WriteString("[Tool search_study_material]: Nothing relevant was found in the materials uploaded for this exam.\n\n")
		}
	}

	// 3. Format history
	startIdx := 0
	if len(priorMessages) > MaxHistoryMessages {
		startIdx = len(priorMessages) - MaxHistoryMessages
	}
	recentHistory := priorMessages[startIdx:]

	var historyBuilder strings.Builder
	for _, m := range recentHistory {
		historyBuilder.WriteString(fmt.Sprintf("%s: %s\n", strings.Title(m.Role), m.Content))
	}

	// 4. Generate final assistant reply
	prompt := fmt.Sprintf(`%s

TOOL DATA EXECUTED ON STUDENT'S DATA:
%s

RECENT CONVERSATION HISTORY:
%s

STUDENT MESSAGE:
%s

Provide a helpful, direct response to the student using the tool data above.`,
		AssistantSystemPrompt, toolContext.String(), historyBuilder.String(), message)

	reply, err := s.geminiClient.GenerateContent(ctx, prompt, "You are a concise, supportive study assistant.", 0.3)
	if err != nil {
		return nil, fmt.Errorf("generating assistant reply: %w", err)
	}

	// Deduplicate tool names
	toolsUsedSet := make(map[string]bool)
	var uniqueTools []string
	for _, t := range toolsUsed {
		if !toolsUsedSet[t] {
			toolsUsedSet[t] = true
			uniqueTools = append(uniqueTools, t)
		}
	}

	// 5. Update and persist session messages
	now := time.Now().UTC()
	updatedMessages := append(priorMessages,
		SessionMessage{Role: "user", Content: message, CreatedAt: now},
		SessionMessage{Role: "assistant", Content: reply, ToolsUsed: uniqueTools, CreatedAt: now},
	)
	updatedJSON, _ := json.Marshal(updatedMessages)

	session, err := s.assistantRepo.Upsert(ctx, examID, userID, updatedJSON)
	if err != nil {
		return nil, fmt.Errorf("persisting assistant session: %w", err)
	}

	return &dto.SendAssistantMessageResponse{
		Reply:     reply,
		ToolsUsed: uniqueTools,
		SessionID: session.ID,
	}, nil
}

