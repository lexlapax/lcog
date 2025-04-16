package cogmem

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lexlapax/cogmem/pkg/config"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/kv/boltdb"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/kv/postgres"
	ltmMock "github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/mock"
	sqlstorePostgres "github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/sqlstore/postgres"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/sqlstore/sqlite"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/vector/chromem_go"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/vector/pgvector"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	reasoningMock "github.com/lexlapax/cogmem/pkg/reasoning/adapters/mock"
	reasoningOpenAI "github.com/lexlapax/cogmem/pkg/reasoning/adapters/openai"
	"github.com/lexlapax/cogmem/pkg/reflection"
	"github.com/lexlapax/cogmem/pkg/scripting"
	
	"database/sql"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"
	chromem "github.com/philippgille/chromem-go"
)

// InputType represents the type of input received by the client.
type InputType string

const (
	// InputTypeStore indicates a request to store information.
	InputTypeStore InputType = "store"
	
	// InputTypeRetrieve indicates a request to retrieve information.
	InputTypeRetrieve InputType = "retrieve"
	
	// InputTypeQuery indicates a request to query/process information.
	InputTypeQuery InputType = "query"
)

// CogMemClient is the main facade for the CogMem library.
type CogMemClient interface {
	// Process handles input and produces a response based on the client's capabilities.
	Process(ctx context.Context, inputType InputType, input string) (string, error)
}

// Config contains configuration options for the client.
type Config struct {
	// EnableReflection determines whether reflection is active
	EnableReflection bool
	
	// ReflectionFrequency is how often reflection occurs (in ops count)
	ReflectionFrequency int
}

// DefaultConfig returns the default configuration for the client.
func DefaultConfig() Config {
	return Config{
		EnableReflection:    true,
		ReflectionFrequency: 10,
	}
}

// CogMemClientImpl is the implementation of the CogMemClient interface.
type CogMemClientImpl struct {
	// memoryManager is the MMU for memory operations
	memoryManager mmu.MMU
	
	// reasoningEngine is the engine for generating responses
	reasoningEngine reasoning.Engine
	
	// scriptingEngine is the Lua scripting engine
	scriptingEngine scripting.Engine
	
	// reflectionModule is the module for self-reflection
	reflectionModule reflection.ReflectionModule
	
	// config contains client configuration options
	config Config
	
	// opCount tracks operations for triggering reflection
	opCount int
	
	// operationHistory stores recent operations for reflection
	operationHistory []OperationRecord
}

// OperationRecord represents a single operation performed by the client
type OperationRecord struct {
	InputType InputType `json:"input_type"`
	Input     string    `json:"input"`
	Response  string    `json:"response"`
}

// NewCogMem creates a new CogMemClient with the specified dependencies.
func NewCogMem(
	memoryManager mmu.MMU,
	reasoningEngine reasoning.Engine,
	scriptingEngine scripting.Engine,
	reflectionModule reflection.ReflectionModule,
	config Config,
) *CogMemClientImpl {
	client := &CogMemClientImpl{
		memoryManager:    memoryManager,
		reasoningEngine:  reasoningEngine,
		scriptingEngine:  scriptingEngine,
		reflectionModule: reflectionModule,
		config:           config,
		opCount:          0,
		operationHistory: make([]OperationRecord, 0, 10), // Keep last 10 operations for reflection
	}
	
	log.Debug("CogMemClient initialized", 
		"reflection_enabled", config.EnableReflection,
		"reflection_frequency", config.ReflectionFrequency,
	)
	
	return client
}

// Process implements the CogMemClient interface.
func (c *CogMemClientImpl) Process(ctx context.Context, inputType InputType, input string) (string, error) {
	// Extract entity context - required for all operations
	entityCtx, ok := entity.GetEntityContext(ctx)
	if !ok {
		return "", entity.ErrMissingEntityContext
	}
	
	log.DebugContext(ctx, "Processing input", 
		"entity_id", entityCtx.EntityID,
		"input_type", inputType,
		"input_length", len(input),
	)
	
	// Increment operation count
	c.opCount++
	
	// Process based on input type
	var response string
	var err error
	
	switch inputType {
	case InputTypeStore:
		response, err = c.handleStore(ctx, input)
	case InputTypeRetrieve:
		response, err = c.handleRetrieve(ctx, input)
	case InputTypeQuery:
		response, err = c.handleQuery(ctx, input)
	default:
		err = fmt.Errorf("unsupported input type: %s", inputType)
		return "", err
	}
	
	// If operation was successful, record it for reflection
	if err == nil {
		c.recordOperation(inputType, input, response)
		
		// Check if reflection should be triggered
		if c.shouldReflect() {
			log.DebugContext(ctx, "Triggering reflection after operation", 
				"operation_count", c.opCount,
				"reflection_frequency", c.config.ReflectionFrequency,
			)
			c.reflect(ctx)
		}
	}
	
	return response, err
}

