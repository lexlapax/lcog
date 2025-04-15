package cogmem

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/lexlapax/cogmem/pkg/reflection"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

// InputType represents the type of input received by the client.
type InputType string

const (
	// InputTypeStore indicates a request to store information.
	InputTypeStore InputType = "store"
	
	// InputTypeRetrieve indicates a request to retrieve information.
	InputTypeRetrieve InputType = "retrieve"
	
	// InputTypeQuery indicates a request to query/process information.
	InputTypeQuery InputType = "query"
)

// CogMemClient is the main facade for the CogMem library.
type CogMemClient interface {
	// Process handles input and produces a response based on the client's capabilities.
	Process(ctx context.Context, inputType InputType, input string) (string, error)
}

// Config contains configuration options for the client.
type Config struct {
	// EnableReflection determines whether reflection is active
	EnableReflection bool
	
	// ReflectionFrequency is how often reflection occurs (in ops count)
	ReflectionFrequency int
}

// DefaultConfig returns the default configuration for the client.
func DefaultConfig() Config {
	return Config{
		EnableReflection:    true,
		ReflectionFrequency: 10,
	}
}

// CogMemClientImpl is the implementation of the CogMemClient interface.
type CogMemClientImpl struct {
	// memoryManager is the MMU for memory operations
	memoryManager mmu.MMU
	
	// reasoningEngine is the engine for generating responses
	reasoningEngine reasoning.Engine
	
	// scriptingEngine is the Lua scripting engine
	scriptingEngine scripting.Engine
	
	// reflectionModule is the module for self-reflection
	reflectionModule reflection.ReflectionModule
	
	// config contains client configuration options
	config Config
	
	// opCount tracks operations for triggering reflection
	opCount int
	
	// operationHistory stores recent operations for reflection
	operationHistory []OperationRecord
}

// OperationRecord represents a single operation performed by the client
type OperationRecord struct {
	InputType InputType `json:"input_type"`
	Input     string    `json:"input"`
	Response  string    `json:"response"`
}

// NewCogMemClient creates a new CogMemClient with the specified dependencies.
func NewCogMemClient(
	memoryManager mmu.MMU,
	reasoningEngine reasoning.Engine,
	scriptingEngine scripting.Engine,
	reflectionModule reflection.ReflectionModule,
	config Config,
) *CogMemClientImpl {
	client := &CogMemClientImpl{
		memoryManager:    memoryManager,
		reasoningEngine:  reasoningEngine,
		scriptingEngine:  scriptingEngine,
		reflectionModule: reflectionModule,
		config:           config,
		opCount:          0,
		operationHistory: make([]OperationRecord, 0, 10), // Keep last 10 operations for reflection
	}
	
	log.Debug("CogMemClient initialized", 
		"reflection_enabled", config.EnableReflection,
		"reflection_frequency", config.ReflectionFrequency,
	)
	
	return client
}

// Process implements the CogMemClient interface.
func (c *CogMemClientImpl) Process(ctx context.Context, inputType InputType, input string) (string, error) {
	// Extract entity context - required for all operations
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return "", entity.ErrMissingEntityContext
	}
	
	log.DebugContext(ctx, "Processing input", 
		"entity_id", entityCtx.EntityID,
		"input_type", inputType,
		"input_length", len(input),
	)
	
	// Increment operation count
	c.opCount++
	
	// Process based on input type
	var response string
	var err error
	
	switch inputType {
	case InputTypeStore:
		response, err = c.handleStore(ctx, input)
	case InputTypeRetrieve:
		response, err = c.handleRetrieve(ctx, input)
	case InputTypeQuery:
		response, err = c.handleQuery(ctx, input)
	default:
		err = fmt.Errorf("unsupported input type: %s", inputType)
		return "", err
	}
	
	// If operation was successful, record it for reflection
	if err == nil {
		c.recordOperation(inputType, input, response)
		
		// Check if reflection should be triggered
		if c.shouldReflect() {
			log.DebugContext(ctx, "Triggering reflection after operation", 
				"operation_count", c.opCount,
				"reflection_frequency", c.config.ReflectionFrequency,
			)
			c.reflect(ctx)
		}
	}
	
	return response, err
}

