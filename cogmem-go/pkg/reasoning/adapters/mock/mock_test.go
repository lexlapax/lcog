package mock

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/reasoning"
)

func TestMockEngine_Process(t *testing.T) {
	tests := []struct {
		name           string
		mockSetup      func(*MockEngine)
		prompt         string
		opts           []reasoning.Option
		expectedResult string
		expectError    bool
	}{
		{
			name: "exact match canned response",
			mockSetup: func(m *MockEngine) {
				m.AddResponse("hello", "Hello, world!")
				m.SetExactMatch(true)
			},
			prompt:         "hello",
			expectedResult: "Hello, world!",
		},
		{
			name: "substring match canned response",
			mockSetup: func(m *MockEngine) {
				m.AddResponse("hello", "Hello, world!")
				m.SetExactMatch(false)
			},
			prompt:         "Say hello to everyone",
			expectedResult: "Hello, world!",
		},
		{
			name: "default response when no match",
			mockSetup: func(m *MockEngine) {
				m.SetDefaultResponse("I don't know how to respond to that.")
			},
			prompt:         "unknown prompt",
			expectedResult: "I don't know how to respond to that.",
		},
		{
			name: "process with custom error",
			mockSetup: func(m *MockEngine) {
				m.SetShouldError(true)
			},
			prompt:      "anything",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create engine with test setup
			engine := NewMockEngine()
			if tt.mockSetup != nil {
				tt.mockSetup(engine)
			}

			// Process the prompt
			ctx := context.Background()
			result, err := engine.Process(ctx, tt.prompt, tt.opts...)

			// Check expectations
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			// Verify call was recorded in history
			assert.Len(t, engine.GetCallHistory(), 1)
			call := engine.GetCallHistory()[0]
			assert.Equal(t, "Process", call.Method)
			assert.Equal(t, tt.prompt, call.Args[1]) // Args[0] is context, Args[1] is prompt
		})
	}
}

func TestMockEngine_ProcessWithEntityContext(t *testing.T) {
	// Create engine
	engine := NewMockEngine()
	
	// Create entity context
	entityCtx := entity.NewContext("test-entity-123", "test-entity")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Process with context
	_, err := engine.Process(ctx, "hello")
	require.NoError(t, err)
	
	// Verify entity context was captured in call history
	calls := engine.GetCallHistory()
	require.Len(t, calls, 1)
	
	// Get the context from the call
	callCtx, ok := calls[0].Args[0].(context.Context)
	require.True(t, ok, "First argument should be context")
	
	// Extract entity from context
	extractedEntity, ok := entity.GetEntityContext(callCtx)
	require.True(t, ok, "Entity context should be present")
	assert.Equal(t, entity.EntityID("test-entity-123"), extractedEntity.EntityID)
	assert.Equal(t, "test-entity", extractedEntity.UserID)
}

func TestMockEngine_GenerateEmbeddings(t *testing.T) {
	tests := []struct {
		name            string
		mockSetup       func(*MockEngine)
		texts           []string
		expectedResults [][]float32
		expectError     bool
	}{
		{
			name: "exact match canned embeddings",
			mockSetup: func(m *MockEngine) {
				m.AddEmbedding("hello", []float32{0.1, 0.2, 0.3})
				m.SetExactMatch(true)
			},
			texts:           []string{"hello"},
			expectedResults: [][]float32{{0.1, 0.2, 0.3}},
		},
		{
			name: "substring match canned embeddings",
			mockSetup: func(m *MockEngine) {
				m.AddEmbedding("hello", []float32{0.1, 0.2, 0.3})
				m.SetExactMatch(false)
			},
			texts:           []string{"Say hello to everyone"},
			expectedResults: [][]float32{{0.1, 0.2, 0.3}},
		},
		{
			name: "default embedding when no match",
			mockSetup: func(m *MockEngine) {
				m.SetDefaultEmbedding([]float32{0.5, 0.5, 0.5})
			},
			texts:           []string{"unknown text"},
			expectedResults: [][]float32{{0.5, 0.5, 0.5}},
		},
		{
			name: "multiple texts",
			mockSetup: func(m *MockEngine) {
				m.AddEmbedding("text1", []float32{0.1, 0.2, 0.3})
				m.AddEmbedding("text2", []float32{0.4, 0.5, 0.6})
				m.SetDefaultEmbedding([]float32{0.7, 0.8, 0.9})
			},
			texts: []string{"text1", "text2", "text3"},
			expectedResults: [][]float32{
				{0.1, 0.2, 0.3},
				{0.4, 0.5, 0.6},
				{0.7, 0.8, 0.9},
			},
		},
		{
			name: "generate with custom error",
			mockSetup: func(m *MockEngine) {
				m.SetShouldError(true)
			},
			texts:       []string{"anything"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create engine with test setup
			engine := NewMockEngine()
			if tt.mockSetup != nil {
				tt.mockSetup(engine)
			}

			// Generate embeddings
			ctx := context.Background()
			results, err := engine.GenerateEmbeddings(ctx, tt.texts)

			// Check expectations
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResults, results)
			}

			// Verify call was recorded in history
			assert.Len(t, engine.GetCallHistory(), 1)
			call := engine.GetCallHistory()[0]
			assert.Equal(t, "GenerateEmbeddings", call.Method)
			assert.Equal(t, tt.texts, call.Args[1]) // Args[0] is context, Args[1] is texts
		})
	}
}

func TestMockEngine_Options(t *testing.T) {
	// Create engine with options
	engine := NewMockEngine(
		WithDefaultResponse("Default response"),
		WithDefaultEmbedding([]float32{0.1, 0.2, 0.3}),
		WithExactMatch(true),
	)

	// Add some responses and embeddings
	engine.AddResponse("hello", "Hello, world!")
	engine.AddEmbedding("text", []float32{0.4, 0.5, 0.6})

	// Test default response
	ctx := context.Background()
	result, err := engine.Process(ctx, "unknown")
	assert.NoError(t, err)
	assert.Equal(t, "Default response", result)

	// Test default embedding
	embeddings, err := engine.GenerateEmbeddings(ctx, []string{"unknown"})
	assert.NoError(t, err)
	assert.Equal(t, [][]float32{{0.1, 0.2, 0.3}}, embeddings)

	// Test that exact match is working
	result, err = engine.Process(ctx, "Say hello")
	assert.NoError(t, err)
	assert.Equal(t, "Default response", result) // Should not match "hello" with exact match

	// Change to non-exact match
	engine.SetExactMatch(false)
	result, err = engine.Process(ctx, "Say hello")
	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!", result) // Should now match "hello" within the text
}

func TestMockEngine_ClearHistory(t *testing.T) {
	// Create engine
	engine := NewMockEngine()

	// Make some calls
	ctx := context.Background()
	_, _ = engine.Process(ctx, "prompt1")
	_, _ = engine.Process(ctx, "prompt2")
	_, _ = engine.GenerateEmbeddings(ctx, []string{"text1", "text2"})

	// Verify history length
	assert.Len(t, engine.GetCallHistory(), 3)

	// Clear history
	engine.ClearHistory()

	// Verify history is cleared
	assert.Len(t, engine.GetCallHistory(), 0)
}
