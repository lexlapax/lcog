package agent

import (
	"context"

	"github.com/spurintel/cogmem-go/pkg/entity"
	"github.com/spurintel/cogmem-go/pkg/mmu"
	"github.com/spurintel/cogmem-go/pkg/reasoning"
	"github.com/spurintel/cogmem-go/pkg/scripting"
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
}

// NewAgent creates a new Agent with the specified dependencies.
func NewAgent(
	memoryManager mmu.MMU,
	reasoningEngine reasoning.Engine,
	scriptingEngine scripting.Engine,
	config Config,
) *AgentI {
	return &AgentI{
		memoryManager:   memoryManager,
		reasoningEngine: reasoningEngine,
		scriptingEngine: scriptingEngine,
		config:          config,
		opCount:         0,
	}
}

// Process implements the Agent interface.
func (a *AgentI) Process(ctx context.Context, inputType InputType, input string) (string, error) {
	// Extract entity context - required for all operations
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return "", entity.ErrMissingEntityContext
	}
	
	// Increment operation count
	a.opCount++
	
	// This is just a placeholder - implementation will be added in later steps
	return "", nil
}
