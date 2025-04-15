// Package agent provides backward compatibility for the renamed cogmem package.
// This package is deprecated and will be removed in a future version.
package agent

import (
	"context"

	"github.com/lexlapax/cogmem/pkg/cogmem"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

// InputType is a compatibility alias for cogmem.InputType
type InputType = cogmem.InputType

// Constants for backward compatibility
const (
	InputTypeStore    = cogmem.InputTypeStore
	InputTypeRetrieve = cogmem.InputTypeRetrieve
	InputTypeQuery    = cogmem.InputTypeQuery
)

// Agent is a compatibility interface that mirrors the CogMemClient interface
type Agent interface {
	// Process handles input and produces a response based on the agent's capabilities.
	Process(ctx context.Context, inputType InputType, input string) (string, error)
}

// AgentI is a compatibility struct for backward compatibility
type AgentI struct {
	client *cogmem.CogMemClientImpl
}

// Config is a compatibility alias for cogmem.Config
type Config = cogmem.Config

// DefaultConfig returns the default configuration for the agent.
func DefaultConfig() Config {
	return cogmem.DefaultConfig()
}

// OperationRecord is a compatibility alias for cogmem.OperationRecord
type OperationRecord = cogmem.OperationRecord

// NewAgent creates a new Agent instance that wraps a CogMemClient.
func NewAgent(
	memoryManager mmu.MMU,
	reasoningEngine reasoning.Engine,
	scriptingEngine scripting.Engine,
	config Config,
) *AgentI {
	client := cogmem.NewCogMemClient(memoryManager, reasoningEngine, scriptingEngine, config)
	return &AgentI{client: client}
}

// Process implements the Agent interface by delegating to the underlying CogMemClient.
func (a *AgentI) Process(ctx context.Context, inputType InputType, input string) (string, error) {
	return a.client.Process(ctx, inputType, input)
}