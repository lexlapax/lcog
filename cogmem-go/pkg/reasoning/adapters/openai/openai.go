package openai

import (
	"context"
	"errors"
	"strings"

	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	"github.com/sashabaranov/go-openai"
)

var (
	// ErrInvalidConfig is returned when the adapter configuration is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")
	// ErrEmptyAPIKey is returned when the API key is missing.
	ErrEmptyAPIKey = errors.New("API key cannot be empty")
)

// Config holds the configuration for the OpenAI adapter.
type Config struct {
	// APIKey is the OpenAI API key.
	APIKey string
	// EmbeddingModel is the model to use for embeddings, e.g., "text-embedding-ada-002".
	EmbeddingModel string
	// ChatModel is the model to use for chat completions, e.g., "gpt-4".
	ChatModel string
	// BaseURL is the base URL for the OpenAI API (for testing).
	BaseURL string
}

// OpenAIAdapter implements the reasoning.Engine interface using the OpenAI API.
type OpenAIAdapter struct {
	client         *openai.Client
	embeddingModel string
	chatModel      string
}

// NewOpenAIAdapter creates a new OpenAI adapter.
func NewOpenAIAdapter(config Config) (*OpenAIAdapter, error) {
	if config.APIKey == "" {
		return nil, ErrEmptyAPIKey
	}

	// Set default models if not specified
	if config.EmbeddingModel == "" {
		config.EmbeddingModel = "text-embedding-ada-002"
	}
	if config.ChatModel == "" {
		config.ChatModel = "gpt-4"
	}

	// Create OpenAI client configuration
	clientConfig := openai.DefaultConfig(config.APIKey)
	if config.BaseURL != "" {
		clientConfig.BaseURL = config.BaseURL
	}

	// Create OpenAI client
	client := openai.NewClientWithConfig(clientConfig)

	return &OpenAIAdapter{
		client:         client,
		embeddingModel: config.EmbeddingModel,
		chatModel:      config.ChatModel,
	}, nil
}

// GenerateEmbeddings generates embeddings for the given texts using the OpenAI API.
func (a *OpenAIAdapter) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	log.Debug("Generating embeddings", "count", len(texts), "model", a.embeddingModel)

	// Create embedding request
	request := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(a.embeddingModel),
	}

	// Call OpenAI API
	response, err := a.client.CreateEmbeddings(ctx, request)
	if err != nil {
		log.Error("Failed to generate embeddings", "error", err)
		return nil, err
	}

	// Extract embeddings from response
	embeddings := make([][]float32, len(response.Data))
	for i, data := range response.Data {
		embeddings[i] = data.Embedding
	}

	log.Debug("Successfully generated embeddings", 
		"count", len(embeddings),
		"dimension", len(embeddings[0]),
		"model", a.embeddingModel)

	return embeddings, nil
}

// ProcessMessages generates a response to the given messages using the OpenAI API.
func (a *OpenAIAdapter) ProcessMessages(ctx context.Context, messages []map[string]string, opts ...reasoning.Option) (string, error) {
	// Apply options
	options := reasoning.DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Override model if specified in options
	model := a.chatModel
	if options.Model != "" {
		model = options.Model
	}

	log.Debug("Processing chat request", "model", model, "messages", len(messages))

	// Convert messages from map to OpenAI message format
	chatMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		role := msg["role"]
		content := msg["content"]
		chatMessages[i] = openai.ChatCompletionMessage{
			Role:    role,
			Content: content,
		}
	}

	// Create chat completion request
	request := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    chatMessages,
		Temperature: float32(options.Temperature),
		MaxTokens:   options.MaxTokens,
	}

	// Call OpenAI API
	response, err := a.client.CreateChatCompletion(ctx, request)
	if err != nil {
		log.Error("Failed to generate chat completion", "error", err)
		return "", err
	}

	// Extract response content
	if len(response.Choices) == 0 {
		return "", errors.New("no response choices returned")
	}

	content := response.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	log.Debug("Successfully generated response", 
		"tokens", response.Usage.TotalTokens,
		"model", model)

	return content, nil
}

// Process implements the reasoning.Engine interface by adapting to our messages format.
func (a *OpenAIAdapter) Process(ctx context.Context, prompt string, opts ...reasoning.Option) (string, error) {
	// Create a simple user message from the prompt
	messages := []map[string]string{
		{"role": "user", "content": prompt},
	}
	
	return a.ProcessMessages(ctx, messages, opts...)
}