package openai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lexlapax/cogmem/pkg/reasoning/adapters/openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOpenAIServer creates a mock OpenAI server for testing.
func mockOpenAIServer(t *testing.T, statusCode int, responseBody string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, err := w.Write([]byte(responseBody))
		require.NoError(t, err)
	}))
	return server
}

// TestGenerateEmbeddings_Success tests successful embedding generation.
func TestGenerateEmbeddings_Success(t *testing.T) {
	// Create mock response
	mockResponse := `{
		"object": "list",
		"data": [
			{
				"object": "embedding",
				"embedding": [0.1, 0.2, 0.3, 0.4, 0.5],
				"index": 0
			},
			{
				"object": "embedding",
				"embedding": [0.6, 0.7, 0.8, 0.9, 1.0],
				"index": 1
			}
		],
		"model": "text-embedding-ada-002",
		"usage": {
			"prompt_tokens": 10,
			"total_tokens": 10
		}
	}`

	// Create a mock server
	server := mockOpenAIServer(t, http.StatusOK, mockResponse)
	defer server.Close()

	// Create adapter with mock server URL
	config := openai.Config{
		APIKey:        "test-key",
		EmbeddingModel: "text-embedding-ada-002",
		BaseURL:       server.URL,
	}
	adapter, err := openai.NewOpenAIAdapter(config)
	require.NoError(t, err)

	// Test GenerateEmbeddings
	texts := []string{"Hello world", "Testing embeddings"}
	embeddings, err := adapter.GenerateEmbeddings(context.Background(), texts)
	require.NoError(t, err)
	require.Len(t, embeddings, 2)

	// Verify the embeddings
	assert.Equal(t, []float32{0.1, 0.2, 0.3, 0.4, 0.5}, embeddings[0])
	assert.Equal(t, []float32{0.6, 0.7, 0.8, 0.9, 1.0}, embeddings[1])
}

// TestGenerateEmbeddings_EmptyInput tests handling of empty input.
func TestGenerateEmbeddings_EmptyInput(t *testing.T) {
	config := openai.Config{
		APIKey:        "test-key",
		EmbeddingModel: "text-embedding-ada-002",
	}
	adapter, err := openai.NewOpenAIAdapter(config)
	require.NoError(t, err)

	// Test with empty input
	embeddings, err := adapter.GenerateEmbeddings(context.Background(), []string{})
	assert.NoError(t, err)
	assert.Empty(t, embeddings)
}

// TestGenerateEmbeddings_APIError tests handling of API errors.
func TestGenerateEmbeddings_APIError(t *testing.T) {
	// Create error response
	errorResponse := `{
		"error": {
			"message": "The API key is invalid",
			"type": "invalid_request_error",
			"param": null,
			"code": "invalid_api_key"
		}
	}`

	// Create a mock server
	server := mockOpenAIServer(t, http.StatusUnauthorized, errorResponse)
	defer server.Close()

	// Create adapter with mock server URL
	config := openai.Config{
		APIKey:        "invalid-key",
		EmbeddingModel: "text-embedding-ada-002",
		BaseURL:       server.URL,
	}
	adapter, err := openai.NewOpenAIAdapter(config)
	require.NoError(t, err)

	// Test GenerateEmbeddings with invalid key
	texts := []string{"Hello world"}
	embeddings, err := adapter.GenerateEmbeddings(context.Background(), texts)
	assert.Error(t, err)
	assert.Nil(t, embeddings)
	assert.Contains(t, err.Error(), "invalid")
}

// TestProcess_Success tests successful LLM processing.
func TestProcess_Success(t *testing.T) {
	// Create mock response
	mockResponse := `{
		"id": "chatcmpl-123",
		"object": "chat.completion",
		"created": 1677858242,
		"model": "gpt-4",
		"choices": [
			{
				"message": {
					"role": "assistant",
					"content": "This is a test response"
				},
				"finish_reason": "stop",
				"index": 0
			}
		],
		"usage": {
			"prompt_tokens": 10,
			"completion_tokens": 10,
			"total_tokens": 20
		}
	}`

	// Create a mock server
	server := mockOpenAIServer(t, http.StatusOK, mockResponse)
	defer server.Close()

	// Create adapter with mock server URL
	config := openai.Config{
		APIKey:     "test-key",
		ChatModel:  "gpt-4",
		BaseURL:    server.URL,
	}
	adapter, err := openai.NewOpenAIAdapter(config)
	require.NoError(t, err)

	// Test Process
	ctx := context.Background()
	prompt := "Hello, how are you?"
	response, err := adapter.Process(ctx, prompt)
	require.NoError(t, err)
	assert.Equal(t, "This is a test response", response)
}

// TestProcess_APIError tests handling of API errors in Process.
func TestProcess_APIError(t *testing.T) {
	// Create error response
	errorResponse := `{
		"error": {
			"message": "Rate limit exceeded",
			"type": "rate_limit_error",
			"param": null,
			"code": "rate_limit_exceeded"
		}
	}`

	// Create a mock server
	server := mockOpenAIServer(t, http.StatusTooManyRequests, errorResponse)
	defer server.Close()

	// Create adapter with mock server URL
	config := openai.Config{
		APIKey:     "test-key",
		ChatModel:  "gpt-4",
		BaseURL:    server.URL,
	}
	adapter, err := openai.NewOpenAIAdapter(config)
	require.NoError(t, err)

	// Test Process with rate limit error
	ctx := context.Background()
	prompt := "Hello, how are you?"
	response, err := adapter.Process(ctx, prompt)
	assert.Error(t, err)
	assert.Empty(t, response)
	assert.Contains(t, err.Error(), "Rate limit")
}

// TestInitialization tests initialization with different configurations.
func TestInitialization(t *testing.T) {
	// Test with valid config
	config := openai.Config{
		APIKey:         "test-key",
		EmbeddingModel: "text-embedding-ada-002",
		ChatModel:      "gpt-4",
	}
	adapter, err := openai.NewOpenAIAdapter(config)
	assert.NoError(t, err)
	assert.NotNil(t, adapter)

	// Test with empty API key
	invalidConfig := openai.Config{
		APIKey:         "",
		EmbeddingModel: "text-embedding-ada-002",
	}
	adapter, err = openai.NewOpenAIAdapter(invalidConfig)
	assert.Error(t, err)
	assert.Nil(t, adapter)
}