// handleStore processes a store operation
func (c *CogMemClientImpl) handleStore(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling store operation", "content_length", len(input))
	
	// Store the information in LTM
	memoryID, err := c.memoryManager.EncodeToLTM(ctx, input)
	if err != nil {
		log.ErrorContext(ctx, "Failed to store memory", "error", err)
		return "", err
	}
	
	log.DebugContext(ctx, "Memory stored successfully", "memory_id", memoryID)
	return fmt.Sprintf("Memory stored successfully with ID: %s", memoryID), nil
}

// handleRetrieve processes a retrieve operation
func (c *CogMemClientImpl) handleRetrieve(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling retrieve operation", "query", input)
	
	// Use default retrieval options
	options := mmu.DefaultRetrievalOptions()
	
	// Retrieve relevant memories from LTM
	memories, err := c.memoryManager.RetrieveFromLTM(ctx, input, options)
	if err != nil {
		log.ErrorContext(ctx, "Failed to retrieve memories", "error", err)
		return "", err
	}
	
	// If no memories found, return a simple message
	if len(memories) == 0 {
		log.DebugContext(ctx, "No memories found for query")
		return "No memories found for the query.", nil
	}
	
	log.DebugContext(ctx, "Retrieved memories", "count", len(memories))
	
	// Format memories to show the user
	var memoriesText strings.Builder
	memoriesText.WriteString(fmt.Sprintf("Found %d memories matching your query:\n\n", len(memories)))
	
	for i, memory := range memories {
		memoriesText.WriteString(fmt.Sprintf("Memory %d: %s\n", i+1, memory.Content))
		
		// Add metadata if available
		if memory.Metadata != nil && len(memory.Metadata) > 0 {
			createdAt, ok := memory.Metadata["encoded_at"].(string)
			if ok {
				memoriesText.WriteString(fmt.Sprintf("  Created: %s\n", createdAt))
			}
		}
		memoriesText.WriteString("\n")
	}
	
	// For lookups, return the actual memories instead of a summary
	result := memoriesText.String()
	log.DebugContext(ctx, "Returning memory list", "memory_count", len(memories))
	return result, nil
}

// handleQuery processes a query operation
func (c *CogMemClientImpl) handleQuery(ctx context.Context, input string) (string, error) {
	log.DebugContext(ctx, "Handling query operation", "query", input)
	
	// Check if this is a special semantic search request
	isSemanticSearchRequest := false
	semanticPrefix := "SEMANTIC_SEARCH: "
	if strings.HasPrefix(input, semanticPrefix) {
		isSemanticSearchRequest = true
		// Remove the prefix for processing
		input = strings.TrimPrefix(input, semanticPrefix)
		log.DebugContext(ctx, "Detected semantic search request", "query", input)
	}
	
	// Configure retrieval for semantic search
	options := mmu.RetrievalOptions{
		MaxResults:     5,   // Limit to most relevant memories
		Strategy:       "semantic",
		IncludeMetadata: true,
	}
	
	// Create a semantic query for related memories
	query := map[string]interface{}{
		"text": input, 
	}
	
	// Retrieve relevant context from LTM
	memories, err := c.memoryManager.RetrieveFromLTM(ctx, query, options)
	if err != nil {
		log.ErrorContext(ctx, "Failed to retrieve context for query", "error", err)
		return "", err
	}
	
	// If this is a semantic search request, return the results directly
	if isSemanticSearchRequest {
		if len(memories) == 0 {
			return "No memories found matching your semantic search.", nil
		}
		
		var resultBuilder strings.Builder
		resultBuilder.WriteString(fmt.Sprintf("Found %d memories semantically related to your search:\n\n", len(memories)))
		
		for i, memory := range memories {
			resultBuilder.WriteString(fmt.Sprintf("Memory %d: %s\n", i+1, memory.Content))
			
			// Include similarity score if available
			if memory.Metadata != nil {
				if score, ok := memory.Metadata["score"].(float64); ok {
					resultBuilder.WriteString(fmt.Sprintf("  Similarity: %.2f%%\n", score*100))
				}
			}
			
			// Add creation time if available
			if !memory.CreatedAt.IsZero() {
				resultBuilder.WriteString(fmt.Sprintf("  Created: %s\n", memory.CreatedAt.Format(time.RFC3339)))
			}
			resultBuilder.WriteString("\n")
		}
		
		return resultBuilder.String(), nil
	}
	
	// For regular queries, build prompt with context if available
	var prompt string
	if len(memories) > 0 {
		log.DebugContext(ctx, "Found relevant context memories", "count", len(memories))
		
		// Format memories for context
		var contextBuilder strings.Builder
		contextBuilder.WriteString("Context from memory:\n")
		
		for i, memory := range memories {
			contextBuilder.WriteString(fmt.Sprintf("Memory %d: %s\n", i+1, memory.Content))
		}
		
		prompt = fmt.Sprintf(
			"Using the following context, please answer this question:\n\n%s\n\nQuestion: %s",
			contextBuilder.String(),
			input,
		)
	} else {
		log.DebugContext(ctx, "No relevant context found for query")
		prompt = fmt.Sprintf("Please answer this question: %s", input)
	}
	
	// Process the query with the reasoning engine
	response, err := c.reasoningEngine.Process(ctx, prompt)
	if err != nil {
		log.ErrorContext(ctx, "Failed to process query", "error", err)
		return "", err
	}
	
	log.DebugContext(ctx, "Query processed successfully", "response_length", len(response))
	return response, nil
}

