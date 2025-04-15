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

	"github.com/lexlapax/cogmem/pkg/entity"
	"github.com/lexlapax/cogmem/pkg/mem/ltm/adapters/mock"
	"github.com/lexlapax/cogmem/pkg/mmu"
	"github.com/lexlapax/cogmem/pkg/scripting"
)

// TestMMUWithLuaHooks tests a simplified version of MMU with Lua hooks.
func TestSimplifiedMMUWithLuaHooks(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Create a temp directory for test scripts
	tempDir, err := os.MkdirTemp("", "cogmem-mmu-lua-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test Lua hooks with minimal functionality
	hooksScript := `
	-- Test for storing data
	hook_called_counter = 0
	
	function before_encode(content)
		hook_called_counter = hook_called_counter + 1
		return content
	end
	
	function after_encode(memory_id)
		hook_called_counter = hook_called_counter + 1
		return true
	end
	
	function before_retrieve(query)
		hook_called_counter = hook_called_counter + 1
		return query
	end
	
	function after_retrieve(results)
		hook_called_counter = hook_called_counter + 1
		return results
	end
	
	function get_call_count()
		return hook_called_counter
	end
	`

	// Save the hooks script
	hooksScriptPath := filepath.Join(tempDir, "simple_hooks.lua")
	err = os.WriteFile(hooksScriptPath, []byte(hooksScript), 0644)
	require.NoError(t, err)

	// Create a mock LTM store
	ltmStore := mock.NewMockStore()

	// Create a scripting engine and load the hooks
	scriptEngine, err := scripting.NewLuaEngine(scripting.DefaultConfig())
	require.NoError(t, err)
	defer scriptEngine.Close()

	err = scriptEngine.LoadScriptFile(hooksScriptPath)
	require.NoError(t, err)

	// Create the MMU with hooks enabled
	mmuInstance := mmu.NewMMU(
		ltmStore,
		nil, // No reasoning engine needed for this test
		scriptEngine,
		mmu.Config{
			EnableLuaHooks: true,
		},
	)

	// Create a context with entity
	entityCtx := entity.NewContext("test-entity", "test-user")
	ctx := entity.ContextWithEntity(context.Background(), entityCtx)

	// Test if hooks get called during operations
	t.Run("Basic Hook Counting", func(t *testing.T) {
		// Get initial hook count
		result, err := scriptEngine.ExecuteFunction(ctx, "get_call_count")
		require.NoError(t, err)
		initialCount, ok := result.(float64)
		require.True(t, ok)
		
		// Store a test memory
		memoryID, err := mmuInstance.EncodeToLTM(ctx, "Test content for hooks")
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Retrieve something to trigger retrieve hooks
		options := mmu.DefaultRetrievalOptions()
		_, err = mmuInstance.RetrieveFromLTM(ctx, "Test", options)
		require.NoError(t, err)

		// Check if hooks were called (should have 4 more calls)
		result, err = scriptEngine.ExecuteFunction(ctx, "get_call_count")
		require.NoError(t, err)
		newCount, ok := result.(float64)
		require.True(t, ok)
		
		assert.Greater(t, newCount, initialCount, "Hook call count should have increased")
	})

	// Test with hooks disabled
	t.Run("Hooks Disabled", func(t *testing.T) {
		// Create a new MMU instance with hooks disabled
		mmuNoHooks := mmu.NewMMU(
			ltmStore,
			nil,
			scriptEngine,
			mmu.Config{
				EnableLuaHooks: false,
			},
		)

		// Get initial hook count
		result, err := scriptEngine.ExecuteFunction(ctx, "get_call_count")
		require.NoError(t, err)
		initialCount, ok := result.(float64)
		require.True(t, ok)

		// Store a memory
		memoryID, err := mmuNoHooks.EncodeToLTM(ctx, "Test content without hooks")
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Check hook count - should not have changed
		result, err = scriptEngine.ExecuteFunction(ctx, "get_call_count")
		require.NoError(t, err)
		newCount, ok := result.(float64)
		require.True(t, ok)
		
		assert.Equal(t, initialCount, newCount, "Hook count should not change when hooks are disabled")
	})
}