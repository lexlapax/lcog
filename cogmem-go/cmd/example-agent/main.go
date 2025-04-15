package main

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/peterh/liner"
	_ "github.com/mattn/go-sqlite3"
	bolt "go.etcd.io/bbolt"

	"github.com/lexlapax/cogmem/pkg/agent"
	"github.com/lexlapax/cogmem/pkg/config"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/kv/boltdb"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/mock"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/sqlstore/sqlite"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/reasoning"
	reasoningMock "github.com/lexlapax/cogmem/pkg/reasoning/adapters/mock"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

// Constants for the command-line interface
const (
	cmdHelp     = "!help"
	cmdQuit     = "!quit"
	cmdEntity   = "!entity"
	cmdUser     = "!user"
	cmdRemember = "!remember"
	cmdLookup   = "!lookup"
	cmdQuery    = "!query"
	cmdConfig   = "!config"
)

// Command-line help text
const helpText = `
CogMem Example Agent - Command Reference:
-----------------------------------------
!help                 - Show this help message
!entity <id>          - Set the current entity ID
!user <id>            - Set the current user ID
!remember <text>      - Store a memory in LTM
!lookup <query>       - Retrieve memories matching query
!query <question>     - Ask a question using context from memories
!config               - Show current configuration
!quit                 - Exit the application

Notes:
- Regular text input is treated as a query
- Tab completion is available for commands
- Use up/down arrows for command history`

// historyFile is the file where command history is stored
const historyFile = ".cogmem_history"

func main() {
	// Initialize logger
	log.Setup(log.Config{
		Level:  log.InfoLevel,
		Format: log.TextFormat,
	})

	log.Info("Starting CogMem example agent")

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Initialize LTM store based on configuration
	ltmStore, err := initLTMStore(cfg)
	if err != nil {
		log.Error("Failed to initialize LTM store", "error", err)
		os.Exit(1)
	}

	// Initialize the Lua scripting engine
	scriptEngine, err := initScriptEngine(cfg)
	if err != nil {
		log.Error("Failed to initialize script engine", "error", err)
		os.Exit(1)
	}
	defer scriptEngine.Close()

	// Initialize the reasoning engine (mock for this example)
	reasoningEngine := initReasoningEngine(cfg)

	// Initialize the MMU
	mmuInstance := mmu.NewMMU(
		ltmStore,
		reasoningEngine,
		scriptEngine,
		mmu.DefaultConfig(),
	)

	// Initialize the Agent with all components
	agentInstance := agent.NewAgent(
		mmuInstance,
		reasoningEngine,
		scriptEngine,
		agent.DefaultConfig(),
	)

	// Start the command-line interface
	runCLI(agentInstance, cfg)
}

// loadConfig loads the application configuration
func loadConfig() (*config.Config, error) {
	// Look for config file in standard locations
	configPaths := []string{
		"./configs/config.yaml",
		"./config.yaml",
		"../configs/config.yaml",
		"../../configs/config.yaml",
	}

	var cfg *config.Config
	var err error

	// Try each path
	for _, path := range configPaths {
		if _, statErr := os.Stat(path); statErr == nil {
			log.Info("Loading configuration", "path", path)
			cfg, err = config.LoadFromFile(path)
			if err == nil {
				return cfg, nil
			}
			log.Warn("Failed to load config file", "path", path, "error", err)
		}
	}

	// If no config found, use the example config
	examplePath := "./configs/config.example.yaml"
	if _, statErr := os.Stat(examplePath); statErr == nil {
		log.Info("Loading example configuration", "path", examplePath)
		cfg, err = config.LoadFromFile(examplePath)
		if err == nil {
			return cfg, nil
		}
		log.Warn("Failed to load example config", "error", err)
	}

	// If still no config, use defaults with mock store
	log.Info("Using default configuration with mock store")
	
	// Create a minimal default config
	defaultCfg := &config.Config{
		LTM: config.LTMConfig{
			Type: "mock",
		},
		Scripting: config.ScriptingConfig{
			Paths: []string{"./scripts", "../scripts", "../../scripts"},
		},
		Reasoning: config.ReasoningConfig{
			Provider: "mock",
		},
		Logging: config.LoggingConfig{
			Level: "info",
		},
	}

	return defaultCfg, nil
}

