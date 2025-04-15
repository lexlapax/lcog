package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockMMU is a mock implementation of the mmu.MMU interface
type MockMMU struct {
	mock.Mock
}

func (m *MockMMU) EncodeToLTM(ctx context.Context, dataToStore interface{}) (string, error) {
	args := m.Called(ctx, dataToStore)
	return args.String(0), args.Error(1)
}

func (m *MockMMU) RetrieveFromLTM(ctx context.Context, query interface{}, options mmu.RetrievalOptions) ([]ltm.MemoryRecord, error) {
	args := m.Called(ctx, query, options)
	return args.Get(0).([]ltm.MemoryRecord), args.Error(1)
}

func (m *MockMMU) ConsolidateLTM(ctx context.Context, insight interface{}) error {
	args := m.Called(ctx, insight)
	return args.Error(0)
}

// MockReasoningEngine is a mock implementation of the reasoning.Engine interface
type MockReasoningEngine struct {
	mock.Mock
}

func (m *MockReasoningEngine) Process(ctx context.Context, prompt string, opts ...reasoning.Option) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockReasoningEngine) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	args := m.Called(ctx, texts)
	return args.Get(0).([][]float32), args.Error(1)
}

// MockScriptingEngine is a mock implementation of the scripting.Engine interface
type MockScriptingEngine struct {
	mock.Mock
}

func (m *MockScriptingEngine) LoadScript(name string, content []byte) error {
	args := m.Called(name, content)
	return args.Error(0)
}

func (m *MockScriptingEngine) LoadScriptFile(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockScriptingEngine) LoadScriptDir(dir string) error {
	args := m.Called(dir)
	return args.Error(0)
}

func (m *MockScriptingEngine) ExecuteFunction(ctx context.Context, funcName string, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(ctx, funcName, args)
	return callArgs.Get(0), callArgs.Error(1)
}

func (m *MockScriptingEngine) Close() error {
	args := m.Called()
	return args.Error(0)
}

// Helper function to set up a test environment
func setupAgentTest(t *testing.T) (*AgentI, *MockMMU, *MockReasoningEngine, *MockScriptingEngine, context.Context) {
	mockMMU := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptingEngine)

	agent := NewAgent(
		mockMMU,
		mockReasoning,
		mockScripting,
		DefaultConfig(),
	)

	// Create a context with entity information
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)

	return agent, mockMMU, mockReasoning, mockScripting, ctx
}

func TestAgent_Process_MissingEntityContext(t *testing.T) {
	// Setup
	agent, _, _, _, _ := setupAgentTest(t)

	// Test with a context that doesn't have entity information
	result, err := agent.Process(context.Background(), InputTypeQuery, "test query")
	
	// Verify error about missing entity context
	assert.Error(t, err)
	assert.Equal(t, entity.ErrMissingEntityContext, err)
	assert.Empty(t, result)
}

