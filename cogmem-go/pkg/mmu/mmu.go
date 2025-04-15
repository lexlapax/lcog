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
	"github.com/lexlapax/cogmem/pkg/reasoning"
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
	
	// EnableVectorOperations determines whether to use vector operations when available
	EnableVectorOperations bool
	
	// WorkingMemoryLimit sets the maximum number of records in working memory
	// before overflow triggers LTM encoding
	WorkingMemoryLimit int
}

// DefaultConfig returns the default configuration for the MMU.
func DefaultConfig() Config {
	return Config{
		EnableLuaHooks:        true,
		EnableVectorOperations: true,
		WorkingMemoryLimit:    100,
	}
}

// MMUI is the implementation of the MMU interface.
type MMUI struct {
	// ltmStore is the long-term memory store
	ltmStore ltm.LTMStore
	
	// reasoningEngine handles embedding generation and language processing
	reasoningEngine reasoning.Engine
	
	// scriptEngine is the Lua scripting engine (optional)
	scriptEngine scripting.Engine
	
	// config contains configuration options
	config Config
	
	// workingMemory holds records not yet committed to LTM
	// This is a simple implementation for Phase 2
	workingMemory []ltm.MemoryRecord
}

// NewMMU creates a new MMU with the specified dependencies.
func NewMMU(
	ltmStore ltm.LTMStore,
	reasoningEngine reasoning.Engine,
	scriptEngine scripting.Engine,
	config Config,
) *MMUI {
	mmu := &MMUI{
		ltmStore:        ltmStore,
		reasoningEngine: reasoningEngine,
		scriptEngine:    scriptEngine,
		config:          config,
		workingMemory:   make([]ltm.MemoryRecord, 0, config.WorkingMemoryLimit),
	}
	
	// Determine if the LTM store supports vector operations
	supportsVectors := false
	if config.EnableVectorOperations {
		if vectorStore, ok := ltmStore.(ltm.VectorCapableLTMStore); ok {
			supportsVectors = vectorStore.SupportsVectorSearch()
		}
	}
	
	log.Debug("Memory Management Unit (MMU) initialized", 
		"lua_hooks_enabled", config.EnableLuaHooks,
		"vector_operations", supportsVectors,
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
		
		// Extract embedding if provided
		if embedding, ok := data["embedding"].([]float32); ok {
			record.Embedding = embedding
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
	
	// Check if we need to generate embeddings
	needsEmbedding := m.shouldGenerateEmbedding(record)
	
	// Apply before_embedding Lua hook if enabled
	if needsEmbedding && m.config.EnableLuaHooks && m.scriptEngine != nil {
		result, err := m.scriptEngine.ExecuteFunction(ctx, beforeEmbeddingFuncName, record.Content)
		if err == nil {
			// If the hook returns false, skip embedding generation
			if skip, ok := result.(bool); ok && skip {
				needsEmbedding = false
				log.Debug("Embedding generation skipped by Lua hook")
			}
		}
	}
	
	// Generate embedding if needed and possible
	if needsEmbedding && m.reasoningEngine != nil {
		embeddings, err := m.reasoningEngine.GenerateEmbeddings(ctx, []string{record.Content})
		if err != nil {
			// Log the error but continue without embedding
			log.WarnContext(ctx, "Failed to generate embedding", 
				"error", err,
				"content_length", len(record.Content))
		} else if len(embeddings) > 0 {
			record.Embedding = embeddings[0]
			log.Debug("Generated embedding for content", 
				"embedding_dimensions", len(record.Embedding),
				"content_preview", truncateString(record.Content, 30))
		}
	}
	
	// Apply before_encode Lua hook if enabled
	if m.config.EnableLuaHooks && m.scriptEngine != nil {
		m.scriptEngine.ExecuteFunction(ctx, beforeEncodeFuncName, record.Content)
	}

	// Store the record in LTM
	memoryID, err := m.ltmStore.Store(ctx, record)
	
	// Apply after_encode hook if enabled
	if err == nil && m.config.EnableLuaHooks && m.scriptEngine != nil {
		m.scriptEngine.ExecuteFunction(ctx, afterEncodeFuncName, memoryID)
	}
	
	// Check if we need to manage working memory overflow
	if err == nil {
		m.ManageWorkingMemoryOverflow(ctx)
	}
	
	return memoryID, err
}

// shouldGenerateEmbedding determines if embedding generation is needed for a record.
func (m *MMUI) shouldGenerateEmbedding(record ltm.MemoryRecord) bool {
	// Skip if vector operations are disabled
	if !m.config.EnableVectorOperations {
		return false
	}
	
	// Skip if reasoning engine is not available
	if m.reasoningEngine == nil {
		return false
	}
	
	// Skip if embedding already exists
	if len(record.Embedding) > 0 {
		return false
	}
	
	// Check if the LTM store supports vector operations
	vectorStore, ok := m.ltmStore.(ltm.VectorCapableLTMStore)
	if !ok || !vectorStore.SupportsVectorSearch() {
		return false
	}
	
	// Skip if content is empty
	if record.Content == "" {
		return false
	}
	
	return true
}

// truncateString truncates a string to the specified length and adds "..." if truncated.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ManageWorkingMemoryOverflow handles eviction of items from working memory to LTM
// when the working memory reaches its capacity limit.
// Exported for testing purposes.
func (m *MMUI) ManageWorkingMemoryOverflow(ctx context.Context) {
	// For Phase 2, this is a simple placeholder implementation
	// In future phases, this would implement more sophisticated overflow management
	
	// Skip handling if not initialized or below limit
	if m.workingMemory == nil || len(m.workingMemory) < m.config.WorkingMemoryLimit {
		return
	}
	
	log.Debug("Managing working memory overflow",
		"current_size", len(m.workingMemory),
		"limit", m.config.WorkingMemoryLimit)
	
	// Simple LRU implementation - evict the oldest half of the records
	evictionCount := len(m.workingMemory) / 2
	if evictionCount < 1 && len(m.workingMemory) > 0 {
		evictionCount = 1
	}
	
	// Ensure we don't have an out-of-bounds error
	if evictionCount > 0 && evictionCount <= len(m.workingMemory) {
		// Records to keep
		m.workingMemory = m.workingMemory[evictionCount:]
	}
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
		// Extract embedding if provided directly
		if embedding, ok := q["embedding"].([]float32); ok {
			query.Embedding = embedding
		}
	default:
		// For any other type, return an error
		return nil, fmt.Errorf("unsupported query type: %T", queryInput)
	}

	// Check if we need to generate embeddings for semantic search
	if m.shouldUseSemanticSearch(options.Strategy) && query.Embedding == nil && query.Text != "" {
		// Generate an embedding for the query text
		if err := m.generateQueryEmbedding(ctx, &query); err != nil {
			// Log error but continue with non-semantic search
			log.WarnContext(ctx, "Failed to generate query embedding", "error", err)
		}
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
	log.Debug("Retrieving from LTM", 
		"strategy", options.Strategy,
		"has_embedding", len(query.Embedding) > 0,
		"text", query.Text,
		"limit", query.Limit)
		
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
	
	// If semantic search requested, sort by semantic relevance using rank_semantic_results Lua hook
	if len(query.Embedding) > 0 && len(results) > 0 && 
		m.config.EnableLuaHooks && m.scriptEngine != nil {
		results, err = m.rankSemanticResults(ctx, results, query)
		if err != nil {
			// Log the error but continue
			log.WarnContext(ctx, "Error in rank_semantic_results hook", "error", err)
		}
	}

	// Remove metadata if not requested
	if !options.IncludeMetadata {
		for i := range results {
			results[i].Metadata = nil
		}
	}

	log.Debug("Retrieved results from LTM", 
		"count", len(results),
		"strategy", options.Strategy)

	return results, nil
}

// shouldUseSemanticSearch determines if semantic search should be used.
func (m *MMUI) shouldUseSemanticSearch(strategy string) bool {
	// Skip if vector operations are disabled
	if !m.config.EnableVectorOperations {
		return false
	}
	
	// Skip if reasoning engine is not available
	if m.reasoningEngine == nil {
		return false
	}
	
	// Check if the LTM store supports vector operations
	vectorStore, ok := m.ltmStore.(ltm.VectorCapableLTMStore)
	if !ok || !vectorStore.SupportsVectorSearch() {
		return false
	}
	
	// Check if strategy explicitly requests semantic search
	return strategy == "semantic"
}

// generateQueryEmbedding generates an embedding for a text query.
func (m *MMUI) generateQueryEmbedding(ctx context.Context, query *ltm.LTMQuery) error {
	if query.Text == "" || m.reasoningEngine == nil {
		return nil
	}
	
	embeddings, err := m.reasoningEngine.GenerateEmbeddings(ctx, []string{query.Text})
	if err != nil {
		return err
	}
	
	if len(embeddings) > 0 {
		query.Embedding = embeddings[0]
		log.Debug("Generated query embedding", 
			"dimensions", len(query.Embedding),
			"text", truncateString(query.Text, 30))
	}
	
	return nil
}

// rankSemanticResults uses Lua to re-rank semantic search results if available.
func (m *MMUI) rankSemanticResults(ctx context.Context, results []ltm.MemoryRecord, query ltm.LTMQuery) ([]ltm.MemoryRecord, error) {
	if m.scriptEngine == nil {
		return results, nil
	}
	
	// Call Lua hook with results and query
	ranked, err := m.scriptEngine.ExecuteFunction(ctx, rankSemanticResultsFuncName, results, query.Text)
	if err != nil {
		// Log the error but continue
		log.DebugContext(ctx, "Error calling rank_semantic_results hook", "error", err)
		return results, nil
	}
	
	// Try to convert the result back to []ltm.MemoryRecord
	if rankedRecords, ok := ranked.([]ltm.MemoryRecord); ok {
		return rankedRecords, nil
	}
	
	// If we couldn't convert, just return the original results
	return results, nil
}

// ConsolidateLTM implements the MMU interface.
func (m *MMUI) ConsolidateLTM(ctx context.Context, insight interface{}) error {
	// This is a placeholder for future implementation
	// In a more advanced version, this would implement memory consolidation
	// strategies like summarization, connection discovery, etc.
	return nil
}