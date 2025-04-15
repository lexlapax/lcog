package reflection

import (
	"context"
	"testing"
	"time"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockMMU mocks the MMU interface for testing
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

// MockReasoningEngine mocks the reasoning engine interface for testing
type MockReasoningEngine struct {
	mock.Mock
}

func (m *MockReasoningEngine) Process(ctx context.Context, prompt string, opts ...reasoning.Option) (string, error) {
	args := m.Called(ctx, prompt, opts)
	return args.String(0), args.Error(1)
}

func (m *MockReasoningEngine) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	args := m.Called(ctx, texts)
	return args.Get(0).([][]float32), args.Error(1)
}

// MockScriptEngine mocks the scripting engine interface for testing
type MockScriptEngine struct {
	mock.Mock
}

func (m *MockScriptEngine) LoadScript(name string, content []byte) error {
	args := m.Called(name, content)
	return args.Error(0)
}

func (m *MockScriptEngine) LoadScriptFile(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockScriptEngine) LoadScriptDir(dir string) error {
	args := m.Called(dir)
	return args.Error(0)
}

func (m *MockScriptEngine) ExecuteFunction(ctx context.Context, functionName string, args ...interface{}) (interface{}, error) {
	callArgs := m.Called(append([]interface{}{ctx, functionName}, args...)...)
	return callArgs.Get(0), callArgs.Error(1)
}

func (m *MockScriptEngine) Close() error {
	args := m.Called()
	return args.Error(0)
}

func init() {
	// Set up test logger
	log.Setup(log.Config{
		Level:  log.DebugLevel,
		Format: log.TextFormat,
	})
}

func TestNewReflectionModule(t *testing.T) {
	mockMmu := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptEngine)
	mockScripting.On("Close").Return(nil).Maybe()
	
	config := DefaultConfig()
	
	module := NewReflectionModule(mockMmu, mockReasoning, mockScripting, config)
	
	assert.NotNil(t, module)
	assert.Equal(t, mockMmu, module.mmu)
	assert.Equal(t, mockReasoning, module.reasoningEngine)
	assert.Equal(t, mockScripting, module.scriptEngine)
	assert.Equal(t, config, module.config)
}

func TestTriggerReflection(t *testing.T) {
	// Create mocks
	mockMmu := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptEngine)
	mockScripting.On("Close").Return(nil).Maybe()
	
	// Create context with entity
	ctx := context.Background()
	ctx = entity.ContextWithEntity(ctx, entity.Context{
		EntityID: "test-entity",
		UserID:   "test-user",
	})
	
	// Create test memory records
	records := []ltm.MemoryRecord{
		{
			ID:         "record1",
			EntityID:   "test-entity",
			Content:    "Memory content 1",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			AccessLevel: entity.SharedWithinEntity,
		},
		{
			ID:         "record2",
			EntityID:   "test-entity",
			Content:    "Memory content 2",
			CreatedAt:  time.Now().Add(-30 * time.Minute),
			AccessLevel: entity.SharedWithinEntity,
		},
	}
	
	// Mock the before hook call (if enabled) - return false to NOT skip analysis
	mockScripting.On("ExecuteFunction", ctx, "before_reflection_analysis", mock.Anything).Return(false, nil)
	
	// Mock the retrieval call
	mockMmu.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return(records, nil)
	
	// Sample LLM response with insights
	llmResponse := `
	{
		"insights": [
			{
				"type": "pattern",
				"description": "Detected a recurring theme about X",
				"confidence": 0.85,
				"related_memory_ids": ["record1", "record2"]
			},
			{
				"type": "connection",
				"description": "Found connection between concepts Y and Z",
				"confidence": 0.72,
				"related_memory_ids": ["record2"]
			}
		]
	}`
	
	// Mock the reasoning engine call
	mockReasoning.On("Process", ctx, mock.Anything, mock.Anything).Return(llmResponse, nil)
	
	// Mock the after hook call (if enabled)
	mockScripting.On("ExecuteFunction", ctx, "after_insight_generation", mock.Anything).Return(nil, nil)
	
	// Mock the before_consolidation hook
	mockScripting.On("ExecuteFunction", ctx, "before_consolidation", mock.Anything).Return(mock.Anything, nil)
	
	// Mock the consolidation call
	mockMmu.On("ConsolidateLTM", ctx, mock.Anything).Return(nil)
	
	// Create the module
	config := DefaultConfig()
	config.EnableLuaHooks = true
	module := NewReflectionModule(mockMmu, mockReasoning, mockScripting, config)
	
	// Trigger reflection
	insights, err := module.TriggerReflection(ctx)
	
	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, insights)
	assert.Len(t, insights, 2)
	assert.Equal(t, "pattern", insights[0].Type)
	assert.Equal(t, "Detected a recurring theme about X", insights[0].Description)
	assert.InDelta(t, 0.85, insights[0].Confidence, 0.001)
	assert.Contains(t, insights[0].RelatedMemoryIDs, "record1")
	
	// Verify mocks were called
	mockMmu.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
	mockScripting.AssertExpectations(t)
}

