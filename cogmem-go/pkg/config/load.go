package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	return LoadFromBytes(data)
}

// LoadFromBytes loads configuration from a byte slice.
func LoadFromBytes(data []byte) (*Config, error) {
	var config Config
	
	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	
	// Apply environment variable overrides
	applyEnvironmentOverrides(&config)
	
	// Validate configuration
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return &config, nil
}

// applyEnvironmentOverrides applies environment variable overrides to the config.
func applyEnvironmentOverrides(config *Config) {
	// LTM SQL DSN override
	if dsn := os.Getenv("COGMEM_LTM_SQL_DSN"); dsn != "" {
		config.LTM.SQL.DSN = dsn
	}
	
	// Redis configuration override
	if addr := os.Getenv("COGMEM_REDIS_ADDR"); addr != "" {
		config.LTM.KV.Redis.Addr = addr
	}
	
	// ChromemGo URL override
	if url := os.Getenv("COGMEM_CHROMEMGO_URL"); url != "" {
		config.LTM.ChromemGo.URL = url
	}
	
	// PgVector connection string override
	if connStr := os.Getenv("PGVECTOR_URL"); connStr != "" {
		config.LTM.PgVector.ConnectionString = connStr
	}
	
	// OpenAI API key override
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		config.Reasoning.OpenAI.APIKey = apiKey
	}
	
	// Anthropic API key override
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		config.Reasoning.Anthropic.APIKey = apiKey
	}
}

// validateConfig validates the configuration.
func validateConfig(config *Config) error {
	// Validate LTM configuration
	switch strings.ToLower(config.LTM.Type) {
	case "sql", "sqlstore":
		if config.LTM.SQL.Driver == "" {
			return fmt.Errorf("sql driver is required for sql LTM type")
		}
		if config.LTM.SQL.DSN == "" {
			return fmt.Errorf("sql DSN is required for sql LTM type")
		}
	case "kv":
		if config.LTM.KV.Provider == "" {
			return fmt.Errorf("kv provider is required for kv LTM type")
		}
		switch strings.ToLower(config.LTM.KV.Provider) {
		case "redis":
			if config.LTM.KV.Redis.Addr == "" {
				return fmt.Errorf("redis address is required for redis KV provider")
			}
		case "postgres_hstore":
			if config.LTM.KV.PostgresHStore.DSN == "" {
				return fmt.Errorf("postgres DSN is required for postgres_hstore KV provider")
			}
		case "boltdb":
			// BoltDB doesn't require additional validation for now
		default:
			return fmt.Errorf("unsupported KV provider: %s", config.LTM.KV.Provider)
		}
	case "chromemgo", "pgvector", "vector":
		// Vector store validation depends on specific type
		switch strings.ToLower(config.LTM.Type) {
		case "chromemgo":
			// Validate ChromemGo configuration
			if config.LTM.ChromemGo.Collection == "" {
				return fmt.Errorf("collection name is required for chromemgo LTM type")
			}
		case "pgvector":
			// Validate PgVector configuration
			if config.LTM.PgVector.ConnectionString == "" {
				return fmt.Errorf("connection string is required for pgvector LTM type")
			}
			if config.LTM.PgVector.TableName == "" {
				// Apply default table name
				config.LTM.PgVector.TableName = "memory_vectors"
			}
			if config.LTM.PgVector.Dimensions <= 0 {
				// Apply default dimensions
				config.LTM.PgVector.Dimensions = 1536
			}
			if config.LTM.PgVector.DistanceMetric == "" {
				// Apply default distance metric
				config.LTM.PgVector.DistanceMetric = "cosine"
			} else {
				// Validate distance metric
				metric := strings.ToLower(config.LTM.PgVector.DistanceMetric)
				if metric != "cosine" && metric != "euclidean" && metric != "dot" {
					return fmt.Errorf("unsupported distance metric for pgvector: %s (must be cosine, euclidean, or dot)", 
						config.LTM.PgVector.DistanceMetric)
				}
			}
		default:
			// General vector type not fully implemented yet
			return fmt.Errorf("generic vector LTM type not yet fully implemented")
		}
	case "mock":
		// Mock store doesn't require additional validation
	case "graph":
		// Graph store not implemented yet
		return fmt.Errorf("graph LTM type not yet implemented")
	default:
		return fmt.Errorf("unsupported LTM type: %s", config.LTM.Type)
	}
	
	// Validate reasoning configuration
	if config.Reasoning.Provider != "mock" {
		switch strings.ToLower(config.Reasoning.Provider) {
		case "openai":
			// API key can be provided via environment variable, so we don't explicitly check for it here
			// But validate model settings
			if config.Reasoning.OpenAI.Model == "" {
				// Apply default
				config.Reasoning.OpenAI.Model = "gpt-4"
			}
			if config.Reasoning.OpenAI.EmbeddingModel == "" {
				// Apply default
				config.Reasoning.OpenAI.EmbeddingModel = "text-embedding-3-small"
			}
		case "anthropic":
			if config.Reasoning.Anthropic.APIKey == "" {
				return fmt.Errorf("Anthropic API key is required for anthropic provider")
			}
			if config.Reasoning.Anthropic.Model == "" {
				// Apply default
				config.Reasoning.Anthropic.Model = "claude-3-opus-20240229"
			}
		default:
			return fmt.Errorf("unsupported reasoning provider: %s", config.Reasoning.Provider)
		}
	}
	
	// Validate reflection configuration (apply defaults if needed)
	if config.Reflection.TriggerFrequency <= 0 {
		config.Reflection.TriggerFrequency = 10 // Default trigger frequency
	}
	
	if config.Reflection.MaxMemoriesToAnalyze <= 0 {
		config.Reflection.MaxMemoriesToAnalyze = 50 // Default max memories
	}
	
	// Validate the temperature range
	if config.Reflection.AnalysisTemperature < 0 || config.Reflection.AnalysisTemperature > 1.0 {
		config.Reflection.AnalysisTemperature = 0.3 // Default temperature
	}
	
	return nil
}