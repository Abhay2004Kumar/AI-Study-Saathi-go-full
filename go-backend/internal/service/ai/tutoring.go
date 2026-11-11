// Package ai implements the Socratic Tutoring state machine and agent workflows.
package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aiStudySaathi/backend/internal/dto"
	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/pkg/apperror"
	"github.com/aiStudySaathi/backend/pkg/gemini"
)

const (
	MaxTutoringQuestions = 6
	NoLaTeXInstruction   = `Write in plain text only — no LaTeX (no $...$, \(...\), \rightarrow, \ge, etc.) and no markdown (no **bold**, no backticks). Use plain words or ordinary keyboard/unicode symbols for any formula or notation, e.g. "X -> A" or "X → A", ">=", "<="."`
)

// TutoringService manages interactive Socratic tutoring state machines.
type TutoringService interface {
	StartSession(ctx context.Context, userID, subject, topic string) (*dto.StartTutoringResponse, error)
	Respond(ctx context.Context, userID, sessionID, studentAnswer string) (*dto.RespondToTutorResponse, error)
	GetSession(ctx context.Context, userID, sessionID string) (*model.TutorSession, error)
}

type tutoringService struct {
	tutorRepo    repository.TutorSessionRepository
	chunkRepo    repository.DocumentChunkRepository
	geminiClient *gemini.Client
}

// NewTutoringService creates a new TutoringService.
func NewTutoringService(
	tutorRepo repository.TutorSessionRepository,
	chunkRepo repository.DocumentChunkRepository,
	geminiClient *gemini.Client,
) TutoringService {
	return &tutoringService{
		tutorRepo:    tutorRepo,
		chunkRepo:    chunkRepo,
		geminiClient: geminiClient,
	}
}

// HistoryMessage represents a single message in the tutoring dialogue.
type HistoryMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func formatHistory(history []HistoryMessage) string {
	if len(history) == 0 {
		return "No prior conversation."
	}
	var b strings.Builder
	for _, m := range history {
		role := "Student"
		if m.Role == "ai" {
			role = "Tutor"
		}
		b.WriteString(fmt.Sprintf("%s: %s\n", role, m.Content))
	}
	return b.String()
}

func parseExplanationAndQuestion(text string) (string, string) {
	upper := strings.ToUpper(text)
	expIdx := strings.Index(upper, "EXPLANATION:")
	qIdx := strings.Index(upper, "QUESTION:")

	if expIdx != -1 && qIdx != -1 && qIdx > expIdx {
		explanation := strings.TrimSpace(text[expIdx+len("EXPLANATION:") : qIdx])
		question := strings.TrimSpace(text[qIdx+len("QUESTION:"):])
		return explanation, question
	} else if expIdx != -1 {
		explanation := strings.TrimSpace(text[expIdx+len("EXPLANATION:"):])
		return explanation, "What did you understand from this?"
	} else if qIdx != -1 {
		question := strings.TrimSpace(text[qIdx+len("QUESTION:"):])
		return strings.TrimSpace(text[:qIdx]), question
	}
	return text, "What did you understand from this?"
}

func parseConcept(text string) string {
	upper := strings.ToUpper(text)
	cIdx := strings.Index(upper, "CONCEPT:")
	if cIdx != -1 {
		rest := text[cIdx+len("CONCEPT:"):]
		lines := strings.SplitN(rest, "\n", 2)
		return strings.TrimSpace(lines[0])
	}
	return ""
}

