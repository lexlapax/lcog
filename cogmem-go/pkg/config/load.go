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
	// Example: Override LTM SQL DSN from environment
	if dsn := os.Getenv("COGMEM_LTM_SQL_DSN"); dsn != "" {
		config.LTM.SQL.DSN = dsn
	}
	
	// Example: Override Redis configuration from environment
	if addr := os.Getenv("COGMEM_REDIS_ADDR"); addr != "" {
		config.LTM.KV.Redis.Addr = addr
	}
	
	// Add more overrides as needed
}

// validateConfig validates the configuration.
func validateConfig(config *Config) error {
	// Validate LTM configuration
	switch strings.ToLower(config.LTM.Type) {
	case "sql":
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
		default:
			return fmt.Errorf("unsupported KV provider: %s", config.LTM.KV.Provider)
		}
	case "vector", "graph":
		// Placeholder for Phase 2 and 3 validation
		return fmt.Errorf("LTM type %s not yet implemented", config.LTM.Type)
	default:
		return fmt.Errorf("unsupported LTM type: %s", config.LTM.Type)
	}
	
	// Validate reasoning configuration
	if config.Reasoning.Provider != "mock" {
		switch strings.ToLower(config.Reasoning.Provider) {
		case "openai":
			if config.Reasoning.OpenAI.APIKey == "" {
				return fmt.Errorf("OpenAI API key is required for openai provider")
			}
		case "anthropic":
			if config.Reasoning.Anthropic.APIKey == "" {
				return fmt.Errorf("Anthropic API key is required for anthropic provider")
			}
		default:
			return fmt.Errorf("unsupported reasoning provider: %s", config.Reasoning.Provider)
		}
	}
	
	return nil
}
