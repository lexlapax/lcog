package config

// Config represents the top-level configuration for the CogMem library.
type Config struct {
	// LTM configures the long-term memory storage
	LTM LTMConfig `yaml:"ltm"`
	
	// Scripting configures the Lua scripting engine
	Scripting ScriptingConfig `yaml:"scripting"`
	
	// Reasoning configures the reasoning engine (LLM)
	Reasoning ReasoningConfig `yaml:"reasoning"`
	
	// Logging configures the logging behavior
	Logging LoggingConfig `yaml:"logging"`
}

// LTMConfig configures the long-term memory storage.
type LTMConfig struct {
	// Type specifies the LTM backend ("sql", "kv", "vector", "graph")
	Type string `yaml:"type"`
	
	// SQL configures SQL-based storage
	SQL SQLConfig `yaml:"sql"`
	
	// KV configures key-value storage
	KV KVConfig `yaml:"kv"`
}

// SQLConfig configures SQL-based LTM storage.
type SQLConfig struct {
	// Driver is the SQL driver ("postgres", "sqlite")
	Driver string `yaml:"driver"`
	
	// DSN is the data source name (connection string)
	DSN string `yaml:"dsn"`
}

// KVConfig configures key-value LTM storage.
type KVConfig struct {
	// Provider is the KV provider ("redis", "postgres_hstore")
	Provider string `yaml:"provider"`
	
	// Redis configures Redis connection
	Redis RedisConfig `yaml:"redis"`
	
	// PostgresHStore configures PostgreSQL with HStore
	PostgresHStore PostgresHStoreConfig `yaml:"postgres_hstore"`
}

// RedisConfig configures Redis connection.
type RedisConfig struct {
	// Addr is the Redis server address
	Addr string `yaml:"addr"`
	
	// Password is the Redis password (optional)
	Password string `yaml:"password"`
	
	// DB is the Redis database number
	DB int `yaml:"db"`
}

// PostgresHStoreConfig configures PostgreSQL with HStore.
type PostgresHStoreConfig struct {
	// DSN is the data source name (connection string)
	DSN string `yaml:"dsn"`
}

// ScriptingConfig configures the Lua scripting engine.
type ScriptingConfig struct {
	// Paths is a list of directories containing Lua scripts
	Paths []string `yaml:"paths"`
}

// ReasoningConfig configures the reasoning engine (LLM).
type ReasoningConfig struct {
	// Provider is the LLM provider ("openai", "anthropic", "mock")
	Provider string `yaml:"provider"`
	
	// OpenAI configures OpenAI integration
	OpenAI OpenAIConfig `yaml:"openai"`
	
	// Anthropic configures Anthropic integration
	Anthropic AnthropicConfig `yaml:"anthropic"`
}

// OpenAIConfig configures OpenAI integration.
type OpenAIConfig struct {
	// APIKey is the OpenAI API key
	APIKey string `yaml:"api_key"`
	
	// Model is the OpenAI model to use
	Model string `yaml:"model"`
}

// AnthropicConfig configures Anthropic integration.
type AnthropicConfig struct {
	// APIKey is the Anthropic API key
	APIKey string `yaml:"api_key"`
	
	// Model is the Anthropic model to use
	Model string `yaml:"model"`
}

// LoggingConfig configures logging behavior.
type LoggingConfig struct {
	// Level is the logging level ("debug", "info", "warn", "error")
	Level string `yaml:"level"`
}
