package reasoning

import (
	"context"
)

// Option is a function that configures a reasoning process.
type Option func(*Options)

// Options holds configuration for a reasoning request.
type Options struct {
	// Temperature controls randomness in generation (0.0-1.0)
	Temperature float64
	
	// MaxTokens limits the length of the generated response
	MaxTokens int
	
	// Model specifies which model variant to use
	Model string
}

// DefaultOptions returns default reasoning options.
func DefaultOptions() Options {
	return Options{
		Temperature: 0.7,
		MaxTokens:   1024,
		Model:       "", // Empty means use the adapter's default
	}
}

// WithTemperature sets the temperature option.
func WithTemperature(temp float64) Option {
	return func(o *Options) {
		o.Temperature = temp
	}
}

// WithMaxTokens sets the max tokens option.
func WithMaxTokens(tokens int) Option {
	return func(o *Options) {
		o.MaxTokens = tokens
	}
}

// WithModel sets the model option.
func WithModel(model string) Option {
	return func(o *Options) {
		o.Model = model
	}
}

// Engine is the interface for reasoning engines (LLMs).
type Engine interface {
	// Process sends a prompt to the reasoning engine and returns the result.
	Process(ctx context.Context, prompt string, opts ...Option) (string, error)
	
	// GenerateEmbeddings creates vector embeddings for the provided texts.
	GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error)
}
