package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/log"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/mock"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

func main() {
	// Set up logging
	logConfig := log.Config{
		Level:  log.DebugLevel, // Use debug level to see all logs
		Format: log.TextFormat,
	}
	log.Setup(logConfig)
	
	log.Info("Starting example agent")
	
	// Create a context with an entity
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)
	
	// Initialize LTM store (using mock for simplicity)
	ltmStore := mock.NewMockStore()
	
	// Initialize the Lua scripting engine
	scriptsPath := "./scripts"
	scriptEngine, err := initScriptEngine(scriptsPath)
	if err != nil {
		log.Error("Failed to initialize script engine", "error", err)
		os.Exit(1)
	}
	defer scriptEngine.Close()
	
	// Initialize the MMU
	mmuInstance := mmu.NewMMU(
		ltmStore,
		scriptEngine,
		mmu.DefaultConfig(),
	)
	
	// Store some example memories
	storeExamples(ctx, mmuInstance)
	
	// Retrieve memories
	retrieveExamples(ctx, mmuInstance)
	
	log.Info("Example agent completed successfully")
}

func initScriptEngine(scriptsPath string) (scripting.Engine, error) {
	// Ensure scripts directory exists
	scriptsPath, err := filepath.Abs(scriptsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for scripts directory: %w", err)
	}
	
	// Create script engine
	scriptEngine, err := scripting.NewLuaEngine(scripting.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create Lua engine: %w", err)
	}
	
	// Load scripts if directory exists
	if _, err := os.Stat(scriptsPath); !os.IsNotExist(err) {
		if err := scriptEngine.LoadScriptDir(scriptsPath); err != nil {
			return nil, fmt.Errorf("failed to load scripts from %s: %w", scriptsPath, err)
		}
	} else {
		log.Warn("Scripts directory not found, continuing without scripts", "path", scriptsPath)
	}
	
	return scriptEngine, nil
}

func storeExamples(ctx context.Context, mmuInstance mmu.MMU) {
	// Store simple string memory
	log.Info("Storing example memory 1")
	id1, err := mmuInstance.EncodeToLTM(ctx, "This is a test memory")
	if err != nil {
		log.Error("Failed to store memory", "error", err)
		return
	}
	log.Info("Successfully stored memory", "id", id1)
	
	// Store structured memory
	log.Info("Storing example memory 2 with metadata")
	memory2 := map[string]interface{}{
		"content": "This is a structured memory with metadata",
		"metadata": map[string]interface{}{
			"type":       "example",
			"tags":       []string{"test", "example", "metadata"},
			"important":  true,
			"priority":   5,
		},
	}
	
	id2, err := mmuInstance.EncodeToLTM(ctx, memory2)
	if err != nil {
		log.Error("Failed to store structured memory", "error", err)
		return
	}
	log.Info("Successfully stored structured memory", "id", id2)
}

func retrieveExamples(ctx context.Context, mmuInstance mmu.MMU) {
	// Retrieve by text
	log.Info("Retrieving memories by text search")
	results, err := mmuInstance.RetrieveFromLTM(ctx, "test", mmu.DefaultRetrievalOptions())
	if err != nil {
		log.Error("Failed to retrieve memories", "error", err)
		return
	}
	
	log.Info("Search results", "count", len(results))
	for i, record := range results {
		log.Info(fmt.Sprintf("Result %d", i+1),
			"id", record.ID,
			"content", record.Content,
			"has_metadata", record.Metadata != nil,
		)
	}
	
	// Retrieve by exact match filter
	log.Info("Retrieving memories by metadata filter")
	query := map[string]interface{}{
		"filters": map[string]interface{}{
			"important": true,
		},
	}
	
	results, err = mmuInstance.RetrieveFromLTM(ctx, query, mmu.DefaultRetrievalOptions())
	if err != nil {
		log.Error("Failed to retrieve memories by filter", "error", err)
		return
	}
	
	log.Info("Filter results", "count", len(results))
	for i, record := range results {
		log.Info(fmt.Sprintf("Result %d", i+1),
			"id", record.ID,
			"content", record.Content,
		)
	}
}