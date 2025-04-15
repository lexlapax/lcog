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

// TestMMUWithLuaHooks tests the MMU integration with Lua hooks.
func TestMMUWithLuaHooks(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Create a temp directory for test scripts
	tempDir, err := os.MkdirTemp("", "cogmem-mmu-lua-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test Lua hooks
	hooksScript := `
	-- Called before encoding a record to LTM
	function before_encode(content)
		print("before_encode called with: " .. content)
		return content .. " [processed by before_encode]"
	end

	-- Called after a record is successfully encoded to LTM
	function after_encode(memory_id)
		print("after_encode called with ID: " .. memory_id)
		return true
	end

	-- Called before retrieving from LTM
	function before_retrieve(query)
		print("before_retrieve called")
		
		-- Add a prefix to the query text if it's a string
		if query and query.text then
			query.text = "modified:" .. query.text
		end
		
		return query
	end

	-- Called after retrieving from LTM
	function after_retrieve(records)
		print("after_retrieve called with " .. #records .. " records")
		
		-- Add a suffix to each record's content
		for i, record in ipairs(records) do
			record.content = record.content .. " [processed by after_retrieve]"
		end
		
		return records
	end

	-- Track number of times hooks are called
	before_encode_count = 0
	after_encode_count = 0
	before_retrieve_count = 0
	after_retrieve_count = 0

	-- Instrumented versions for testing
	function instrumented_before_encode(content)
		before_encode_count = before_encode_count + 1
		return before_encode(content)
	end

	function instrumented_after_encode(memory_id)
		after_encode_count = after_encode_count + 1
		return after_encode(memory_id)
	end

	function instrumented_before_retrieve(query)
		before_retrieve_count = before_retrieve_count + 1
		return before_retrieve(query)
	end

	function instrumented_after_retrieve(records)
		after_retrieve_count = after_retrieve_count + 1
		return after_retrieve(records)
	end

	-- Function to get hook call counts
	function get_hook_call_counts()
		return {
			before_encode = before_encode_count,
			after_encode = after_encode_count,
			before_retrieve = before_retrieve_count,
			after_retrieve = after_retrieve_count
		}
	end
	`

	// Save the hooks script
	hooksScriptPath := filepath.Join(tempDir, "mmu_hooks.lua")
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

	t.Run("Store with Before/After Encode Hooks", func(t *testing.T) {
		// Rename the hook functions to use the instrumented versions
		_, err := scriptEngine.ExecuteFunction(ctx, "pcall", "before_encode", "instrumented_before_encode")
		require.NoError(t, err)
		_, err = scriptEngine.ExecuteFunction(ctx, "pcall", "after_encode", "instrumented_after_encode")
		require.NoError(t, err)

		// Store a test memory
		memoryID, err := mmuInstance.EncodeToLTM(ctx, "Test content for encoding")
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Verify the hook modified the content
		retrievedMemory := ltmStore.GetRecord(memoryID)
		assert.Contains(t, retrievedMemory.Content, "[processed by before_encode]",
			"before_encode hook should have modified the content")

		// Check hook call counts
		result, err := scriptEngine.ExecuteFunction(ctx, "get_hook_call_counts")
		require.NoError(t, err)
		counts, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, float64(1), counts["before_encode"], "before_encode should have been called once")
		assert.Equal(t, float64(1), counts["after_encode"], "after_encode should have been called once")
	})

	t.Run("Retrieve with Before/After Retrieve Hooks", func(t *testing.T) {
		// Rename the hook functions to use the instrumented versions
		_, err := scriptEngine.ExecuteFunction(ctx, "pcall", "before_retrieve", "instrumented_before_retrieve")
		require.NoError(t, err)
		_, err = scriptEngine.ExecuteFunction(ctx, "pcall", "after_retrieve", "instrumented_after_retrieve")
		require.NoError(t, err)

		// First store some test data
		_, err = mmuInstance.EncodeToLTM(ctx, "Test content for retrieval")
		require.NoError(t, err)

		// Retrieve with a query
		options := mmu.DefaultRetrievalOptions()
		records, err := mmuInstance.RetrieveFromLTM(ctx, "retrieval", options)
		require.NoError(t, err)

		// Verify both hooks were applied
		for _, record := range records {
			// The query should have been modified by before_retrieve
			assert.Contains(t, record.Content, "modified:", 
				"Query should have been modified by before_retrieve hook")
			
			// The results should have been modified by after_retrieve
			assert.Contains(t, record.Content, "[processed by after_retrieve]",
				"Results should have been modified by after_retrieve hook")
		}

		// Check hook call counts
		result, err := scriptEngine.ExecuteFunction(ctx, "get_hook_call_counts")
		require.NoError(t, err)
		counts, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, float64(1), counts["before_retrieve"], "before_retrieve should have been called once")
		assert.Equal(t, float64(1), counts["after_retrieve"], "after_retrieve should have been called once")
	})

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

		// Reset hook counters
		_, err := scriptEngine.ExecuteFunction(ctx, "pcall", "before_encode_count = 0")
		require.NoError(t, err)
		_, err = scriptEngine.ExecuteFunction(ctx, "pcall", "after_encode_count = 0")
		require.NoError(t, err)

		// Store a memory
		memoryID, err := mmuNoHooks.EncodeToLTM(ctx, "Test content without hooks")
		require.NoError(t, err)

		// Verify hooks were not called
		retrievedMemory := ltmStore.GetRecord(memoryID)
		assert.NotContains(t, retrievedMemory.Content, "[processed by before_encode]",
			"before_encode hook should not have modified the content")

		// Check hook call counts - should still be 0
		result, err := scriptEngine.ExecuteFunction(ctx, "get_hook_call_counts")
		require.NoError(t, err)
		counts, ok := result.(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, float64(0), counts["before_encode"], "before_encode should not have been called")
		assert.Equal(t, float64(0), counts["after_encode"], "after_encode should not have been called")
	})

	t.Run("Missing Hook Functions", func(t *testing.T) {
		// Create a new scripting engine without loading the hook script
		emptyScriptEngine, err := scripting.NewLuaEngine(scripting.DefaultConfig())
		require.NoError(t, err)
		defer emptyScriptEngine.Close()

		// Create MMU with the empty scripting engine
		mmuEmptyHooks := mmu.NewMMU(
			ltmStore,
			nil,
			emptyScriptEngine,
			mmu.Config{
				EnableLuaHooks: true,
			},
		)

		// Store should still work even if hooks are missing
		memoryID, err := mmuEmptyHooks.EncodeToLTM(ctx, "Test with missing hooks")
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Retrieve should still work as well - we're ignoring results
		options := mmu.DefaultRetrievalOptions()
		_, err = mmuEmptyHooks.RetrieveFromLTM(ctx, "missing hooks", options)
		require.NoError(t, err)
		// Just checking that it doesn't crash, not the actual results
	})

	t.Run("Error Handling in Hooks", func(t *testing.T) {
		// Create a script with errors
		errorScript := `
		function before_encode(content)
			error("Intentional error in before_encode")
			return content
		end

		function after_encode(memory_id)
			-- This hook works fine
			return true
		end

		function before_retrieve(query)
			-- Force a nil dereference error
			return query.non_existent_field.value
		end

		function after_retrieve(records)
			-- This hook works fine
			return records
		end
		`

		errorScriptPath := filepath.Join(tempDir, "error_hooks.lua")
		err = os.WriteFile(errorScriptPath, []byte(errorScript), 0644)
		require.NoError(t, err)

		// Create a new scripting engine with the error-prone hooks
		errorScriptEngine, err := scripting.NewLuaEngine(scripting.DefaultConfig())
		require.NoError(t, err)
		defer errorScriptEngine.Close()

		err = errorScriptEngine.LoadScriptFile(errorScriptPath)
		require.NoError(t, err)

		// Create a MMU with the error-prone scripting engine
		mmuErrorHooks := mmu.NewMMU(
			ltmStore,
			nil,
			errorScriptEngine,
			mmu.Config{
				EnableLuaHooks: true,
			},
		)

		// Store should still work even if before_encode hook errors
		memoryID, err := mmuErrorHooks.EncodeToLTM(ctx, "Test with error in hooks")
		require.NoError(t, err)
		require.NotEmpty(t, memoryID)

		// Retrieve should still work even if before_retrieve hook errors
		options := mmu.DefaultRetrievalOptions()
		_, err = mmuErrorHooks.RetrieveFromLTM(ctx, "error hooks", options)
		require.NoError(t, err)
		// Just checking that it doesn't crash
	})
}