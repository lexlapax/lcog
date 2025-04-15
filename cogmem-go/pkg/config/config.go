package config

// Config represents the top-level configuration for the CogMem library.
type Config struct {
	// LTM configures the long-term memory storage
	LTM LTMConfig `yaml:"ltm"`
	
	// Scripting configures the Lua scripting engine
	Scripting ScriptingConfig `yaml:"scripting"`
	
	// Reasoning configures the reasoning engine (LLM)
	Reasoning ReasoningConfig `yaml:"reasoning"`
	
	// Reflection configures the reflection module
	Reflection ReflectionConfig `yaml:"reflection"`
	
	// Logging configures the logging behavior
	Logging LoggingConfig `yaml:"logging"`
}

// LTMConfig configures the long-term memory storage.
type LTMConfig struct {
	// Type specifies the LTM backend ("sql", "kv", "vector", "graph", "chromemgo", "pgvector")
	Type string `yaml:"type"`
	
	// SQL configures SQL-based storage
	SQL SQLConfig `yaml:"sql"`
	
	// KV configures key-value storage
	KV KVConfig `yaml:"kv"`
	
	// ChromemGo configures ChromemGo vector storage
	ChromemGo ChromemGoConfig `yaml:"chromemgo"`
	
	// PgVector configures PostgreSQL pgvector storage
	PgVector PgVectorConfig `yaml:"pgvector"`
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
	
	// Model is the OpenAI model to use for chat/completion
	Model string `yaml:"model"`
	
	// EmbeddingModel is the model to use for generating embeddings
	EmbeddingModel string `yaml:"embedding_model"`
	
	// MaxTokens is the maximum number of tokens to generate
	MaxTokens int `yaml:"max_tokens"`
	
	// Temperature controls randomness in generation (0.0-1.0)
	Temperature float64 `yaml:"temperature"`
}

// AnthropicConfig configures Anthropic integration.
type AnthropicConfig struct {
	// APIKey is the Anthropic API key
	APIKey string `yaml:"api_key"`
	
	// Model is the Anthropic model to use
	Model string `yaml:"model"`
}

// ChromemGoConfig configures ChromemGo vector storage.
type ChromemGoConfig struct {
	// URL is the ChromemGo server address
	URL string `yaml:"url"`
	
	// Collection is the collection name to use
	Collection string `yaml:"collection"`
	
	// Dimensions specifies the embedding dimensions
	Dimensions int `yaml:"dimensions"`
	
	// StoragePath is the path for on-disk persistent storage (if empty, in-memory is used)
	StoragePath string `yaml:"storage_path"`
}

// PgVectorConfig configures PostgreSQL with pgvector extension
type PgVectorConfig struct {
	// ConnectionString is the PostgreSQL connection string
	ConnectionString string `yaml:"connection_string"`
	
	// TableName is the name of the table to use
	TableName string `yaml:"table_name"`
	
	// Dimensions specifies the embedding dimensions
	Dimensions int `yaml:"dimensions"`
	
	// DistanceMetric is the distance metric to use (cosine, euclidean, dot)
	DistanceMetric string `yaml:"distance_metric"`
}

// ReflectionConfig configures the reflection module.
type ReflectionConfig struct {
	// Enabled determines whether reflection is active
	Enabled bool `yaml:"enabled"`
	
	// TriggerFrequency is the number of interactions between reflection cycles
	TriggerFrequency int `yaml:"trigger_frequency"`
	
	// MaxMemoriesToAnalyze sets the maximum number of memories to include in analysis
	MaxMemoriesToAnalyze int `yaml:"max_memories_to_analyze"`
	
	// AnalysisModel specifies the model to use for analysis (uses default if empty)
	AnalysisModel string `yaml:"analysis_model"`
	
	// AnalysisTemperature sets the temperature for reasoning during analysis
	AnalysisTemperature float64 `yaml:"analysis_temperature"`
}

// LoggingConfig configures logging behavior.
type LoggingConfig struct {
	// Level is the logging level ("debug", "info", "warn", "error")
	Level string `yaml:"level"`
}