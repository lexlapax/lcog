package reflection

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
)

// analyzeMemories retrieves relevant memories and analyzes them to generate insights
func (m *Module) analyzeMemories(ctx context.Context) ([]*Insight, error) {
	// Verify we have entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}
	
	// Retrieve recent memories to analyze
	memories, err := m.retrieveMemoriesForAnalysis(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve memories for analysis: %w", err)
	}
	
	if len(memories) == 0 {
		log.Info("No memories available for reflection analysis", 
			"entity_id", entityCtx.EntityID)
		return []*Insight{}, nil
	}
	
	log.Debug("Retrieved memories for reflection analysis", 
		"count", len(memories),
		"entity_id", entityCtx.EntityID)
	
	// Apply before_reflection_analysis hook if enabled
	if m.config.EnableLuaHooks && m.scriptEngine != nil {
		result, err := m.scriptEngine.ExecuteFunction(ctx, beforeReflectionAnalysisFuncName, memories)
		if err != nil {
			log.WarnContext(ctx, "Error in before_reflection_analysis hook", "error", err)
		} else if skip, ok := result.(bool); ok && skip {
			log.Info("Reflection analysis skipped by Lua hook")
			return []*Insight{}, nil
		}
	}
	
	// Format memories for analysis
	analysisPrompt, err := m.formatMemoriesForAnalysis(memories)
	if err != nil {
		return nil, fmt.Errorf("failed to format memories for analysis: %w", err)
	}
	
	// Call the reasoning engine to analyze the memories
	reasoningOpts := []reasoning.Option{
		reasoning.WithTemperature(m.config.AnalysisTemperature),
		reasoning.WithMaxTokens(m.config.AnalysisMaxTokens),
	}
	
	if m.config.AnalysisModel != "" {
		reasoningOpts = append(reasoningOpts, reasoning.WithModel(m.config.AnalysisModel))
	}
	
	log.Debug("Calling reasoning engine for reflection analysis",
		"temperature", m.config.AnalysisTemperature,
		"max_tokens", m.config.AnalysisMaxTokens,
		"model", m.config.AnalysisModel)
	
	analysisResult, err := m.reasoningEngine.Process(ctx, analysisPrompt, reasoningOpts...)
	if err != nil {
		return nil, fmt.Errorf("reasoning engine analysis failed: %w", err)
	}
	
	// Parse insights from the analysis result
	insights, err := ParseInsightsFromResponse(analysisResult)
	if err != nil {
		return nil, fmt.Errorf("failed to parse insights from analysis: %w", err)
	}
	
	log.Info("Generated insights from reflection analysis", 
		"insight_count", len(insights),
		"entity_id", entityCtx.EntityID)
	
	// Apply after_insight_generation hook if enabled
	if m.config.EnableLuaHooks && m.scriptEngine != nil {
		_, err := m.scriptEngine.ExecuteFunction(ctx, afterInsightGenerationFuncName, insights)
		if err != nil {
			log.WarnContext(ctx, "Error in after_insight_generation hook", "error", err)
		}
	}
	
	return insights, nil
}

// retrieveMemoriesForAnalysis fetches recent memories for reflection analysis
func (m *Module) retrieveMemoriesForAnalysis(ctx context.Context) ([]ltm.MemoryRecord, error) {
	// Construct a query for recent memories
	query := map[string]interface{}{
		"limit": m.config.MaxMemoriesToAnalyze,
	}
	
	// Configure retrieval options
	options := mmu.RetrievalOptions{
		MaxResults:      m.config.MaxMemoriesToAnalyze,
		Strategy:        "keyword",  // Use keyword search for now, not semantic
		IncludeMetadata: true,       // Include metadata for analysis
	}
	
	return m.mmu.RetrieveFromLTM(ctx, query, options)
}

// formatMemoriesForAnalysis prepares memories for input to the reasoning engine
func (m *Module) formatMemoriesForAnalysis(memories []ltm.MemoryRecord) (string, error) {
	// Basic format for the reflection analysis prompt
	prompt := fmt.Sprintf(`
You are performing a reflection analysis on a set of memories. Your task is to analyze these memories and identify insights, patterns, connections, gaps, or anomalies.

Below is a list of memories to analyze:

%s

Based on these memories, generate one or more insights. Each insight should identify a pattern, connection, gap, or anomaly in the memories.

Format your response as a JSON object with an "insights" array. Each insight should include:
- "type": The type of insight (pattern, connection, gap, anomaly)
- "description": A clear description of the insight
- "confidence": A confidence score between 0.0 and 1.0
- "related_memory_ids": An array of memory IDs related to this insight

Example response format:
{
  "insights": [
    {
      "type": "pattern",
      "description": "There is a recurring theme of...",
      "confidence": 0.85,
      "related_memory_ids": ["memory-id-1", "memory-id-2"]
    }
  ]
}

Provide your insights as valid JSON only, with no preamble or additional text.
`, m.formatMemoriesText(memories))

	return prompt, nil
}

// formatMemoriesText converts memory records to a text representation for the prompt
func (m *Module) formatMemoriesText(memories []ltm.MemoryRecord) string {
	var sb strings.Builder
	
	for i, memory := range memories {
		// Format timestamp
		timestamp := memory.CreatedAt.Format(time.RFC3339)
		
		// Add memory details
		sb.WriteString(fmt.Sprintf("Memory #%d (ID: %s, Created: %s):\n", i+1, memory.ID, timestamp))
		sb.WriteString(fmt.Sprintf("Content: %s\n", memory.Content))
		
		// Add metadata if available
		if len(memory.Metadata) > 0 {
			metadataJSON, err := json.Marshal(memory.Metadata)
			if err == nil {
				sb.WriteString(fmt.Sprintf("Metadata: %s\n", string(metadataJSON)))
			}
		}
		
		// Add separator between memories
		sb.WriteString("\n---\n\n")
	}
	
	return sb.String()
}