// initLTMStore initializes the appropriate LTM store based on configuration
func initLTMStore(cfg *config.Config) (ltm.LTMStore, error) {
	ltmType := strings.ToLower(cfg.LTM.Type)
	log.Info("Initializing LTM store", "type", ltmType)

	switch ltmType {
	case "mock", "":
		// Use mock store for testing/demo
		log.Info("Using mock LTM store")
		return mock.NewMockStore(), nil

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
		}
		return nil, fmt.Errorf("unsupported SQL driver: %s", sqlDriver)

	case "kv":
		kvProvider := strings.ToLower(cfg.LTM.KV.Provider)
		if kvProvider == "boltdb" || kvProvider == "" {
			// Ensure directory exists
			// For BoltDB, we'll use a default path since it's not in the config structure yet
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
		}
		return nil, fmt.Errorf("unsupported KV provider: %s", kvProvider)

	default:
		return nil, fmt.Errorf("unsupported LTM store type: %s", ltmType)
	}
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
func initReasoningEngine(cfg *config.Config) reasoning.Engine {
	provider := strings.ToLower(cfg.Reasoning.Provider)
	log.Info("Initializing reasoning engine", "provider", provider)

	switch provider {
	case "openai":
		// In Phase 1, we only support the mock engine
		log.Warn("OpenAI provider not yet implemented, using mock reasoning engine")
		return reasoningMock.NewMockEngine()

	case "anthropic":
		// In Phase 1, we only support the mock engine
		log.Warn("Anthropic provider not yet implemented, using mock reasoning engine")
		return reasoningMock.NewMockEngine()

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
		return mockEngine

	default:
		log.Warn("Unsupported reasoning provider, using mock engine", "provider", provider)
		return reasoningMock.NewMockEngine()
	}
}