func (s *tutoringService) StartSession(ctx context.Context, userID, subject, topic string) (*dto.StartTutoringResponse, error) {
	// 1. Retrieve study material via vector search
	queryVector, err := s.geminiClient.EmbedContent(ctx, fmt.Sprintf("%s %s key concepts definitions", subject, topic))
	var studyContext string
	if err == nil {
		chunks, _ := s.chunkRepo.SimilaritySearch(ctx, repository.SimilaritySearchParams{
			UserID:      userID,
			QueryVector: queryVector,
			TopK:        8,
		})
		var b strings.Builder
		for _, ch := range chunks {
			b.WriteString(fmt.Sprintf("[%s]: %s\n\n", ch.Title, ch.Content))
		}
		studyContext = b.String()
	}

	currentConcept := fmt.Sprintf("Introduction to %s", topic)

	// 2. Explain concept and ask opening question
	prompt := fmt.Sprintf(`You are a friendly, encouraging AI tutor teaching a student about "%s" in %s.

STUDY MATERIAL:
%s

CURRENT CONCEPT: %s

Your task:
1. Explain the current concept clearly and concisely (2-3 sentences).
2. Ask ONE specific, targeted question to check the student's understanding.

%s

Format your response as:
EXPLANATION: <your explanation>
QUESTION: <your question>`, topic, subject, studyContext, currentConcept, NoLaTeXInstruction)

	respText, err := s.geminiClient.GenerateContent(ctx, prompt, "You are a warm, encouraging Socratic tutor.", 0.4)
	if err != nil {
		return nil, fmt.Errorf("generating tutor opening: %w", err)
	}

	explanation, question := parseExplanationAndQuestion(respText)
	aiMessage := fmt.Sprintf("%s\n\n❓ %s", explanation, question)
	initialHistory := []HistoryMessage{
		{Role: "ai", Content: aiMessage},
	}
	historyJSON, _ := json.Marshal(initialHistory)
	weakAreasJSON, _ := json.Marshal([]string{})

	session := &model.TutorSession{
		UserID:              userID,
		Subject:             subject,
		Topic:               topic,
		Status:              "ACTIVE",
		CurrentConcept:      &currentConcept,
		ConversationHistory: historyJSON,
		WeakAreas:           weakAreasJSON,
		QuestionCount:       1,
	}

	if err := s.tutorRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("creating tutor session: %w", err)
	}

	return &dto.StartTutoringResponse{
		SessionID:     session.ID,
		AIMessage:     aiMessage,
		QuestionCount: 1,
		SessionStatus: "ACTIVE",
	}, nil
}

type tutorEvaluation struct {
	Evaluation string `json:"evaluation"` // strong or weak
	Feedback   string `json:"feedback"`
	WeakArea   string `json:"weakArea"`
}