func TestAgent_Process_Store(t *testing.T) {
	// Setup
	agent, mockMMU, _, _, ctx := setupAgentTest(t)

	// Set up expectations
	mockMMU.On("EncodeToLTM", ctx, "test memory").Return("memory-123", nil)

	// Test store operation
	result, err := agent.Process(ctx, InputTypeStore, "test memory")
	require.NoError(t, err)
	assert.Equal(t, "Memory stored successfully with ID: memory-123", result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestAgent_Process_Store_Error(t *testing.T) {
	// Setup
	agent, mockMMU, _, _, ctx := setupAgentTest(t)

	// Set up expectations with an error
	mockError := errors.New("storage error")
	mockMMU.On("EncodeToLTM", ctx, "test memory").Return("", mockError)

	// Test store operation with error
	result, err := agent.Process(ctx, InputTypeStore, "test memory")
	assert.Error(t, err)
	assert.Equal(t, mockError, err)
	assert.Empty(t, result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestAgent_Process_Retrieve(t *testing.T) {
	// Setup
	agent, mockMMU, _, _, ctx := setupAgentTest(t)

	// Create some mock memory records
	mockRecords := []ltm.MemoryRecord{
		{
			ID:      "memory-1",
			Content: "First memory",
		},
		{
			ID:      "memory-2",
			Content: "Second memory",
		},
	}

	// Set up MMU expectation
	mockMMU.On("RetrieveFromLTM", ctx, "test query", mock.Anything).Return(mockRecords, nil)

	// No longer use reasoning for retrieval - we display the memories directly

	// Test retrieve operation
	result, err := agent.Process(ctx, InputTypeRetrieve, "test query")
	require.NoError(t, err)
	
	// Check that the result contains the expected content
	assert.Contains(t, result, "Found 2 memories")
	assert.Contains(t, result, "First memory")
	assert.Contains(t, result, "Second memory")

	// Verify the MMU mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestAgent_Process_Retrieve_NoResults(t *testing.T) {
	// Setup
	agent, mockMMU, _, _, ctx := setupAgentTest(t)

	// Set up MMU expectation with empty results
	mockMMU.On("RetrieveFromLTM", ctx, "test query", mock.Anything).Return([]ltm.MemoryRecord{}, nil)

	// Test retrieve operation with no results
	result, err := agent.Process(ctx, InputTypeRetrieve, "test query")
	require.NoError(t, err)
	assert.Equal(t, "No memories found for the query.", result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestAgent_Process_Retrieve_Error(t *testing.T) {
	// Setup
	agent, mockMMU, _, _, ctx := setupAgentTest(t)

	// Set up MMU expectation with an error
	mockError := errors.New("retrieval error")
	mockMMU.On("RetrieveFromLTM", ctx, "test query", mock.Anything).Return([]ltm.MemoryRecord{}, mockError)

	// Test retrieve operation with error
	result, err := agent.Process(ctx, InputTypeRetrieve, "test query")
	assert.Error(t, err)
	assert.Equal(t, mockError, err)
	assert.Empty(t, result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestAgent_Process_Query(t *testing.T) {
	// Setup
	agent, mockMMU, mockReasoning, _, ctx := setupAgentTest(t)

	// Create some mock memory records
	mockRecords := []ltm.MemoryRecord{
		{
			ID:      "memory-1",
			Content: "Context memory 1",
		},
		{
			ID:      "memory-2",
			Content: "Context memory 2",
		},
	}

	// Set up MMU expectation for semantic search
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return(mockRecords, nil)

	// Set up reasoning expectation for processing the query with context
	mockReasoning.On("Process", ctx, mock.Anything).Return("Query response with context", nil)

	// Test query operation
	result, err := agent.Process(ctx, InputTypeQuery, "test question")
	require.NoError(t, err)
	assert.Equal(t, "Query response with context", result)

	// Verify the mocks were called as expected
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
}

func TestAgent_Process_Query_NoRelevantMemories(t *testing.T) {
	// Setup
	agent, mockMMU, mockReasoning, _, ctx := setupAgentTest(t)

	// Set up MMU expectation with empty results
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, nil)

	// Set up reasoning expectation for processing without context
	mockReasoning.On("Process", ctx, mock.Anything).Return("Query response without context", nil)

	// Test query operation with no context memories
	result, err := agent.Process(ctx, InputTypeQuery, "test question")
	require.NoError(t, err)
	assert.Equal(t, "Query response without context", result)

	// Verify the mocks were called as expected
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
}

func TestAgent_Process_Query_MMUError(t *testing.T) {
	// Setup
	agent, mockMMU, _, _, ctx := setupAgentTest(t)

	// Set up MMU expectation with an error
	mockError := errors.New("memory retrieval error")
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, mockError)

	// Test query operation with MMU error
	result, err := agent.Process(ctx, InputTypeQuery, "test question")
	assert.Error(t, err)
	assert.Equal(t, mockError, err)
	assert.Empty(t, result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestAgent_Process_Query_ReasoningError(t *testing.T) {
	// Setup
	agent, mockMMU, mockReasoning, _, ctx := setupAgentTest(t)

	// Set up MMU expectation
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, nil)

	// Set up reasoning expectation with an error
	mockError := errors.New("reasoning error")
	mockReasoning.On("Process", ctx, mock.Anything).Return("", mockError)

	// Test query operation with reasoning error
	result, err := agent.Process(ctx, InputTypeQuery, "test question")
	assert.Error(t, err)
	assert.Equal(t, mockError, err)
	assert.Empty(t, result)

	// Verify the mocks were called as expected
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
}

func TestAgent_Process_InvalidInputType(t *testing.T) {
	// Setup
	agent, _, _, _, ctx := setupAgentTest(t)

	// Test with invalid input type
	result, err := agent.Process(ctx, "invalid", "test input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported input type")
	assert.Empty(t, result)
}

func TestAgent_Reflection(t *testing.T) {
	// Setup
	mockMMU := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptingEngine)

	// Configure agent with reflection enabled and frequency of 2
	agent := NewAgent(
		mockMMU,
		mockReasoning,
		mockScripting,
		Config{
			EnableReflection:    true,
			ReflectionFrequency: 2, // Trigger reflection every 2 operations
		},
	)

	// Create a context with entity information
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)

	// Set up mock expectations for first operation (no reflection)
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, nil)
	mockReasoning.On("Process", ctx, mock.Anything).Return("response1", nil).Once()

	// First operation
	_, err := agent.Process(ctx, InputTypeQuery, "query1")
	require.NoError(t, err)

	// Set up expectations for second operation (with reflection)
	// We need a second retrieval call as our implementation does semantic search
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, nil)
	mockReasoning.On("Process", ctx, mock.Anything).Return("response2", nil).Once()
	
	// Expect a call to the scripting engine for reflection
	mockScripting.On("ExecuteFunction", ctx, "reflect", mock.Anything).Return(nil, nil).Once()
	
	// Expect a call to consolidate the reflection insights
	mockMMU.On("ConsolidateLTM", ctx, mock.Anything).Return(nil).Once()

	// Second operation should trigger reflection
	_, err = agent.Process(ctx, InputTypeQuery, "query2")
	require.NoError(t, err)

	// Verify all expectations were met
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
	mockScripting.AssertExpectations(t)
}

func TestAgent_ReflectionDisabled(t *testing.T) {
	// Setup
	mockMMU := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptingEngine)

	// Configure agent with reflection disabled
	agent := NewAgent(
		mockMMU,
		mockReasoning,
		mockScripting,
		Config{
			EnableReflection:    false,
			ReflectionFrequency: 1, // Would trigger every operation if enabled
		},
	)

	// Create a context with entity information
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)

	// Set up mock expectations for operation (no reflection)
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, nil)
	mockReasoning.On("Process", ctx, mock.Anything).Return("response", nil)

	// Operation with reflection disabled
	_, err := agent.Process(ctx, InputTypeQuery, "query")
	require.NoError(t, err)

	// Verify all expectations were met (no calls to scripting for reflection)
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
	mockScripting.AssertNotCalled(t, "ExecuteFunction")
	mockMMU.AssertNotCalled(t, "ConsolidateLTM")
}
