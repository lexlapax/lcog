package mock

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/reasoning"
)

// Call represents a recorded method call on the mock engine.
type Call struct {
	// Method is the name of the method that was called.
	Method string
	
	// Args contains the arguments passed to the method.
	Args []interface{}
}

// MockEngine implements the reasoning.Engine interface with canned responses.
type MockEngine struct {
	// cannedResponses maps prompts to predetermined responses
	cannedResponses map[string]string
	
	// defaultResponse is returned when no matching canned response is found
	defaultResponse string
	
	// cannedEmbeddings maps text to predetermined embeddings
	cannedEmbeddings map[string][]float32
	
	// defaultEmbedding is returned when no matching canned embedding is found
	defaultEmbedding []float32
	
	// exactMatch determines if prompt matching is exact or uses Contains
	exactMatch bool
	
	// shouldError indicates if the engine should return errors
	shouldError bool
	
	// mutex protects the maps from concurrent access
	mutex sync.RWMutex
	
	// callHistory records all calls to Process and GenerateEmbeddings
	callHistory []Call
}

// MockOption is a function that configures a MockEngine.
type MockOption func(*MockEngine)

// WithDefaultResponse sets the default response for the mock engine.
func WithDefaultResponse(resp string) MockOption {
	return func(m *MockEngine) {
		m.defaultResponse = resp
	}
}

// WithDefaultEmbedding sets the default embedding for the mock engine.
func WithDefaultEmbedding(embedding []float32) MockOption {
	return func(m *MockEngine) {
		m.defaultEmbedding = embedding
	}
}

// WithExactMatch configures whether the mock engine uses exact matching.
func WithExactMatch(exact bool) MockOption {
	return func(m *MockEngine) {
		m.exactMatch = exact
	}
}

// WithShouldError configures whether the mock engine returns errors.
func WithShouldError(shouldErr bool) MockOption {
	return func(m *MockEngine) {
		m.shouldError = shouldErr
	}
}

// NewMockEngine creates a new MockEngine with the given options.
func NewMockEngine(opts ...MockOption) *MockEngine {
	m := &MockEngine{
		cannedResponses:  make(map[string]string),
		defaultResponse:  "This is a mock response",
		cannedEmbeddings: make(map[string][]float32),
		defaultEmbedding: []float32{0.0, 0.0, 0.0},
		exactMatch:       false, // Default to substring matching
		shouldError:      false,
		callHistory:      make([]Call, 0),
	}
	
	// Apply options
	for _, opt := range opts {
		opt(m)
	}
	
	log.Debug("Created mock reasoning engine", "exact_match", m.exactMatch)
	return m
}

// Process implements the reasoning.Engine interface.
func (m *MockEngine) Process(ctx context.Context, prompt string, opts ...reasoning.Option) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Record the call
	m.callHistory = append(m.callHistory, Call{
		Method: "Process",
		Args:   []interface{}{ctx, prompt, opts},
	})
	
	// Check if should return error
	if m.shouldError {
		return "", errors.New("mock reasoning engine error")
	}
	
	// Apply reasoning options
	options := reasoning.DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	
	log.Debug("Processing prompt with mock engine", 
		"prompt_length", len(prompt),
		"temperature", options.Temperature,
		"max_tokens", options.MaxTokens,
		"model", options.Model)
	
	// Find a matching response
	if m.exactMatch {
		// Exact match
		if response, ok := m.cannedResponses[prompt]; ok {
			return response, nil
		}
	} else {
		// Substring match
		for key, response := range m.cannedResponses {
			if strings.Contains(prompt, key) {
				return response, nil
			}
		}
	}
	
	// Return default response if no match found
	return m.defaultResponse, nil
}

// GenerateEmbeddings implements the reasoning.Engine interface.
func (m *MockEngine) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Record the call
	m.callHistory = append(m.callHistory, Call{
		Method: "GenerateEmbeddings",
		Args:   []interface{}{ctx, texts},
	})
	
	// Check if should return error
	if m.shouldError {
		return nil, errors.New("mock reasoning engine error")
	}
	
	log.Debug("Generating embeddings with mock engine", "text_count", len(texts))
	
	// Generate embeddings for each text
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		if m.exactMatch {
			// Exact match
			if embedding, ok := m.cannedEmbeddings[text]; ok {
				embeddings[i] = embedding
				continue
			}
		} else {
			// Substring match
			var matched bool
			for key, embedding := range m.cannedEmbeddings {
				if strings.Contains(text, key) {
					embeddings[i] = embedding
					matched = true
					break
				}
			}
			if matched {
				continue
			}
		}
		
		// Use default embedding if no match found
		embeddings[i] = m.defaultEmbedding
	}
	
	return embeddings, nil
}

// AddResponse adds a canned response for a specific prompt.
func (m *MockEngine) AddResponse(prompt, response string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.cannedResponses[prompt] = response
	log.Debug("Added canned response", "prompt", prompt)
}

// SetDefaultResponse sets the default response.
func (m *MockEngine) SetDefaultResponse(response string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.defaultResponse = response
	log.Debug("Set default response")
}

// AddEmbedding adds a canned embedding for a specific text.
func (m *MockEngine) AddEmbedding(text string, embedding []float32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.cannedEmbeddings[text] = embedding
	log.Debug("Added canned embedding", "text", text, "dimensions", len(embedding))
}

// SetDefaultEmbedding sets the default embedding.
func (m *MockEngine) SetDefaultEmbedding(embedding []float32) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.defaultEmbedding = embedding
	log.Debug("Set default embedding", "dimensions", len(embedding))
}

// SetExactMatch configures whether the engine uses exact matching.
func (m *MockEngine) SetExactMatch(exact bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.exactMatch = exact
	log.Debug("Set exact match mode", "exact_match", exact)
}

// SetShouldError configures whether the engine returns errors.
func (m *MockEngine) SetShouldError(shouldErr bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.shouldError = shouldErr
	log.Debug("Set should error mode", "should_error", shouldErr)
}

// GetCallHistory returns a copy of the call history.
func (m *MockEngine) GetCallHistory() []Call {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	// Return a copy to prevent race conditions
	history := make([]Call, len(m.callHistory))
	copy(history, m.callHistory)
	
	return history
}

// ClearHistory clears the call history.
func (m *MockEngine) ClearHistory() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.callHistory = make([]Call, 0)
	log.Debug("Cleared call history")
}
