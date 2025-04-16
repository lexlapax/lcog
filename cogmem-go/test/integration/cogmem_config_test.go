//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/lexlapax/cogmem/pkg/cogmem"
	"github.com/lexlapax/cogmem/pkg/config"
	"github.com/lexlapax/cogmem/pkg/entity"
)

// TestNewCogMemFromConfig tests the simplified initialization from config file.
func TestNewCogMemFromConfig(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Create a temp directory for test resources
	tempDir, err := os.MkdirTemp("", "cogmem-config-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a temp directory for scripts
	scriptsDir := filepath.Join(tempDir, "scripts")
	err = os.Mkdir(scriptsDir, 0755)
	require.NoError(t, err)

	// Create reflection script
	reflectionScript := `
	-- Track reflection calls
	reflection_count = 0

	-- Primary reflection function
	function reflect(history_json)
		-- Increment the counter
		reflection_count = reflection_count + 1
		
		-- Return a simple result
		return {
			summary = "Reflection performed on operations",
			timestamp = os.time()
		}
	end

	-- Function to get reflection count
	function get_reflection_count()
		return reflection_count
	end
	
	-- Hooks needed for reflection module
	function before_reflection_analysis(memories)
		return false  -- Don't skip analysis
	end
	
	function after_insight_generation(insights)
		return nil
	end
	
	function before_consolidation(insights)
		return insights
	end
	
	-- Hooks needed for MMU
	function before_encode(content)
		return content
	end
	
	function after_encode(memory_id)
	end
	
	function before_retrieve(query)
		return query
	end
	
	function after_retrieve(results)
		return results
	end
	`

	// Create retrieval filter script
	retrievalFilterScript := `
	function filter_memories(memories, query)
		-- Simple pass-through filter
		return memories
	end
	`

	// Save the scripts
	err = os.WriteFile(filepath.Join(scriptsDir, "reflection.lua"), []byte(reflectionScript), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(scriptsDir, "retrieval_filter.lua"), []byte(retrievalFilterScript), 0644)
	require.NoError(t, err)

	// Create a test config file
	testConfig := config.Config{
		LTM: config.LTMConfig{
			Type: "mock", // Use mock for testing
		},
		Scripting: config.ScriptingConfig{
			Paths: []string{scriptsDir},
		},
		Reasoning: config.ReasoningConfig{
			Provider: "mock",
		},
		Reflection: config.ReflectionConfig{
			Enabled:              true,
			TriggerFrequency:     3,
			MaxMemoriesToAnalyze: 10,
		},
		Logging: config.LoggingConfig{
			Level: "debug",
		},
	}

	// Save the config to a YAML file
	configYaml, err := yaml.Marshal(testConfig)
	require.NoError(t, err)
	configPath := filepath.Join(tempDir, "test_config.yaml")
	err = os.WriteFile(configPath, configYaml, 0644)
	require.NoError(t, err)

	// Initialize client from config
	client, err := cogmem.NewCogMemFromConfig(configPath)
	require.NoError(t, err)
	require.NotNil(t, client, "Client should be initialized")

	// Test basic functionality
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)

	// Store a memory
	response, err := client.Process(ctx, cogmem.InputTypeStore, "Test memory from config initialization")
	require.NoError(t, err)
	assert.Contains(t, response, "Memory stored successfully")

	// Retrieve the memory
	response, err = client.Process(ctx, cogmem.InputTypeRetrieve, "Test memory")
	require.NoError(t, err)
	assert.Contains(t, response, "Test memory from config")

	// Test with entity isolation
	entityCtx2 := entity.NewContext("other-entity", "other-user")
	ctx2 := entity.ContextWithEntity(context.Background(), entityCtx2)

	// Store memory for the second entity
	response, err = client.Process(ctx2, cogmem.InputTypeStore, "Entity 2's private data")
	require.NoError(t, err)
	assert.Contains(t, response, "Memory stored successfully")

	// First entity should not see second entity's memory
	response, err = client.Process(ctx, cogmem.InputTypeRetrieve, "private")
	require.NoError(t, err)
	assert.Contains(t, response, "No memories found", "Entity isolation should prevent cross-entity retrieval")

	// Test various storage locations in the config
	t.Run("ConfigFilePaths", func(t *testing.T) {
		// Test with nonexistent config
		_, err := cogmem.NewCogMemFromConfig("/path/does/not/exist.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to load configuration")

		// Test with minimal valid config
		mockConfigPath := filepath.Join(tempDir, "mock_config.yaml")
		mockConfig := config.Config{
			LTM: config.LTMConfig{
				Type: "mock",
			},
			Reasoning: config.ReasoningConfig{
				Provider: "mock", // Must specify a provider
			},
			Scripting: config.ScriptingConfig{
				Paths: []string{scriptsDir}, // Need script path for reflection hooks
			},
		}
		mockConfigYaml, err := yaml.Marshal(mockConfig)
		require.NoError(t, err)
		err = os.WriteFile(mockConfigPath, mockConfigYaml, 0644)
		require.NoError(t, err)

		mockClient, err := cogmem.NewCogMemFromConfig(mockConfigPath)
		assert.NoError(t, err, "Error creating client with minimal config: %v", err)
		assert.NotNil(t, mockClient, "Client should not be nil with minimal config")
	})
}