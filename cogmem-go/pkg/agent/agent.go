package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

// InputType represents the type of input received by the agent.
type InputType string

const (
	// InputTypeStore indicates a request to store information.
	InputTypeStore InputType = "store"
	
	// InputTypeRetrieve indicates a request to retrieve information.
	InputTypeRetrieve InputType = "retrieve"
	
	// InputTypeQuery indicates a request to query/process information.
	InputTypeQuery InputType = "query"
)

// Agent is the main facade for the CogMem library.
type Agent interface {
	// Process handles input and produces a response based on the agent's capabilities.
	Process(ctx context.Context, inputType InputType, input string) (string, error)
}

// Config contains configuration options for the agent.
type Config struct {
	// EnableReflection determines whether reflection is active
	EnableReflection bool
	
	// ReflectionFrequency is how often reflection occurs (in ops count)
	ReflectionFrequency int
}

// DefaultConfig returns the default configuration for the agent.
func DefaultConfig() Config {
	return Config{
		EnableReflection:    true,
		ReflectionFrequency: 10,
	}
}

// AgentI is the implementation of the Agent interface.
type AgentI struct {
	// memoryManager is the MMU for memory operations
	memoryManager mmu.MMU
	
	// reasoningEngine is the engine for generating responses
	reasoningEngine reasoning.Engine
	
	// scriptingEngine is the Lua scripting engine
	scriptingEngine scripting.Engine
	
	// config contains agent configuration options
	config Config
	
	// opCount tracks operations for triggering reflection
	opCount int
	
	// operationHistory stores recent operations for reflection
	operationHistory []OperationRecord
}

// OperationRecord represents a single operation performed by the agent
type OperationRecord struct {
	InputType InputType `json:"input_type"`
	Input     string    `json:"input"`
	Response  string    `json:"response"`
}

// NewAgent creates a new Agent with the specified dependencies.
func NewAgent(
	memoryManager mmu.MMU,
	reasoningEngine reasoning.Engine,
	scriptingEngine scripting.Engine,
	config Config,
) *AgentI {
	agent := &AgentI{
		memoryManager:    memoryManager,
		reasoningEngine:  reasoningEngine,
		scriptingEngine:  scriptingEngine,
		config:           config,
		opCount:          0,
		operationHistory: make([]OperationRecord, 0, 10), // Keep last 10 operations for reflection
	}
	
	log.Debug("Agent initialized", 
		"reflection_enabled", config.EnableReflection,
		"reflection_frequency", config.ReflectionFrequency,
	)
	
	return agent
}

// Process implements the Agent interface.
func (a *AgentI) Process(ctx context.Context, inputType InputType, input string) (string, error) {
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
	a.opCount++
	
	// Process based on input type
	var response string
	var err error
	
	switch inputType {
	case InputTypeStore:
		response, err = a.handleStore(ctx, input)
	case InputTypeRetrieve:
		response, err = a.handleRetrieve(ctx, input)
	case InputTypeQuery:
		response, err = a.handleQuery(ctx, input)
	default:
		err = fmt.Errorf("unsupported input type: %s", inputType)
		return "", err
	}
	
	// If operation was successful, record it for reflection
	if err == nil {
		a.recordOperation(inputType, input, response)
		
		// Check if reflection should be triggered
		if a.shouldReflect() {
			log.DebugContext(ctx, "Triggering reflection after operation", 
				"operation_count", a.opCount,
				"reflection_frequency", a.config.ReflectionFrequency,
			)
			a.reflect(ctx)
		}
	}
	
	return response, err
}

// handleStore processes a store operation
func (a *AgentI) handleStore(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling store operation", "content_length", len(input))
	
	// Store the information in LTM
	memoryID, err := a.memoryManager.EncodeToLTM(ctx, input)
	if err != nil {
		log.ErrorContext(ctx, "Failed to store memory", "error", err)
		return "", err
	}
	
	log.DebugContext(ctx, "Memory stored successfully", "memory_id", memoryID)
	return fmt.Sprintf("Memory stored successfully with ID: %s", memoryID), nil
}

// handleRetrieve processes a retrieve operation
func (a *AgentI) handleRetrieve(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling retrieve operation", "query", input)
	
	// Use default retrieval options
	options := mmu.DefaultRetrievalOptions()
	
	// Retrieve relevant memories from LTM
	memories, err := a.memoryManager.RetrieveFromLTM(ctx, input, options)
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
func (a *AgentI) handleQuery(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling query operation", "query", input)
	
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
	memories, err := a.memoryManager.RetrieveFromLTM(ctx, query, options)
	if err != nil {
		log.ErrorContext(ctx, "Failed to retrieve context for query", "error", err)
		return "", err
	}
	
	// Build prompt with context if available
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
	response, err := a.reasoningEngine.Process(ctx, prompt)
	if err != nil {
		log.ErrorContext(ctx, "Failed to process query", "error", err)
		return "", err
	}
	
	log.DebugContext(ctx, "Query processed successfully", "response_length", len(response))
	return response, nil
}

// recordOperation adds an operation to the history for reflection
func (a *AgentI) recordOperation(inputType InputType, input, response string) {
	record := OperationRecord{
		InputType: inputType,
		Input:     input,
		Response:  response,
	}
	
	// Keep last 10 operations maximum
	a.operationHistory = append(a.operationHistory, record)
	if len(a.operationHistory) > 10 {
		a.operationHistory = a.operationHistory[1:]
	}
}

// shouldReflect determines if reflection should be triggered
func (a *AgentI) shouldReflect() bool {
	// Skip if reflection is disabled
	if !a.config.EnableReflection {
		return false
	}
	
	// Check if enough operations have been performed
	return a.opCount > 0 && a.opCount%a.config.ReflectionFrequency == 0
}

// reflect performs reflection on recent operations
func (a *AgentI) reflect(ctx context.Context) {
	// Skip if there's no scripting engine or no operations to reflect on
	if a.scriptingEngine == nil || len(a.operationHistory) == 0 {
		return
	}
	
	log.DebugContext(ctx, "Performing reflection", "history_length", len(a.operationHistory))
	
	// Convert operation history to JSON for Lua
	historyJSON, err := json.Marshal(a.operationHistory)
	if err != nil {
		log.ErrorContext(ctx, "Failed to marshal operation history for reflection", "error", err)
		return
	}
	
	// Call the Lua reflection function
	insights, err := a.scriptingEngine.ExecuteFunction(ctx, "reflect", string(historyJSON))
	if err != nil {
		log.ErrorContext(ctx, "Error executing reflection script", "error", err)
		return
	}
	
	// Always attempt to consolidate insights, even if nil
	// This makes testing easier and is consistent with our implementation
	log.DebugContext(ctx, "Reflection completed", "insights", insights)
	
	if err := a.memoryManager.ConsolidateLTM(ctx, insights); err != nil {
		log.ErrorContext(ctx, "Failed to consolidate reflection insights", "error", err)
	}
}
