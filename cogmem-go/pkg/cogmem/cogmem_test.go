package cogmem

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/config"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/lexlapax/cogmem/pkg/reflection"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

// MockReflectionModule is a mock implementation of the reflection.ReflectionModule interface
type MockReflectionModule struct {
	mock.Mock
}

func (m *MockReflectionModule) TriggerReflection(ctx context.Context) ([]*reflection.Insight, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*reflection.Insight), args.Error(1)
}

// Helper function to set up a test environment
func setupClientTest(t *testing.T) (*CogMemClientImpl, *MockMMU, *MockReasoningEngine, *MockScriptingEngine, *MockReflectionModule, context.Context) {
	mockMMU := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptingEngine)
	mockReflection := new(MockReflectionModule)

	client := NewCogMem(
		mockMMU,
		mockReasoning,
		mockScripting,
		mockReflection,
		DefaultConfig(),
	)

	// Create a context with entity information
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)

	return client, mockMMU, mockReasoning, mockScripting, mockReflection, ctx
}

func TestCogMemClient_Process_MissingEntityContext(t *testing.T) {
	// Setup
	client, _, _, _, _, _ := setupClientTest(t)

	// Test with a context that doesn't have entity information
	result, err := client.Process(context.Background(), InputTypeQuery, "test query")
	
	// Verify error about missing entity context
	assert.Error(t, err)
	assert.Equal(t, entity.ErrMissingEntityContext, err)
	assert.Empty(t, result)
}