// runCLI starts the command-line interface for user interaction
func runCLI(agentInstance *agent.AgentI, cfg *config.Config) {
	// Initialize with default entity and user
	currentEntity := entity.EntityID("default-entity")
	currentUser := "default-user"
	entityCtx := entity.NewContext(currentEntity, currentUser)

	// Create and configure the liner (command line) state
	line := liner.NewLiner()
	defer line.Close()

	// Enable history
	line.SetCtrlCAborts(true)
	line.SetMultiLineMode(false)
	
	// Set tab completion
	line.SetCompleter(func(line string) (c []string) {
		commands := []string{cmdHelp, cmdQuit, cmdEntity, cmdUser, cmdRemember, cmdLookup, cmdQuery, cmdConfig}
		for _, cmd := range commands {
			if strings.HasPrefix(cmd, line) {
				c = append(c, cmd)
			}
		}
		return
	})

	// Load history from file if it exists
	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
	
	// Save history when exiting
	defer func() {
		if f, err := os.Create(historyFile); err == nil {
			line.WriteHistory(f)
			f.Close()
		}
	}()

	// Print welcome message
	fmt.Println("\n=== CogMem Example Agent ===")
	fmt.Println("LTM Store:", cfg.LTM.Type)
	if cfg.LTM.Type == "sql" {
		fmt.Println("SQL Driver:", cfg.LTM.SQL.Driver)
	} else if cfg.LTM.Type == "kv" {
		fmt.Println("KV Provider:", cfg.LTM.KV.Provider)
	}
	fmt.Printf("Current Entity: %s | Current User: %s\n", currentEntity, currentUser)
	fmt.Println("Type !help for available commands.")

	// Main loop
	for {
		// Read user input
		prompt := fmt.Sprintf("cogmem::%s@%s> ", currentUser, currentEntity)
		input, err := line.Prompt(prompt)
		
		if err != nil {
			if err == liner.ErrPromptAborted || err == io.EOF {
				fmt.Println("\nGoodbye!")
				break
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		// Skip empty input
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		// Add to history
		line.AppendHistory(input)

		// Process commands
		if strings.HasPrefix(input, "!") {
			parts := strings.SplitN(input, " ", 2)
			cmd := parts[0]

			switch cmd {
			case cmdHelp:
				fmt.Println(helpText)

			case cmdQuit:
				fmt.Println("Goodbye!")
				return

			case cmdEntity:
				if len(parts) == 1 {
					fmt.Printf("Current entity: %s\n", currentEntity)
					// Prompt for entity ID if not provided
					entityIDInput, err := line.Prompt("Enter new entity ID (or press Enter to keep current): ")
					if err == nil && strings.TrimSpace(entityIDInput) != "" {
						currentEntity = entity.EntityID(strings.TrimSpace(entityIDInput))
						entityCtx = entity.NewContext(currentEntity, currentUser)
						fmt.Printf("Entity set to: %s\n", currentEntity)
					}
				} else {
					currentEntity = entity.EntityID(parts[1])
					entityCtx = entity.NewContext(currentEntity, currentUser)
					fmt.Printf("Entity set to: %s\n", currentEntity)
				}

			case cmdUser:
				if len(parts) == 1 {
					fmt.Printf("Current user: %s\n", currentUser)
					// Prompt for user ID if not provided
					userIDInput, err := line.Prompt("Enter new user ID (or press Enter to keep current): ")
					if err == nil && strings.TrimSpace(userIDInput) != "" {
						currentUser = strings.TrimSpace(userIDInput)
						entityCtx = entity.NewContext(currentEntity, currentUser)
						fmt.Printf("User set to: %s\n", currentUser)
					}
				} else {
					currentUser = parts[1]
					entityCtx = entity.NewContext(currentEntity, currentUser)
					fmt.Printf("User set to: %s\n", currentUser)
				}

			case cmdRemember:
				memory := ""
				if len(parts) == 1 {
					// Prompt for memory content if not provided
					var err error
					memory, err = line.Prompt("Enter memory to store: ")
					if err != nil || strings.TrimSpace(memory) == "" {
						fmt.Println("Memory storage cancelled")
						continue
					}
				} else {
					memory = parts[1]
				}
				
				ctx := entity.ContextWithEntity(context.Background(), entityCtx)
				response, err := agentInstance.Process(ctx, agent.InputTypeStore, memory)
				if err != nil {
					fmt.Printf("Error storing memory: %v\n", err)
				} else {
					fmt.Println(response)
				}

			case cmdLookup:
				query := ""
				if len(parts) == 1 {
					// Prompt for query if not provided
					var err error
					query, err = line.Prompt("Enter lookup query: ")
					if err != nil || strings.TrimSpace(query) == "" {
						fmt.Println("Lookup cancelled")
						continue
					}
				} else {
					query = parts[1]
				}
				
				ctx := entity.ContextWithEntity(context.Background(), entityCtx)
				response, err := agentInstance.Process(ctx, agent.InputTypeRetrieve, query)
				if err != nil {
					fmt.Printf("Error looking up memories: %v\n", err)
				} else {
					fmt.Println(response)
				}

			case cmdQuery:
				question := ""
				if len(parts) == 1 {
					// Prompt for question if not provided
					var err error
					question, err = line.Prompt("Enter question: ")
					if err != nil || strings.TrimSpace(question) == "" {
						fmt.Println("Query cancelled")
						continue
					}
				} else {
					question = parts[1]
				}
				
				ctx := entity.ContextWithEntity(context.Background(), entityCtx)
				response, err := agentInstance.Process(ctx, agent.InputTypeQuery, question)
				if err != nil {
					fmt.Printf("Error querying: %v\n", err)
				} else {
					fmt.Println(response)
				}
				
			case cmdConfig:
				// Display current configuration
				fmt.Println("\nCurrent Configuration:")
				fmt.Println("======================")
				fmt.Printf("LTM Store Type: %s\n", cfg.LTM.Type)
				if cfg.LTM.Type == "sql" {
					fmt.Printf("SQL Driver: %s\n", cfg.LTM.SQL.Driver)
					fmt.Printf("SQL DSN: %s\n", cfg.LTM.SQL.DSN)
				} else if cfg.LTM.Type == "kv" {
					fmt.Printf("KV Provider: %s\n", cfg.LTM.KV.Provider)
				}
				fmt.Printf("Reasoning Provider: %s\n", cfg.Reasoning.Provider)
				fmt.Printf("Log Level: %s\n", cfg.Logging.Level)
				fmt.Printf("Entity: %s\n", currentEntity)
				fmt.Printf("User: %s\n", currentUser)

			default:
				fmt.Printf("Unknown command: %s\nType !help for available commands.\n", cmd)
			}
		} else {
			// Treat as a query by default
			ctx := entity.ContextWithEntity(context.Background(), entityCtx)
			response, err := agentInstance.Process(ctx, agent.InputTypeQuery, input)
			if err != nil {
				fmt.Printf("Error processing query: %v\n", err)
			} else {
				fmt.Println(response)
			}
		}
	}
}