// recordOperation adds an operation to the history for reflection
func (c *CogMemClientImpl) recordOperation(inputType InputType, input, response string) {
	record := OperationRecord{
		InputType: inputType,
		Input:     input,
		Response:  response,
	}
	
	// Keep last 10 operations maximum
	c.operationHistory = append(c.operationHistory, record)
	if len(c.operationHistory) > 10 {
		c.operationHistory = c.operationHistory[1:]
	}
}

// shouldReflect determines if reflection should be triggered
func (c *CogMemClientImpl) shouldReflect() bool {
	// Skip if reflection is disabled
	if !c.config.EnableReflection {
		return false
	}
	
	// Check if enough operations have been performed
	return c.opCount > 0 && c.opCount%c.config.ReflectionFrequency == 0
}

// reflect performs reflection on recent operations
func (c *CogMemClientImpl) reflect(ctx context.Context) {
	// Skip if there's no reflection module or no operations to reflect on
	if c.reflectionModule == nil || len(c.operationHistory) == 0 {
		return
	}
	
	log.DebugContext(ctx, "Performing reflection", "history_length", len(c.operationHistory))
	
	// Also store the recent operation history in LTM before reflection
	historyJSON, err := json.Marshal(c.operationHistory)
	if err != nil {
		log.ErrorContext(ctx, "Failed to marshal operation history for reflection", "error", err)
		return
	}
	
	// Store the history with metadata
	historyData := map[string]interface{}{
		"content": string(historyJSON),
		"metadata": map[string]interface{}{
			"type":           "operation_history",
			"operation_count": c.opCount,
			"timestamp":      time.Now().Format(time.RFC3339),
		},
	}
	
	// Store the history in LTM (ignore errors, this is just for context)
	_, _ = c.memoryManager.EncodeToLTM(ctx, historyData)
	
	// Trigger the reflection process
	insights, err := c.reflectionModule.TriggerReflection(ctx)
	if err != nil {
		log.ErrorContext(ctx, "Error during reflection process", "error", err)
		return
	}
	
	log.DebugContext(ctx, "Reflection completed", 
		"insight_count", len(insights),
		"operation_count", c.opCount)
}