func TestCogMemClient_Process_Store(t *testing.T) {
	// Setup
	client, mockMMU, _, _, _, ctx := setupClientTest(t)

	// Set up expectations
	mockMMU.On("EncodeToLTM", ctx, "test memory").Return("memory-123", nil)

	// Test store operation
	result, err := client.Process(ctx, InputTypeStore, "test memory")
	require.NoError(t, err)
	assert.Equal(t, "Memory stored successfully with ID: memory-123", result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestCogMemClient_Process_Store_Error(t *testing.T) {
	// Setup
	client, mockMMU, _, _, _, ctx := setupClientTest(t)

	// Set up expectations with an error
	mockError := errors.New("storage error")
	mockMMU.On("EncodeToLTM", ctx, "test memory").Return("", mockError)

	// Test store operation with error
	result, err := client.Process(ctx, InputTypeStore, "test memory")
	assert.Error(t, err)
	assert.Equal(t, mockError, err)
	assert.Empty(t, result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestCogMemClient_Process_Retrieve(t *testing.T) {
	// Setup
	client, mockMMU, _, _, _, ctx := setupClientTest(t)

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
	result, err := client.Process(ctx, InputTypeRetrieve, "test query")
	require.NoError(t, err)
	
	// Check that the result contains the expected content
	assert.Contains(t, result, "Found 2 memories")
	assert.Contains(t, result, "First memory")
	assert.Contains(t, result, "Second memory")

	// Verify the MMU mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestCogMemClient_Process_Retrieve_NoResults(t *testing.T) {
	// Setup
	client, mockMMU, _, _, _, ctx := setupClientTest(t)

	// Set up MMU expectation with empty results
	mockMMU.On("RetrieveFromLTM", ctx, "test query", mock.Anything).Return([]ltm.MemoryRecord{}, nil)

	// Test retrieve operation with no results
	result, err := client.Process(ctx, InputTypeRetrieve, "test query")
	require.NoError(t, err)
	assert.Equal(t, "No memories found for the query.", result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestCogMemClient_Process_Retrieve_Error(t *testing.T) {
	// Setup
	client, mockMMU, _, _, _, ctx := setupClientTest(t)

	// Set up MMU expectation with an error
	mockError := errors.New("retrieval error")
	mockMMU.On("RetrieveFromLTM", ctx, "test query", mock.Anything).Return([]ltm.MemoryRecord{}, mockError)

	// Test retrieve operation with error
	result, err := client.Process(ctx, InputTypeRetrieve, "test query")
	assert.Error(t, err)
	assert.Equal(t, mockError, err)
	assert.Empty(t, result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestCogMemClient_Process_Query(t *testing.T) {
	// Setup
	client, mockMMU, mockReasoning, _, _, ctx := setupClientTest(t)

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
	result, err := client.Process(ctx, InputTypeQuery, "test question")
	require.NoError(t, err)
	assert.Equal(t, "Query response with context", result)

	// Verify the mocks were called as expected
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
}

func TestCogMemClient_Process_Query_NoRelevantMemories(t *testing.T) {
	// Setup
	client, mockMMU, mockReasoning, _, _, ctx := setupClientTest(t)

	// Set up MMU expectation with empty results
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, nil)

	// Set up reasoning expectation for processing without context
	mockReasoning.On("Process", ctx, mock.Anything).Return("Query response without context", nil)

	// Test query operation with no context memories
	result, err := client.Process(ctx, InputTypeQuery, "test question")
	require.NoError(t, err)
	assert.Equal(t, "Query response without context", result)

	// Verify the mocks were called as expected
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
}

func TestCogMemClient_Process_Query_MMUError(t *testing.T) {
	// Setup
	client, mockMMU, _, _, _, ctx := setupClientTest(t)

	// Set up MMU expectation with an error
	mockError := errors.New("memory retrieval error")
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, mockError)

	// Test query operation with MMU error
	result, err := client.Process(ctx, InputTypeQuery, "test question")
	assert.Error(t, err)
	assert.Equal(t, mockError, err)
	assert.Empty(t, result)

	// Verify the mock was called as expected
	mockMMU.AssertExpectations(t)
}

func TestCogMemClient_Process_Query_ReasoningError(t *testing.T) {
	// Setup
	client, mockMMU, mockReasoning, _, _, ctx := setupClientTest(t)

	// Set up MMU expectation
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, nil)

	// Set up reasoning expectation with an error
	mockError := errors.New("reasoning error")
	mockReasoning.On("Process", ctx, mock.Anything).Return("", mockError)

	// Test query operation with reasoning error
	result, err := client.Process(ctx, InputTypeQuery, "test question")
	assert.Error(t, err)
	assert.Equal(t, mockError, err)
	assert.Empty(t, result)

	// Verify the mocks were called as expected
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
}

func TestCogMemClient_Process_InvalidInputType(t *testing.T) {
	// Setup
	client, _, _, _, _, ctx := setupClientTest(t)

	// Test with invalid input type
	result, err := client.Process(ctx, "invalid", "test input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported input type")
	assert.Empty(t, result)
}

func TestCogMemClient_Reflection(t *testing.T) {
	// Setup
	mockMMU := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptingEngine)
	mockReflection := new(MockReflectionModule)

	// Configure client with reflection enabled and frequency of 2
	client := NewCogMem(
		mockMMU,
		mockReasoning,
		mockScripting,
		mockReflection,
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
	_, err := client.Process(ctx, InputTypeQuery, "query1")
	require.NoError(t, err)

	// Set up expectations for second operation (with reflection)
	// We need a second retrieval call as our implementation does semantic search
	mockMMU.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, nil)
	mockReasoning.On("Process", ctx, mock.Anything).Return("response2", nil).Once()
	
	// Expect a call to store the operation history
	mockMMU.On("EncodeToLTM", mock.Anything, mock.Anything).Return("history-id", nil).Once()
	
	// Create some mock insights for the reflection module to return
	mockInsights := []*reflection.Insight{
		{
			ID:          "insight-1",
			Type:        "pattern",
			Description: "Test insight",
			Confidence:  0.9,
		},
	}
	
	// Expect a call to the reflection module
	mockReflection.On("TriggerReflection", mock.Anything).Return(mockInsights, nil).Once()

	// Second operation should trigger reflection
	_, err = client.Process(ctx, InputTypeQuery, "query2")
	require.NoError(t, err)

	// Verify all expectations were met
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
	mockScripting.AssertExpectations(t)
	mockReflection.AssertExpectations(t)
}

func TestCogMemClient_ReflectionDisabled(t *testing.T) {
	// Setup
	mockMMU := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptingEngine)
	mockReflection := new(MockReflectionModule)

	// Configure client with reflection disabled
	client := NewCogMem(
		mockMMU,
		mockReasoning,
		mockScripting,
		mockReflection,
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
	_, err := client.Process(ctx, InputTypeQuery, "query")
	require.NoError(t, err)

	// Verify all expectations were met (no calls to scripting or reflection module)
	mockMMU.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
	mockScripting.AssertNotCalled(t, "ExecuteFunction")
	mockReflection.AssertNotCalled(t, "TriggerReflection")
}

// TestNewCogMemFromConfig is a unit test for the config initialization function
func TestNewCogMemFromConfig(t *testing.T) {
	// Create a temp directory for test config
	tempDir, err := os.MkdirTemp("", "cogmem-config-unit-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a minimal mock config
	mockConfig := config.Config{
		LTM: config.LTMConfig{
			Type: "mock", // Mock store for unit tests
		},
		Scripting: config.ScriptingConfig{
			Paths: []string{tempDir}, // Path doesn't need to exist for mock
		},
		Reasoning: config.ReasoningConfig{
			Provider: "mock",
		},
		Reflection: config.ReflectionConfig{
			Enabled: true,
		},
	}

	// Write config to file
	configYaml, err := yaml.Marshal(mockConfig)
	require.NoError(t, err)
	configPath := filepath.Join(tempDir, "mock_config.yaml")
	err = os.WriteFile(configPath, configYaml, 0644)
	require.NoError(t, err)

	// Test NewCogMemFromConfig with the mock config
	client, err := NewCogMemFromConfig(configPath)
	
	// Should succeed with a mock config
	require.NoError(t, err)
	require.NotNil(t, client)

	// Test with invalid config path
	_, err = NewCogMemFromConfig("/path/does/not/exist.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}