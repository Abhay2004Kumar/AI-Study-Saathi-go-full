package dto

import "time"

// TopicResourceDocument represents the document details embedded in topic resources.
type TopicResourceDocument struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Category         string `json:"category"`
	FileName         string `json:"fileName"`
	ProcessingStatus string `json:"processingStatus"`
}

// TopicResourceSubtopic represents the subtopic details embedded in topic resources.
type TopicResourceSubtopic struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// TopicResourceMappingItem is an individual mapping item in the topic resources view.
type TopicResourceMappingItem struct {
	ID         string                 `json:"id"`
	StartPage  *int                   `json:"startPage"`
	EndPage    *int                   `json:"endPage"`
	Confidence *float64               `json:"confidence"`
	Status     string                 `json:"status"`
	Subtopic   *TopicResourceSubtopic `json:"subtopic"`
	Document   TopicResourceDocument  `json:"document"`
}

// TopicResourcesResponse matches the exact shape returned by Node.js getTopicResources.
type TopicResourcesResponse struct {
	Topic    TopicReference             `json:"topic"`
	Notes    []TopicResourceMappingItem `json:"notes"`
	Pyqs     []TopicResourceMappingItem `json:"pyqs"`
	Syllabus []TopicResourceMappingItem `json:"syllabus"`
}

// EnrichedMappingItem is returned by getDocumentMappings and mapDocumentTopics.
type EnrichedMappingItem struct {
	ID         string                 `json:"id"`
	DocumentID string                 `json:"documentId"`
	TopicID    string                 `json:"topicId"`
	SubtopicID *string                `json:"subtopicId"`
	StartPage  *int                   `json:"startPage"`
	EndPage    *int                   `json:"endPage"`
	Confidence *float64               `json:"confidence"`
	Status     string                 `json:"status"`
	CreatedAt  time.Time              `json:"createdAt"`
	Topic      *TopicReference        `json:"topic"`
	Subtopic   *TopicResourceSubtopic `json:"subtopic"`
}

// UpdateMappingStatusRequest defines the payload for PATCH /api/mappings/:mappingId.
type UpdateMappingStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