// NewCogMemFromConfig creates a new CogMemClient using the provided configuration file.
// This is a convenience function that handles all component initialization.
func NewCogMemFromConfig(configPath string) (*CogMemClientImpl, error) {
	// Load configuration
	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize the LTM store based on configuration
	ltmStore, err := initLTMStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LTM store: %w", err)
	}

	// Initialize the scripting engine
	scriptEngine, err := initScriptEngine(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize scripting engine: %w", err)
	}

	// Initialize the reasoning engine
	reasoningEngine, err := initReasoningEngine(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize reasoning engine: %w", err)
	}

	// Initialize the MMU
	mmuInstance := mmu.NewMMU(
		ltmStore,
		reasoningEngine,
		scriptEngine,
		mmu.DefaultConfig(),
	)

	// Initialize the Reflection Module
	reflectionModule := reflection.NewReflectionModule(
		mmuInstance,
		reasoningEngine,
		scriptEngine,
		reflection.DefaultConfig(),
	)

	// Create the client instance
	clientConfig := DefaultConfig()
	if cfg.Reflection.Enabled {
		clientConfig.EnableReflection = true
		clientConfig.ReflectionFrequency = cfg.Reflection.TriggerFrequency
	}

	// Create and return the client
	client := NewCogMem(
		mmuInstance,
		reasoningEngine,
		scriptEngine,
		reflectionModule,
		clientConfig,
	)

	log.Info("CogMem client initialized from config", 
		"ltm_type", cfg.LTM.Type,
		"reasoning_provider", cfg.Reasoning.Provider,
		"reflection_enabled", clientConfig.EnableReflection,
	)

	return client, nil
}

// initLTMStore initializes the appropriate LTM store based on configuration
func initLTMStore(cfg *config.Config) (ltm.LTMStore, error) {
	ltmType := strings.ToLower(cfg.LTM.Type)
	log.Info("Initializing LTM store", "type", ltmType)
	
	// Special case for pgvector
	if ltmType == "pgvector" {
		return initPgVectorStore(cfg)
	}

	switch ltmType {
	case "mock", "":
		// Use mock store for testing/demo
		log.Info("Using mock LTM store")
		return ltmMock.NewMockStore(), nil

	case "sql", "sqlstore":
		sqlDriver := strings.ToLower(cfg.LTM.SQL.Driver)
		if sqlDriver == "sqlite" || sqlDriver == "" {
			// Ensure directory exists
			dbPath := cfg.LTM.SQL.DSN
			if dbPath == "" {
				dbPath = "./data/cogmem.db" // Default path
			}
			dirPath := filepath.Dir(dbPath)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory for SQLite DB: %w", err)
			}

			log.Info("Using SQLite LTM store", "path", dbPath)
			db, err := sql.Open("sqlite3", dbPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open SQLite database: %w", err)
			}

			// Create the memory_records table if it doesn't exist
			_, err = db.Exec(`
				CREATE TABLE IF NOT EXISTS memory_records (
					id TEXT PRIMARY KEY,
					entity_id TEXT NOT NULL,
					user_id TEXT,
					access_level INTEGER NOT NULL,
					content TEXT NOT NULL,
					metadata TEXT,
					created_at TEXT,
					updated_at TEXT
				);
				CREATE INDEX IF NOT EXISTS idx_memory_records_entity_id ON memory_records(entity_id);
			`)
			if err != nil {
				return nil, fmt.Errorf("failed to create memory_records table: %w", err)
			}

			return sqlite.NewSQLiteStore(db), nil
		} else if sqlDriver == "postgres" {
			// Get PostgreSQL connection string
			dsn := cfg.LTM.SQL.DSN
			if dsn == "" {
				// Try to get it from environment
				dsn = os.Getenv("POSTGRES_URL")
				if dsn == "" {
					return nil, fmt.Errorf("PostgreSQL connection string not provided")
				}
			}

			log.Info("Using PostgreSQL SQL store")

			// Need to use pgxpool for PostgreSQL SQLStore
			log.Info("Creating pgxpool for PostgreSQL SQL store")
			ctx := context.Background()
			pool, err := pgxpool.New(ctx, dsn)
			if err != nil {
				return nil, fmt.Errorf("failed to create PostgreSQL connection pool: %w", err)
			}
			
			// Create the store with pgxpool
			store := sqlstorePostgres.NewPostgresStore(pool)

			return store, nil
		}
		return nil, fmt.Errorf("unsupported SQL driver: %s", sqlDriver)

	case "kv":
		kvProvider := strings.ToLower(cfg.LTM.KV.Provider)
		if kvProvider == "boltdb" || kvProvider == "" {
			// Ensure directory exists
			dbPath := "./data/cogmem.bolt.db" // Default path
			dirPath := filepath.Dir(dbPath)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create directory for BoltDB: %w", err)
			}

			log.Info("Using BoltDB LTM store", "path", dbPath)
			db, err := bolt.Open(dbPath, 0600, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to open BoltDB database: %w", err)
			}

			store := boltdb.NewBoltStore(db)
			if err := store.Initialize(context.Background()); err != nil {
				return nil, fmt.Errorf("failed to initialize BoltDB store: %w", err)
			}

			return store, nil
		} else if kvProvider == "postgres_hstore" {
			// Get PostgreSQL connection string
			dsn := cfg.LTM.KV.PostgresHStore.DSN
			if dsn == "" {
				// Try to get it from environment
				dsn = os.Getenv("POSTGRES_URL")
				if dsn == "" {
					return nil, fmt.Errorf("PostgreSQL connection string not provided")
				}
			}

			log.Info("Using PostgreSQL HStore KV store")
			db, err := sqlx.Connect("postgres", dsn)
			if err != nil {
				return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
			}

			// Create the store
			store := postgres.NewHstoreStore(db)
			
			// Use custom table name if provided in code since there's no TableName field in config
			// Default to memory_records_hstore
			store.WithTableName("memory_records_hstore")
			
			// Initialize the store (enable hstore extension if needed)
			if err := store.Initialize(context.Background()); err != nil {
				return nil, fmt.Errorf("failed to initialize PostgreSQL HStore: %w", err)
			}

			return store, nil
		}
		return nil, fmt.Errorf("unsupported KV provider: %s", kvProvider)
		
	case "chromemgo", "vector":
		// Initialize ChromemGo vector store
		log.Info("Initializing ChromemGo vector store")
		
		// Get configuration values or use defaults
		url := cfg.LTM.ChromemGo.URL
		if url == "" {
			url = "http://localhost:8080" // Default
		}
		
		collectionName := cfg.LTM.ChromemGo.Collection
		if collectionName == "" {
			collectionName = "memories" // Default
		}
		
		// Initialize the ChromemGo DB client (in-memory mode)
		chromemClient := chromem.NewDB()
		
		// Get embedding dimensions from config
		dimensions := cfg.LTM.ChromemGo.Dimensions
		if dimensions == 0 {
			dimensions = 1536 // Default for OpenAI embeddings
		}
		
		// Create the adapter with the client
		chromemAdapter, err := chromem_go.NewChromemGoAdapter(chromemClient, collectionName)
		if err != nil {
			return nil, fmt.Errorf("failed to create ChromemGo adapter: %w", err)
		}
		
		log.Info("Using ChromemGo vector store", 
			"collection", collectionName,
			"dimensions", dimensions)
		
		return chromemAdapter, nil
		
	default:
		return nil, fmt.Errorf("unsupported LTM store type: %s", ltmType)
	}
}

