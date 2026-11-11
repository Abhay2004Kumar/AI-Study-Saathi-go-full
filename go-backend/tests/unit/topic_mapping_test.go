package service_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/aiStudySaathi/backend/internal/model"
	"github.com/aiStudySaathi/backend/internal/repository"
	"github.com/aiStudySaathi/backend/internal/service"
	"github.com/aiStudySaathi/backend/pkg/response"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// TestResponse_NullDataSerializesAsNull verifies that response.OK(c, nil)
// outputs {"success":true,"data":null} matching Node.js res.json({ success: true, data: null })
// rather than omitting the data key.
func TestResponse_NullDataSerializesAsNull(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	response.OK(c, nil)

	body := w.Body.String()
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("failed to parse json: %v", err)
	}

	val, exists := raw["data"]
	if !exists {
		t.Fatalf("expected 'data' key to exist in json response, got: %s", body)
	}
	if val != nil {
		t.Fatalf("expected 'data' to be null, got: %v", val)
	}
	if raw["success"] != true {
		t.Fatalf("expected 'success' to be true")
	}
}

// MockTopicMappingRepository implements repository.TopicMappingRepository for testing.
type MockTopicMappingRepository struct {
	items []repository.TopicResourceItem
}

func (m *MockTopicMappingRepository) ReplaceForDocument(ctx context.Context, documentID string, mappings []model.ResourceTopicMapping) error {
	return nil
}

func (m *MockTopicMappingRepository) FindByDocument(ctx context.Context, documentID string) ([]repository.TopicResourceItem, error) {
	return m.items, nil
}

func (m *MockTopicMappingRepository) FindByTopicWithRelations(ctx context.Context, topicID, userID string) ([]repository.TopicResourceItem, error) {
	return m.items, nil
}

func (m *MockTopicMappingRepository) FindDocumentIDsByTopic(ctx context.Context, topicID string, excludeRejected bool) ([]string, error) {
	return nil, nil
}

func (m *MockTopicMappingRepository) FindByID(ctx context.Context, id, userID string) (*model.ResourceTopicMapping, error) {
	return &model.ResourceTopicMapping{ID: id, Status: "PENDING_REVIEW"}, nil
}

func (m *MockTopicMappingRepository) UpdateStatus(ctx context.Context, id, status string) (*model.ResourceTopicMapping, error) {
	return &model.ResourceTopicMapping{ID: id, Status: status}, nil
}

func TestTopicMappingService_GetTopicResources_EmptyArrays(t *testing.T) {
	examRepo := NewMockExamRepository()
	examRepo.examsByID["exam-1"] = &model.Exam{
		ID:     "exam-1",
		UserID: "user-1",
		Subjects: []model.Subject{
			{
				ID:   "sub-1",
				Name: "Computer Networks",
				Topics: []model.Topic{
					{ID: "top-1", Name: "Routing Protocols"},
				},
			},
		},
	}

	mockMappingRepo := &MockTopicMappingRepository{items: []repository.TopicResourceItem{}}
	svc := service.NewTopicMappingService(mockMappingRepo, examRepo, nil, nil, nil)

	res, err := svc.GetTopicResources(context.Background(), "user-1", "exam-1", "top-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.Topic.ID != "top-1" || res.Topic.Name != "Routing Protocols" {
		t.Errorf("unexpected topic: %+v", res.Topic)
	}

	// Verify slices serialize to [] rather than null
	jsonBytes, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var jsonMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonMap); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	for _, key := range []string{"notes", "pyqs", "syllabus"} {
		arr, ok := jsonMap[key].([]interface{})
		if !ok {
			t.Errorf("expected '%s' to be a JSON array, got: %v", key, jsonMap[key])
		}
		if len(arr) != 0 {
			t.Errorf("expected empty array for '%s', got length %d", key, len(arr))
		}
	}
}

func TestTopicMappingService_UpdateMappingStatus(t *testing.T) {
	mockMappingRepo := &MockTopicMappingRepository{}
	svc := service.NewTopicMappingService(mockMappingRepo, nil, nil, nil, nil)

	// Valid status
	updated, err := svc.UpdateMappingStatus(context.Background(), "user-1", "map-1", "CONFIRMED")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Status != "CONFIRMED" {
		t.Errorf("expected status CONFIRMED, got %s", updated.Status)
	}

	// Invalid status
	_, err = svc.UpdateMappingStatus(context.Background(), "user-1", "map-1", "INVALID")
	if err == nil {
		t.Fatalf("expected error for invalid status")
	}
}