// handleStore processes a store operation
func (c *CogMemClientImpl) handleStore(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling store operation", "content_length", len(input))
	
	// Store the information in LTM
	memoryID, err := c.memoryManager.EncodeToLTM(ctx, input)
	if err != nil {
		log.ErrorContext(ctx, "Failed to store memory", "error", err)
		return "", err
	}
	
	log.DebugContext(ctx, "Memory stored successfully", "memory_id", memoryID)
	return fmt.Sprintf("Memory stored successfully with ID: %s", memoryID), nil
}

// handleRetrieve processes a retrieve operation
func (c *CogMemClientImpl) handleRetrieve(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling retrieve operation", "query", input)
	
	// Use default retrieval options
	options := mmu.DefaultRetrievalOptions()
	
	// Retrieve relevant memories from LTM
	memories, err := c.memoryManager.RetrieveFromLTM(ctx, input, options)
	if err != nil {
		log.ErrorContext(ctx, "Failed to retrieve memories", "error", err)
		return "", err
	}
	
	// If no memories found, return a simple message
	if len(memories) == 0 {
		log.DebugContext(ctx, "No memories found for query")
		return "No memories found for the query.", nil
	}
	
	log.DebugContext(ctx, "Retrieved memories", "count", len(memories))
	
	// Format memories to show the user
	var memoriesText strings.Builder
	memoriesText.WriteString(fmt.Sprintf("Found %d memories matching your query:\n\n", len(memories)))
	
	for i, memory := range memories {
		memoriesText.WriteString(fmt.Sprintf("Memory %d: %s\n", i+1, memory.Content))
		
		// Add metadata if available
		if memory.Metadata != nil && len(memory.Metadata) > 0 {
			createdAt, ok := memory.Metadata["encoded_at"].(string)
			if ok {
				memoriesText.WriteString(fmt.Sprintf("  Created: %s\n", createdAt))
			}
		}
		memoriesText.WriteString("\n")
	}
	
	// For lookups, return the actual memories instead of a summary
	result := memoriesText.String()
	log.DebugContext(ctx, "Returning memory list", "memory_count", len(memories))
	return result, nil
}