// initPgVectorStore initializes the PostgreSQL pgvector store
func initPgVectorStore(cfg *config.Config) (ltm.LTMStore, error) {
	log.Info("Initializing PostgreSQL pgvector store")
	
	// Get configuration parameters
	connectionString := cfg.LTM.PgVector.ConnectionString
	
	// Environment variable substitution
	if strings.Contains(connectionString, "${POSTGRES_URL}") {
		connectionString = strings.Replace(connectionString, "${POSTGRES_URL}", os.Getenv("POSTGRES_URL"), 1)
	}
	
	if connectionString == "" {
		// Try to get it from environment
		connectionString = os.Getenv("POSTGRES_URL")
		if connectionString == "" {
			return nil, fmt.Errorf("PostgreSQL connection string not provided")
		}
	}
	
	log.Debug("Using PostgreSQL connection string", "connection_string", connectionString)
	
	tableName := cfg.LTM.PgVector.TableName
	if tableName == "" {
		tableName = "memory_vectors" // Default
	}
	
	dimensions := cfg.LTM.PgVector.Dimensions
	if dimensions == 0 {
		dimensions = 1536 // Default for OpenAI embeddings
	}
	
	distanceMetric := cfg.LTM.PgVector.DistanceMetric
	if distanceMetric == "" {
		distanceMetric = "cosine" // Default
	}
	
	// Create pgvector configuration
	pgvectorConfig := pgvector.PgvectorConfig{
		ConnectionString: connectionString,
		TableName:        tableName,
		DimensionSize:    dimensions,
		DistanceMetric:   distanceMetric,
	}
	
	log.Info("Using PostgreSQL pgvector store", 
		"table", tableName,
		"dimensions", dimensions,
		"distance_metric", distanceMetric)
	
	// Create the adapter
	ctx := context.Background()
	pgvectorAdapter, err := pgvector.NewPgvectorAdapter(ctx, pgvectorConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgvector adapter: %w", err)
	}
	
	return pgvectorAdapter, nil
}

