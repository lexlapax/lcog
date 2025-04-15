package agent

import (
	"context"
	"testing"

	"github.com/lexlapax/cogmem/pkg/cogmem"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mmu"
	reasoningmock "github.com/lexlapax/cogmem/pkg/reasoning/adapters/mock"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/mock"
	"github.com/lexlapax/cogmem/pkg/scripting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test that the compatibility layer correctly delegates to the CogMemClient implementation
func TestAgent_Compatibility(t *testing.T) {
	// Create a mock LTM store
	ltmStore := mock.NewMockStore()
	
	// Create a mock reasoning engine
	mockReasoning := reasoningmock.NewMockEngine()
	mockReasoning.AddResponse("Please answer this question: test", "Test response")
	
	// Create a scripting engine
	scriptEngine, err := scripting.NewLuaEngine(scripting.DefaultConfig())
	require.NoError(t, err)
	defer scriptEngine.Close()
	
	// Create an MMU
	mmuInstance := mmu.NewMMU(ltmStore, mockReasoning, scriptEngine, mmu.DefaultConfig())
	
	// Create a context with entity information
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Create both a direct client and a compatibility client
	clientDirect := cogmem.NewCogMemClient(mmuInstance, mockReasoning, scriptEngine, cogmem.DefaultConfig())
	clientCompat := NewAgent(mmuInstance, mockReasoning, scriptEngine, DefaultConfig())
	
	// Test that both clients produce the same result
	responseDirect, err := clientDirect.Process(ctx, cogmem.InputTypeQuery, "test")
	require.NoError(t, err)
	
	responseCompat, err := clientCompat.Process(ctx, InputTypeQuery, "test")
	require.NoError(t, err)
	
	assert.Equal(t, responseDirect, responseCompat, "Direct and compatibility clients should return the same response")
}