func (s *tutoringService) Respond(ctx context.Context, userID, sessionID, studentAnswer string) (*dto.RespondToTutorResponse, error) {
	session, err := s.tutorRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying tutor session: %w", err)
	}
	if session == nil {
		return nil, apperror.NotFound("Session not found")
	}

	var history []HistoryMessage
	_ = json.Unmarshal(session.ConversationHistory, &history)

	var weakAreas []string
	_ = json.Unmarshal(session.WeakAreas, &weakAreas)

	if session.Status == "COMPLETED" {
		return &dto.RespondToTutorResponse{
			AIMessage:     "This tutoring session has already been completed.",
			QuestionCount: session.QuestionCount,
			SessionStatus: "COMPLETED",
			WeakAreas:     weakAreas,
		}, nil
	}

	currentConcept := ""
	if session.CurrentConcept != nil {
		currentConcept = *session.CurrentConcept
	}

	// 1. Evaluate student's answer
	evalPrompt := fmt.Sprintf(`You are an AI tutor evaluating a student's answer.

TOPIC: %s
CURRENT CONCEPT: %s
CONVERSATION HISTORY:
%s
STUDENT'S ANSWER: "%s"

%s

Evaluate the answer. Respond with ONLY a JSON object:
{
  "evaluation": "strong" or "weak",
  "feedback": "<brief encouraging feedback on their answer>",
  "weakArea": "<if weak, what specific part they didn't understand, else empty string>"
}`, session.Topic, currentConcept, formatHistory(history), studentAnswer, NoLaTeXInstruction)

	var eval tutorEvaluation
	err = s.geminiClient.GenerateJSON(ctx, evalPrompt, "You evaluate student understanding with empathy.", 0.2, &eval)
	if err != nil {
		eval = tutorEvaluation{
			Evaluation: "weak",
			Feedback:   "Good effort! Let me explain that again.",
			WeakArea:   currentConcept,
		}
	}

	if eval.Evaluation == "weak" && eval.WeakArea != "" {
		weakAreas = append(weakAreas, eval.WeakArea)
	}

	history = append(history,
		HistoryMessage{Role: "student", Content: studentAnswer},
		HistoryMessage{Role: "ai", Content: eval.Feedback},
	)

	newQuestionCount := session.QuestionCount + 1

	// Check if maximum questions reached -> complete session
	if newQuestionCount >= MaxTutoringQuestions {
		weakSummary := "Great job! No significant weak areas identified."
		if len(weakAreas) > 0 {
			weakSummary = fmt.Sprintf("Areas to review: %s", strings.Join(weakAreas, ", "))
		}

		finalMessage := fmt.Sprintf(`🎉 Session Complete!

You have completed your tutoring session on %s.

Summary:
- Questions answered: %d
- %s

Keep up the great work! Consider uploading more notes to deepen your understanding.`, session.Topic, newQuestionCount, weakSummary)

		history = append(history, HistoryMessage{Role: "ai", Content: finalMessage})
		histJSON, _ := json.Marshal(history)
		weakJSON, _ := json.Marshal(weakAreas)

		_, _ = s.tutorRepo.Update(ctx, session.ID, map[string]interface{}{
			"status":              "COMPLETED",
			"currentConcept":      session.CurrentConcept,
			"conversationHistory": histJSON,
			"weakAreas":           weakJSON,
			"questionCount":       newQuestionCount,
		})

		return &dto.RespondToTutorResponse{
			AIMessage:     finalMessage,
			QuestionCount: newQuestionCount,
			SessionStatus: "COMPLETED",
			WeakAreas:     weakAreas,
		}, nil
	}

	// 2. Branch: If weak -> explainAgain; If strong -> advanceConcept
	var nextConcept = currentConcept
	var turnMessage string

	if eval.Evaluation == "weak" {
		prompt := fmt.Sprintf(`You are a patient AI tutor. The student struggled with: "%s".

Re-explain this concept using a different, simpler approach or a real-world analogy.
Then ask a simpler version of the question.

%s

Format:
EXPLANATION: <simpler explanation>
QUESTION: <simpler question>`, currentConcept, NoLaTeXInstruction)

		turnResp, _ := s.geminiClient.GenerateContent(ctx, prompt, "You are a gentle, patient tutor.", 0.4)
		exp, q := parseExplanationAndQuestion(turnResp)
		turnMessage = fmt.Sprintf("Let me try explaining that differently!\n\n%s\n\n❓ %s", exp, q)
	} else {
		// Student was strong -> advance to harder concept
		prompt := fmt.Sprintf(`You are an AI tutor. The student understood "%s" well.

TOPIC: %s
CONVERSATION HISTORY:
%s

Move to the NEXT concept in "%s". Explain it at a slightly harder level.
Then ask a challenging question.

%s

Format:
CONCEPT: <name of the new concept>
EXPLANATION: <harder explanation>
QUESTION: <challenging question>`, currentConcept, session.Topic, formatHistory(history), session.Topic, NoLaTeXInstruction)

		turnResp, _ := s.geminiClient.GenerateContent(ctx, prompt, "You challenge students to level up.", 0.4)
		if c := parseConcept(turnResp); c != "" {
			nextConcept = c
		}
		exp, q := parseExplanationAndQuestion(turnResp)
		turnMessage = fmt.Sprintf("Great work! Let's move on to: %s\n\n%s\n\n❓ %s", nextConcept, exp, q)
	}

	history = append(history, HistoryMessage{Role: "ai", Content: turnMessage})
	histJSON, _ := json.Marshal(history)
	weakJSON, _ := json.Marshal(weakAreas)

	_, err = s.tutorRepo.Update(ctx, session.ID, map[string]interface{}{
		"status":              "ACTIVE",
		"currentConcept":      &nextConcept,
		"conversationHistory": histJSON,
		"weakAreas":           weakJSON,
		"questionCount":       newQuestionCount,
	})
	if err != nil {
		return nil, fmt.Errorf("updating tutor turn: %w", err)
	}

	return &dto.RespondToTutorResponse{
		AIMessage:     turnMessage,
		QuestionCount: newQuestionCount,
		SessionStatus: "ACTIVE",
		WeakAreas:     weakAreas,
	}, nil
}

func (s *tutoringService) GetSession(ctx context.Context, userID, sessionID string) (*model.TutorSession, error) {
	session, err := s.tutorRepo.FindByID(ctx, sessionID, userID)
	if err != nil {
		return nil, fmt.Errorf("verifying tutor session: %w", err)
	}
	if session == nil {
		return nil, apperror.NotFound("Session not found")
	}
	return session, nil
}