// initScriptEngine initializes the Lua scripting engine
func initScriptEngine(cfg *config.Config) (scripting.Engine, error) {
	// Get script paths from config
	scriptPaths := cfg.Scripting.Paths
	if len(scriptPaths) == 0 {
		// Default script paths if none provided
		scriptPaths = []string{"./scripts", "../scripts", "../../scripts"}
	}

	// Create script engine
	scriptEngine, err := scripting.NewLuaEngine(scripting.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create Lua engine: %w", err)
	}

	// Try to load scripts from each path
	scriptFound := false
	for _, basePath := range scriptPaths {
		basePath, err := filepath.Abs(basePath)
		if err != nil {
			log.Warn("Failed to get absolute path", "path", basePath, "error", err)
			continue
		}

		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			log.Debug("Scripts directory not found", "path", basePath)
			continue
		}

		// Load scripts from directory
		err = scriptEngine.LoadScriptDir(basePath)
		if err != nil {
			log.Warn("Failed to load scripts", "path", basePath, "error", err)
			continue
		}

		log.Info("Loaded scripts", "path", basePath)
		scriptFound = true
	}

	if !scriptFound {
		log.Warn("No scripts were loaded from any path")
	}

	return scriptEngine, nil
}

// initReasoningEngine initializes the reasoning engine based on configuration
func initReasoningEngine(cfg *config.Config) (reasoning.Engine, error) {
	provider := strings.ToLower(cfg.Reasoning.Provider)
	log.Info("Initializing reasoning engine", "provider", provider)

	switch provider {
	case "openai":
		// Check if the API key is set
		apiKey := cfg.Reasoning.OpenAI.APIKey
		if apiKey == "" {
			// Try to get it from environment
			apiKey = os.Getenv("OPENAI_API_KEY")
		}
		
		if apiKey == "" {
			log.Warn("OpenAI API key not found, falling back to mock engine")
			return reasoningMock.NewMockEngine(), nil
		}
		
		// Initialize the OpenAI adapter
		openaiCfg := reasoningOpenAI.Config{
			APIKey:         apiKey,
			ChatModel:      cfg.Reasoning.OpenAI.Model,
			EmbeddingModel: cfg.Reasoning.OpenAI.EmbeddingModel,
		}
		
		// If model isn't set, use defaults
		if openaiCfg.ChatModel == "" {
			openaiCfg.ChatModel = "gpt-4" // Default model
		}
		if openaiCfg.EmbeddingModel == "" {
			openaiCfg.EmbeddingModel = "text-embedding-3-small" // Default embedding model
		}
		
		log.Info("Using OpenAI reasoning engine", 
			"chat_model", openaiCfg.ChatModel,
			"embedding_model", openaiCfg.EmbeddingModel)
		
		openaiAdapter, err := reasoningOpenAI.NewOpenAIAdapter(openaiCfg)
		if err != nil {
			log.Error("Failed to initialize OpenAI adapter, falling back to mock", "error", err)
			return reasoningMock.NewMockEngine(), nil
		}
		
		return openaiAdapter, nil

	case "anthropic":
		// In Phase 2, we only support OpenAI and mock
		log.Warn("Anthropic provider not yet implemented, using mock reasoning engine")
		return reasoningMock.NewMockEngine(), nil

	case "mock", "":
		// Create a mock engine with some canned responses
		mockEngine := reasoningMock.NewMockEngine()

		// Add some helpful canned responses
		mockEngine.AddResponse("help", "I'm here to assist with memory management. You can store memories, retrieve them, and ask questions.")
		mockEngine.AddResponse("store", "Your memory has been stored successfully.")
		mockEngine.AddResponse("retrieve", "Here are the memories I've found that match your query.")
		
		// Default response for summarizing memories
		mockEngine.SetDefaultResponse("I've analyzed the memories and here's what I found: The information seems to relate to your previous interactions and stored knowledge.")

		log.Info("Using mock reasoning engine")
		return mockEngine, nil

	default:
		log.Warn("Unsupported reasoning provider, using mock engine", "provider", provider)
		return reasoningMock.NewMockEngine(), nil
	}
}