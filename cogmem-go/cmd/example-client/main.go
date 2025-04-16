package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/peterh/liner"

	"github.com/lexlapax/cogmem/pkg/cogmem"
	"github.com/lexlapax/cogmem/pkg/config"
	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
)

// Constants for the command-line interface
const (
	cmdHelp     = "!help"
	cmdQuit     = "!quit"
	cmdEntity   = "!entity"
	cmdUser     = "!user"
	cmdRemember = "!remember"
	cmdLookup   = "!lookup"
	cmdSearch   = "!search"  // New semantic search command
	cmdQuery    = "!query"
	cmdReflect  = "!reflect"
	cmdConfig   = "!config"
)

// Command-line help text
const helpText = `
CogMem Client - Command Reference:
-----------------------------------------
!help                 - Show this help message
!entity <id>          - Set the current entity ID
!user <id>            - Set the current user ID
!remember <text>      - Store a memory in LTM
!lookup <query>       - Retrieve memories matching query by keyword
!search <query>       - Retrieve memories using semantic (vector) search
!query <question>     - Ask a question using context from memories
!reflect              - Trigger a reflection cycle manually
!config               - Show current configuration
!quit                 - Exit the application

Notes:
- Regular text input is treated as a query
- Tab completion is available for commands
- Use up/down arrows for command history
- Reflection occurs automatically based on configured frequency
- Semantic search requires vector LTM and a reasoning engine (OpenAI)`

// historyFile is the file where command history is stored
const historyFile = ".cogmem_history"

func main() {
	// Parse command-line flags
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()
	
	// Initialize logger
	log.Setup(log.Config{
		Level:  log.InfoLevel,
		Format: log.TextFormat,
	})

	log.Info("Starting CogMem client")

	// Simplified initialization with single function call
	clientInstance, err := cogmem.NewCogMemFromConfig(*configPath)
	if err != nil {
		log.Error("Failed to initialize CogMem client", "error", err)
		os.Exit(1)
	}

	// Load config for CLI display purposes only
	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		log.Error("Failed to load configuration for CLI", "error", err)
		os.Exit(1)
	}
	
	// Start the command-line interface
	runCLI(clientInstance, cfg)
}

// runCLI starts the command-line interface for user interaction
func runCLI(clientInstance *cogmem.CogMemClientImpl, cfg *config.Config) {
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
		commands := []string{cmdHelp, cmdQuit, cmdEntity, cmdUser, cmdRemember, cmdLookup, cmdSearch, cmdQuery, cmdReflect, cmdConfig}
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
	fmt.Println("\n=== CogMem Client ===")
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
				response, err := clientInstance.Process(ctx, cogmem.InputTypeStore, memory)
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
				response, err := clientInstance.Process(ctx, cogmem.InputTypeRetrieve, query)
				if err != nil {
					fmt.Printf("Error looking up memories: %v\n", err)
				} else {
					fmt.Println(response)
				}
				
			case cmdSearch:
				query := ""
				if len(parts) == 1 {
					// Prompt for query if not provided
					var err error
					query, err = line.Prompt("Enter semantic search query: ")
					if err != nil || strings.TrimSpace(query) == "" {
						fmt.Println("Search cancelled")
						continue
					}
				} else {
					query = parts[1]
				}
				
				// Create a special query for semantic search
				ctx := entity.ContextWithEntity(context.Background(), entityCtx)
				fmt.Println("Performing semantic search...")
				
				// Use InputTypeQuery with a semantic search prefix to trigger vector search
				response, err := clientInstance.Process(ctx, cogmem.InputTypeQuery, "SEMANTIC_SEARCH: "+query)
				if err != nil {
					fmt.Printf("Error in semantic search: %v\n", err)
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
				response, err := clientInstance.Process(ctx, cogmem.InputTypeQuery, question)
				if err != nil {
					fmt.Printf("Error querying: %v\n", err)
				} else {
					fmt.Println(response)
				}

			case cmdReflect:
				// Manually trigger reflection
				fmt.Println("Manually triggering reflection cycle...")
				ctx := entity.ContextWithEntity(context.Background(), entityCtx)
				
				// We need to perform a dummy operation first to put something in working memory
				_, err := clientInstance.Process(ctx, cogmem.InputTypeStore, "Manual reflection trigger at "+time.Now().Format(time.RFC3339))
				if err != nil {
					fmt.Printf("Error preparing for reflection: %v\n", err)
					continue
				}
				
				// Now trigger reflection directly (normally happens automatically based on operation count)
				response, err := clientInstance.Process(ctx, cogmem.InputTypeQuery, "Perform reflection on recent memories and operations")
				if err != nil {
					fmt.Printf("Error during reflection: %v\n", err)
				} else {
					fmt.Println("Reflection completed successfully")
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
				} else if cfg.LTM.Type == "chromemgo" || cfg.LTM.Type == "vector" {
					fmt.Printf("ChromemGo URL: %s\n", cfg.LTM.ChromemGo.URL)
					fmt.Printf("ChromemGo Collection: %s\n", cfg.LTM.ChromemGo.Collection)
				}
						if cfg.LTM.KV.Provider == "postgres_hstore" {
							fmt.Printf("PostgreSQL HStore (using table: memory_records_hstore)\n")
						}
				
				fmt.Printf("\nReasoning Provider: %s\n", cfg.Reasoning.Provider)
				if cfg.Reasoning.Provider == "openai" {
					fmt.Printf("OpenAI Model: %s\n", cfg.Reasoning.OpenAI.Model)
					fmt.Printf("OpenAI Embedding Model: %s\n", cfg.Reasoning.OpenAI.EmbeddingModel)
				}
				
				fmt.Printf("\nReflection Enabled: %v\n", cfg.Reflection.Enabled)
				fmt.Printf("Reflection Frequency: %d\n", cfg.Reflection.TriggerFrequency)
				
				fmt.Printf("\nLog Level: %s\n", cfg.Logging.Level)
				fmt.Printf("Entity: %s\n", currentEntity)
				fmt.Printf("User: %s\n", currentUser)

			default:
				fmt.Printf("Unknown command: %s\nType !help for available commands.\n", cmd)
			}
		} else {
			// Treat as a query by default
			ctx := entity.ContextWithEntity(context.Background(), entityCtx)
			response, err := clientInstance.Process(ctx, cogmem.InputTypeQuery, input)
			if err != nil {
				fmt.Printf("Error processing query: %v\n", err)
			} else {
				fmt.Println(response)
			}
		}
	}
}