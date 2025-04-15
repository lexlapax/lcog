package mmu

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

// RetrievalOptions configures the behavior of memory retrieval.
type RetrievalOptions struct {
	// MaxResults limits the number of records returned
	MaxResults int
	
	// Strategy determines the retrieval approach ("exact", "keyword", "semantic")
	Strategy string
	
	// IncludeMetadata determines whether to include metadata in the results
	IncludeMetadata bool
}

// DefaultRetrievalOptions returns the default options for memory retrieval.
func DefaultRetrievalOptions() RetrievalOptions {
	return RetrievalOptions{
		MaxResults:     10,
		Strategy:       "exact",
		IncludeMetadata: true,
	}
}

// MMU (Memory Management Unit) manages the flow of information between
// working memory and long-term memory.
type MMU interface {
	// EncodeToLTM stores information in long-term memory
	EncodeToLTM(ctx context.Context, dataToStore interface{}) (string, error)
	
	// RetrieveFromLTM retrieves information from long-term memory
	RetrieveFromLTM(ctx context.Context, query interface{}, options RetrievalOptions) ([]ltm.MemoryRecord, error)
	
	// ConsolidateLTM performs memory consolidation operations
	// This is a placeholder for more advanced functionality in later phases
	ConsolidateLTM(ctx context.Context, insight interface{}) error
}

// Config contains configuration options for the MMU.
type Config struct {
	// EnableLuaHooks determines whether to call Lua hooks during operations
	EnableLuaHooks bool
}

// DefaultConfig returns the default configuration for the MMU.
func DefaultConfig() Config {
	return Config{
		EnableLuaHooks: true,
	}
}

// MMUI is the implementation of the MMU interface.
type MMUI struct {
	// ltmStore is the long-term memory store
	ltmStore ltm.LTMStore
	
	// scriptEngine is the Lua scripting engine (optional)
	scriptEngine scripting.Engine
	
	// config contains configuration options
	config Config
}

// NewMMU creates a new MMU with the specified dependencies.
func NewMMU(
	ltmStore ltm.LTMStore,
	scriptEngine scripting.Engine,
	config Config,
) *MMUI {
	mmu := &MMUI{
		ltmStore:     ltmStore,
		scriptEngine: scriptEngine,
		config:       config,
	}
	
	log.Debug("Memory Management Unit (MMU) initialized", 
		"lua_hooks_enabled", config.EnableLuaHooks,
		"ltm_store_type", fmt.Sprintf("%T", ltmStore),
	)
	
	return mmu
}

// EncodeToLTM implements the MMU interface.
func (m *MMUI) EncodeToLTM(ctx context.Context, dataToStore interface{}) (string, error) {
	// Verify entity context
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return "", entity.ErrMissingEntityContext
	}

	// Prepare record to store
	record := ltm.MemoryRecord{
		EntityID:    entityCtx.EntityID,
		UserID:      entityCtx.UserID,
		AccessLevel: entity.SharedWithinEntity, // Default to shared
	}

	// Process the dataToStore based on its type
	switch data := dataToStore.(type) {
	case string:
		// If it's a simple string, store it as content
		record.Content = data
	case map[string]interface{}:
		// If it contains a "content" field, use that as the content
		if content, ok := data["content"].(string); ok {
			record.Content = content
		} else {
			// Otherwise, convert the entire map to JSON and store it
			jsonBytes, err := json.Marshal(data)
			if err != nil {
				return "", fmt.Errorf("failed to marshal data: %w", err)
			}
			record.Content = string(jsonBytes)
		}

		// Extract access level if provided
		if accessInt, ok := data["access_level"].(int); ok {
			record.AccessLevel = entity.AccessLevel(accessInt)
		}

		// Extract metadata if provided
		if meta, ok := data["metadata"].(map[string]interface{}); ok {
			record.Metadata = meta
		}
	default:
		// For any other type, try to marshal to JSON
		jsonBytes, err := json.Marshal(dataToStore)
		if err != nil {
			return "", fmt.Errorf("failed to marshal data: %w", err)
		}
		record.Content = string(jsonBytes)
	}

	// Ensure we have metadata
	if record.Metadata == nil {
		record.Metadata = make(map[string]interface{})
	}

	// Add timestamp metadata
	record.Metadata["encoded_at"] = time.Now().Format(time.RFC3339)

	// Generate a unique ID for the record
	record.ID = uuid.New().String()
	
	// Apply Lua hooks if enabled
	if m.config.EnableLuaHooks && m.scriptEngine != nil {
		// For Phase 1, just call the hook but don't use its result
		if m.scriptEngine != nil {
			m.scriptEngine.ExecuteFunction(ctx, "before_encode", record.Content)
		}
	}

	// Store the record in LTM
	memoryID, err := m.ltmStore.Store(ctx, record)
	
	// Apply after_encode hook if enabled
	if err == nil && m.config.EnableLuaHooks && m.scriptEngine != nil {
		if m.scriptEngine != nil {
			m.scriptEngine.ExecuteFunction(ctx, "after_encode", memoryID)
		}
	}
	
	return memoryID, err
}

// RetrieveFromLTM implements the MMU interface.
func (m *MMUI) RetrieveFromLTM(ctx context.Context, queryInput interface{}, options RetrievalOptions) ([]ltm.MemoryRecord, error) {
	// Verify entity context
	_, ok := entity.GetEntityContext(ctx)
	if !ok {
		return nil, entity.ErrMissingEntityContext
	}

	// Parse the query based on its type
	var query ltm.LTMQuery

	// Set limit from options
	query.Limit = options.MaxResults

	// Process the queryInput based on its type
	switch q := queryInput.(type) {
	case string:
		// If it's a simple string, use it as a text search
		query.Text = q
	case map[string]interface{}:
		// If it contains specific fields, extract them
		if text, ok := q["text"].(string); ok {
			query.Text = text
		}
		if exactMatch, ok := q["exact_match"].(map[string]interface{}); ok {
			query.ExactMatch = exactMatch
		}
		if filters, ok := q["filters"].(map[string]interface{}); ok {
			query.Filters = filters
		}
		if limit, ok := q["limit"].(float64); ok {
			query.Limit = int(limit)
		} else if limit, ok := q["limit"].(int); ok {
			query.Limit = limit
		}
	default:
		// For any other type, return an error
		return nil, fmt.Errorf("unsupported query type: %T", queryInput)
	}

	// Apply Lua hooks if enabled
	var err error
	if m.config.EnableLuaHooks && m.scriptEngine != nil {
		query, err = callBeforeRetrieveHook(ctx, m.scriptEngine, query)
		if err != nil {
			// Log the error but continue
			log.WarnContext(ctx, "Error in before_retrieve hook", "error", err)
		}
	}

	// Perform the retrieval
	results, err := m.ltmStore.Retrieve(ctx, query)
	if err != nil {
		return nil, err
	}

	// Apply after_retrieve hook if enabled
	if m.config.EnableLuaHooks && m.scriptEngine != nil {
		results, err = callAfterRetrieveHook(ctx, m.scriptEngine, results)
		if err != nil {
			// Log the error but continue
			log.WarnContext(ctx, "Error in after_retrieve hook", "error", err)
		}
	}

	// Remove metadata if not requested
	if !options.IncludeMetadata {
		for i := range results {
			results[i].Metadata = nil
		}
	}

	return results, nil
}

// ConsolidateLTM implements the MMU interface.
func (m *MMUI) ConsolidateLTM(ctx context.Context, insight interface{}) error {
	// This is a placeholder for future implementation
	// In a more advanced version, this would implement memory consolidation
	// strategies like summarization, connection discovery, etc.
	return nil
}