func TestTriggerReflectionWithoutLuaHooks(t *testing.T) {
	// Create mocks
	mockMmu := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptEngine)
	mockScripting.On("Close").Return(nil).Maybe()
	
	// Create context with entity
	ctx := context.Background()
	ctx = entity.ContextWithEntity(ctx, entity.Context{
		EntityID: "test-entity",
		UserID:   "test-user",
	})
	
	// Create test memory records
	records := []ltm.MemoryRecord{
		{
			ID:         "record1",
			EntityID:   "test-entity",
			Content:    "Memory content 1",
			CreatedAt:  time.Now().Add(-1 * time.Hour),
			AccessLevel: entity.SharedWithinEntity,
		},
	}
	
	// Mock the retrieval call
	mockMmu.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return(records, nil)
	
	// Sample LLM response with insights
	llmResponse := `
	{
		"insights": [
			{
				"type": "pattern",
				"description": "Detected a pattern",
				"confidence": 0.85,
				"related_memory_ids": ["record1"]
			}
		]
	}`
	
	// Mock the reasoning engine call
	mockReasoning.On("Process", ctx, mock.Anything, mock.Anything).Return(llmResponse, nil)
	
	// Mock the consolidation call
	mockMmu.On("ConsolidateLTM", ctx, mock.Anything).Return(nil)
	
	// Create the module with hooks disabled
	config := DefaultConfig()
	config.EnableLuaHooks = false
	module := NewReflectionModule(mockMmu, mockReasoning, mockScripting, config)
	
	// Trigger reflection
	insights, err := module.TriggerReflection(ctx)
	
	// Verify results
	assert.NoError(t, err)
	assert.NotNil(t, insights)
	assert.Len(t, insights, 1)
	
	// Verify mocks were called (and scripting engine was NOT called)
	mockMmu.AssertExpectations(t)
	mockReasoning.AssertExpectations(t)
	mockScripting.AssertNotCalled(t, "ExecuteFunction", mock.Anything, mock.Anything)
}

func TestTriggerReflectionErrorHandling(t *testing.T) {
	// Create mocks
	mockMmu := new(MockMMU)
	mockReasoning := new(MockReasoningEngine)
	mockScripting := new(MockScriptEngine)
	mockScripting.On("Close").Return(nil).Maybe()
	
	// Create context with entity
	ctx := context.Background()
	ctx = entity.ContextWithEntity(ctx, entity.Context{
		EntityID: "test-entity",
		UserID:   "test-user",
	})
	
	// Mock the retrieval call with an error
	mockMmu.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{}, assert.AnError)
	
	// Create the module
	module := NewReflectionModule(mockMmu, mockReasoning, mockScripting, DefaultConfig())
	
	// Trigger reflection
	insights, err := module.TriggerReflection(ctx)
	
	// Verify error is returned
	assert.Error(t, err)
	assert.Nil(t, insights)
	assert.ErrorIs(t, err, assert.AnError)
	
	// Create new mocks for testing reasoning error
	mockMmu2 := new(MockMMU)
	mockReasoning2 := new(MockReasoningEngine)
	
	// Mock successful retrieval
	mockMmu2.On("RetrieveFromLTM", ctx, mock.Anything, mock.Anything).Return([]ltm.MemoryRecord{
		{ID: "record1", Content: "test"},
	}, nil)
	
	// Mock reasoning error
	mockReasoning2.On("Process", ctx, mock.Anything, mock.Anything).Return("", assert.AnError)
	
	// Create module with new mocks
	module2 := NewReflectionModule(mockMmu2, mockReasoning2, nil, DefaultConfig())
	
	// Trigger reflection
	insights2, err2 := module2.TriggerReflection(ctx)
	
	// Verify reasoning error is returned
	assert.Error(t, err2)
	assert.Nil(t, insights2)
	assert.ErrorIs(t, err2, assert.AnError)
}