// handleQuery processes a query operation
func (c *CogMemClientImpl) handleQuery(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling query operation", "query", input)
	
	// Check if this is a special semantic search request
	isSemanticSearchRequest := false
	semanticPrefix := "SEMANTIC_SEARCH: "
	if strings.HasPrefix(input, semanticPrefix) {
		isSemanticSearchRequest = true
		// Remove the prefix for processing
		input = strings.TrimPrefix(input, semanticPrefix)
		log.DebugContext(ctx, "Detected semantic search request", "query", input)
	}
	
	// Configure retrieval for semantic search
	options := mmu.RetrievalOptions{
		MaxResults:     5,   // Limit to most relevant memories
		Strategy:       "semantic",
		IncludeMetadata: true,
	}
	
	// Create a semantic query for related memories
	query := map[string]interface{}{
		"text": input, 
	}
	
	// Retrieve relevant context from LTM
	memories, err := c.memoryManager.RetrieveFromLTM(ctx, query, options)
	if err != nil {
		log.ErrorContext(ctx, "Failed to retrieve context for query", "error", err)
		return "", err
	}
	
	// If this is a semantic search request, return the results directly
	if isSemanticSearchRequest {
		if len(memories) == 0 {
			return "No memories found matching your semantic search.", nil
		}
		
		var resultBuilder strings.Builder
		resultBuilder.WriteString(fmt.Sprintf("Found %d memories semantically related to your search:\n\n", len(memories)))
		
		for i, memory := range memories {
			resultBuilder.WriteString(fmt.Sprintf("Memory %d: %s\n", i+1, memory.Content))
			
			// Include similarity score if available
			if memory.Metadata != nil {
				if score, ok := memory.Metadata["score"].(float64); ok {
					resultBuilder.WriteString(fmt.Sprintf("  Similarity: %.2f%%\n", score*100))
				}
			}
			
			// Add creation time if available
			if !memory.CreatedAt.IsZero() {
				resultBuilder.WriteString(fmt.Sprintf("  Created: %s\n", memory.CreatedAt.Format(time.RFC3339)))
			}
			resultBuilder.WriteString("\n")
		}
		
		return resultBuilder.String(), nil
	}
	
	// For regular queries, build prompt with context if available
	var prompt string
	if len(memories) > 0 {
		log.DebugContext(ctx, "Found relevant context memories", "count", len(memories))
		
		// Format memories for context
		var contextBuilder strings.Builder
		contextBuilder.WriteString("Context from memory:\n")
		
		for i, memory := range memories {
			contextBuilder.WriteString(fmt.Sprintf("Memory %d: %s\n", i+1, memory.Content))
		}
		
		prompt = fmt.Sprintf(
			"Using the following context, please answer this question:\n\n%s\n\nQuestion: %s",
			contextBuilder.String(),
			input,
		)
	} else {
		log.DebugContext(ctx, "No relevant context found for query")
		prompt = fmt.Sprintf("Please answer this question: %s", input)
	}
	
	// Process the query with the reasoning engine
	response, err := c.reasoningEngine.Process(ctx, prompt)
	if err != nil {
		log.ErrorContext(ctx, "Failed to process query", "error", err)
		return "", err
	}
	
	log.DebugContext(ctx, "Query processed successfully", "response_length", len(response))
	return response, nil
}

// recordOperation adds an operation to the history for reflection
func (c *CogMemClientImpl) recordOperation(inputType InputType, input, response string) {
	record := OperationRecord{
		InputType: inputType,
		Input:     input,
		Response:  response,
	}
	
	// Keep last 10 operations maximum
	c.operationHistory = append(c.operationHistory, record)
	if len(c.operationHistory) > 10 {
		c.operationHistory = c.operationHistory[1:]
	}
}

// shouldReflect determines if reflection should be triggered
func (c *CogMemClientImpl) shouldReflect() bool {
	// Skip if reflection is disabled
	if !c.config.EnableReflection {
		return false
	}
	
	// Check if enough operations have been performed
	return c.opCount > 0 && c.opCount%c.config.ReflectionFrequency == 0
}

// reflect performs reflection on recent operations
func (c *CogMemClientImpl) reflect(ctx context.Context) {
	// Skip if there's no reflection module or no operations to reflect on
	if c.reflectionModule == nil || len(c.operationHistory) == 0 {
		return
	}
	
	log.DebugContext(ctx, "Performing reflection", "history_length", len(c.operationHistory))
	
	// Also store the recent operation history in LTM before reflection
	historyJSON, err := json.Marshal(c.operationHistory)
	if err != nil {
		log.ErrorContext(ctx, "Failed to marshal operation history for reflection", "error", err)
		return
	}
	
	// Store the history with metadata
	historyData := map[string]interface{}{
		"content": string(historyJSON),
		"metadata": map[string]interface{}{
			"type":           "operation_history",
			"operation_count": c.opCount,
			"timestamp":      time.Now().Format(time.RFC3339),
		},
	}
	
	// Store the history in LTM (ignore errors, this is just for context)
	_, _ = c.memoryManager.EncodeToLTM(ctx, historyData)
	
	// Trigger the reflection process
	insights, err := c.reflectionModule.TriggerReflection(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Error during reflection process", "error", err)
		return
	}
	
	log.DebugContext(ctx, "Reflection completed", 
		"insight_count", len(insights),
		"operation_count", c.opCount)
}