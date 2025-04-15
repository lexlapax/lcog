package reflection

import (
	"context"
	"fmt"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

// ReflectionModule defines the interface for self-reflection functionality
type ReflectionModule interface {
	// TriggerReflection initiates the reflection process 
	// and returns any insights that were generated
	TriggerReflection(ctx context.Context) ([]*Insight, error)
}

// Config contains configuration options for the ReflectionModule
type Config struct {
	// EnableLuaHooks determines whether to call Lua hooks during operations
	EnableLuaHooks bool
	
	// MaxMemoriesToAnalyze sets the maximum number of memories to include in analysis
	MaxMemoriesToAnalyze int
	
	// AnalysisTemperature sets the temperature for reasoning during analysis
	AnalysisTemperature float64
	
	// AnalysisMaxTokens sets the maximum tokens for reasoning responses
	AnalysisMaxTokens int
	
	// AnalysisModel specifies the model to use for analysis
	AnalysisModel string
}

// DefaultConfig returns the default configuration for the ReflectionModule
func DefaultConfig() Config {
	return Config{
		EnableLuaHooks:       true,
		MaxMemoriesToAnalyze: 50,
		AnalysisTemperature:  0.3,  // Lower temperature for more focused analysis
		AnalysisMaxTokens:    2048, // Larger context for thorough analysis
		AnalysisModel:        "",   // Empty means use the adapter's default
	}
}

// Module is the implementation of the ReflectionModule interface
type Module struct {
	// mmu is used to retrieve memories and store insights
	mmu mmu.MMU
	
	// reasoningEngine handles the analysis of memories
	reasoningEngine reasoning.Engine
	
	// scriptEngine is the Lua scripting engine (optional)
	scriptEngine scripting.Engine
	
	// config contains configuration options
	config Config
}

// NewReflectionModule creates a new reflection module with the specified dependencies
func NewReflectionModule(
	mmu mmu.MMU,
	reasoningEngine reasoning.Engine,
	scriptEngine scripting.Engine,
	config Config,
) *Module {
	module := &Module{
		mmu:             mmu,
		reasoningEngine: reasoningEngine,
		scriptEngine:    scriptEngine,
		config:          config,
	}
	
	log.Debug("Reflection Module initialized",
		"lua_hooks_enabled", config.EnableLuaHooks,
		"max_memories", config.MaxMemoriesToAnalyze,
		"analysis_temperature", config.AnalysisTemperature)
	
	return module
}

// TriggerReflection implements the ReflectionModule interface
func (m *Module) TriggerReflection(ctx context.Context) ([]*Insight, error) {
	// Verify entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}
	
	log.Info("Triggering reflection process",
		"entity_id", entityCtx.EntityID,
		"user_id", entityCtx.UserID)
	
	// Analyze memories to generate insights
	insights, err := m.analyzeMemories(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Failed to analyze memories for reflection",
			"error", err,
			"entity_id", entityCtx.EntityID)
		return nil, err
	}
	
	// Skip consolidation if no insights were generated
	if len(insights) == 0 {
		log.Info("No insights generated during reflection",
			"entity_id", entityCtx.EntityID)
		return insights, nil
	}
	
	// Apply before_consolidation Lua hook if enabled
	if m.config.EnableLuaHooks && m.scriptEngine != nil {
		result, err := m.scriptEngine.ExecuteFunction(ctx, beforeConsolidationFuncName, insights)
		if err != nil {
			log.WarnContext(ctx, "Error in before_consolidation hook", "error", err)
		} else if modifiedInsights, ok := result.([]*Insight); ok {
			insights = modifiedInsights
			log.Debug("Insights modified by before_consolidation hook",
				"count", len(insights))
		}
	}
	
	// Consolidate insights into long-term memory
	for _, insight := range insights {
		err := m.consolidateInsight(ctx, insight)
		if err != nil {
			log.WarnContext(ctx, "Failed to consolidate insight",
				"error", err,
				"insight_id", insight.ID,
				"insight_type", insight.Type)
		}
	}
	
	log.Info("Reflection process completed",
		"entity_id", entityCtx.EntityID,
		"insight_count", len(insights))
	
	return insights, nil
}

// consolidateInsight stores a single insight in long-term memory
func (m *Module) consolidateInsight(ctx context.Context, insight *Insight) error {
	if insight == nil {
		return fmt.Errorf("cannot consolidate nil insight")
	}
	
	// Prepare insight for storage
	insightData := map[string]interface{}{
		"content": insight.Description,
		"metadata": map[string]interface{}{
			"insight_type":        insight.Type,
			"insight_id":          insight.ID,
			"confidence":          insight.Confidence,
			"related_memory_ids":  insight.RelatedMemoryIDs,
			"created_at":          insight.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"source":              "reflection",
		},
	}
	
	// Add any custom metadata from the insight
	for k, v := range insight.Metadata {
		insightData["metadata"].(map[string]interface{})[k] = v
	}
	
	// Store the insight via the MMU
	err := m.mmu.ConsolidateLTM(ctx, insightData)
	if err != nil {
		return fmt.Errorf("failed to consolidate insight: %w", err)
	}
	
	log.Debug("Consolidated insight into LTM",
		"insight_id", insight.ID,
		"type", insight.Type,
		"confidence", insight.Confidence)
	
	return nil
}