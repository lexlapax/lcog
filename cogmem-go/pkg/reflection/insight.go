package reflection

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lexlapax/cogmem/pkg/log"
)

// Insight represents a single insight generated during reflection
type Insight struct {
	// ID is a unique identifier for this insight
	ID string `json:"id"`
	
	// Type categorizes the insight (e.g., pattern, connection, gap, anomaly)
	Type string `json:"type"`
	
	// Description is a human-readable explanation of the insight
	Description string `json:"description"`
	
	// Confidence is a numerical measure of certainty (0.0-1.0)
	Confidence float64 `json:"confidence"`
	
	// RelatedMemoryIDs lists memory records related to this insight
	RelatedMemoryIDs []string `json:"related_memory_ids"`
	
	// Metadata contains additional structured data about the insight
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	
	// CreatedAt is when this insight was generated
	CreatedAt time.Time `json:"created_at"`
}

// NewInsight creates a new insight with a unique ID and current timestamp
func NewInsight(insightType, description string, confidence float64, relatedMemoryIDs []string) *Insight {
	return &Insight{
		ID:               uuid.New().String(),
		Type:             insightType,
		Description:      description,
		Confidence:       confidence,
		RelatedMemoryIDs: relatedMemoryIDs,
		Metadata:         make(map[string]interface{}),
		CreatedAt:        time.Now(),
	}
}

// InsightsResponse represents the structured format of insights from the LLM
type InsightsResponse struct {
	Insights []InsightData `json:"insights"`
}

// InsightData represents an individual insight in the LLM response
type InsightData struct {
	Type             string   `json:"type"`
	Description      string   `json:"description"`
	Confidence       float64  `json:"confidence"`
	RelatedMemoryIDs []string `json:"related_memory_ids"`
}

// ParseInsightsFromResponse parses the LLM response text into insight objects
func ParseInsightsFromResponse(response string) ([]*Insight, error) {
	// Handle empty response
	if response == "" {
		return nil, fmt.Errorf("empty response from reasoning engine")
	}
	
	// Try to parse the response as JSON
	var insightsResp InsightsResponse
	if err := json.Unmarshal([]byte(response), &insightsResp); err != nil {
		log.Warn("Failed to parse insights from JSON response", 
			"error", err, 
			"response", truncateString(response, 100))
		return nil, fmt.Errorf("failed to parse insights: %w", err)
	}
	
	// Create insight objects from the parsed data
	insights := make([]*Insight, 0, len(insightsResp.Insights))
	for _, data := range insightsResp.Insights {
		insight := NewInsight(
			data.Type,
			data.Description,
			data.Confidence,
			data.RelatedMemoryIDs,
		)
		insights = append(insights, insight)
	}
	
	return insights, nil
}

// truncateString truncates a string to the specified length and adds "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}