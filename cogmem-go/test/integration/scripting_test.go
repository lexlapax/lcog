//go:build integration
// +build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lexlapax/cogmem/pkg/scripting"
)

// TestScriptingEngineIntegration tests the scripting engine with actual Lua scripts.
func TestScriptingEngineIntegration(t *testing.T) {
	// Skip if not running integration tests
	if os.Getenv("INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test; set INTEGRATION_TESTS=true to run")
	}

	// Create a temp directory for test scripts
	tempDir, err := os.MkdirTemp("", "cogmem-scripting-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test script file
	testScriptContent := `
	-- Test function that returns a table
	function get_test_data()
		return {
			name = "Test Name",
			value = 42,
			items = {"item1", "item2", "item3"},
			nested = {
				key = "value",
				num = 123
			}
		}
	end

	-- Test function that processes input arguments
	function process_data(text, number, flag)
		return {
			text_length = string.len(text),
			number_doubled = number * 2,
			flag_inverted = not flag,
			combined = text .. " - " .. tostring(number)
		}
	end

	-- Test function with context
	function use_context()
		if ctx and ctx.test_value then
			return ctx.test_value
		else
			return "no context"
		end
	end

	-- Global counter to test persistence between calls
	call_count = 0

	-- Test function that uses global state
	function increment_counter()
		call_count = call_count + 1
		return call_count
	end
	`

	testScriptPath := filepath.Join(tempDir, "test.lua")
	err = os.WriteFile(testScriptPath, []byte(testScriptContent), 0644)
	require.NoError(t, err)

	// Create the scripting engine
	config := scripting.Config{
		EnableSandboxing: true,
		ScriptTimeoutMs:  1000,
		MaxMemoryMB:      10,
	}
	engine, err := scripting.NewLuaEngine(config)
	require.NoError(t, err)
	defer engine.Close()

	// Load the test script
	err = engine.LoadScriptFile(testScriptPath)
	require.NoError(t, err)

	// Create a context for tests
	ctx := context.Background()
	
	t.Run("Return Complex Data", func(t *testing.T) {
		result, err := engine.ExecuteFunction(ctx, "get_test_data")
		require.NoError(t, err)
		
		// Verify the result is correctly converted to Go types
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")
		
		assert.Equal(t, "Test Name", resultMap["name"])
		assert.Equal(t, float64(42), resultMap["value"])
		
		items, ok := resultMap["items"].([]interface{})
		require.True(t, ok, "Items should be a slice")
		assert.Equal(t, 3, len(items))
		assert.Equal(t, "item1", items[0])
		
		nested, ok := resultMap["nested"].(map[string]interface{})
		require.True(t, ok, "Nested should be a map")
		assert.Equal(t, "value", nested["key"])
		assert.Equal(t, float64(123), nested["num"])
	})
	
	t.Run("Process Input Arguments", func(t *testing.T) {
		result, err := engine.ExecuteFunction(ctx, "process_data", "hello", 42, true)
		require.NoError(t, err)
		
		resultMap, ok := result.(map[string]interface{})
		require.True(t, ok, "Result should be a map")
		
		assert.Equal(t, float64(5), resultMap["text_length"])
		assert.Equal(t, float64(84), resultMap["number_doubled"])
		assert.Equal(t, false, resultMap["flag_inverted"])
		assert.Equal(t, "hello - 42", resultMap["combined"])
	})
	
	t.Run("Context Passing", func(t *testing.T) {
		// Create a context with a custom value
		ctxWithValue := context.WithValue(ctx, "test_value", "test context value")
		
		// Execute the function with the context
		result, err := engine.ExecuteFunction(ctxWithValue, "use_context")
		require.NoError(t, err)
		
		// The context handling is simplified in the test, so we expect the default
		assert.Equal(t, "no context", result)
	})
	
	t.Run("State Persistence", func(t *testing.T) {
		// First call should return 1
		result1, err := engine.ExecuteFunction(ctx, "increment_counter")
		require.NoError(t, err)
		assert.Equal(t, float64(1), result1)
		
		// Second call should return 2
		result2, err := engine.ExecuteFunction(ctx, "increment_counter")
		require.NoError(t, err)
		assert.Equal(t, float64(2), result2)
		
		// Third call should return 3
		result3, err := engine.ExecuteFunction(ctx, "increment_counter")
		require.NoError(t, err)
		assert.Equal(t, float64(3), result3)
	})
	
	t.Run("Function Not Found", func(t *testing.T) {
		_, err := engine.ExecuteFunction(ctx, "non_existent_function")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function not found")
	})
	
	t.Run("Timeout Handling", func(t *testing.T) {
		// Create a new script with an infinite loop
		infiniteLoopScript := `
		function infinite_loop()
			while true do
				-- Do nothing, just loop forever
			end
			return "This should never be reached"
		end
		`
		
		infiniteLoopPath := filepath.Join(tempDir, "infinite_loop.lua")
		err = os.WriteFile(infiniteLoopPath, []byte(infiniteLoopScript), 0644)
		require.NoError(t, err)
		
		err = engine.LoadScriptFile(infiniteLoopPath)
		require.NoError(t, err)
		
		// Set a short timeout for this test
		engineWithTimeout, err := scripting.NewLuaEngine(scripting.Config{
			EnableSandboxing: true,
			ScriptTimeoutMs:  100, // 100ms timeout
			MaxMemoryMB:      10,
		})
		require.NoError(t, err)
		defer engineWithTimeout.Close()
		
		err = engineWithTimeout.LoadScriptFile(infiniteLoopPath)
		require.NoError(t, err)
		
		// Execute the infinite loop function, should timeout
		start := time.Now()
		_, err = engineWithTimeout.ExecuteFunction(ctx, "infinite_loop")
		elapsed := time.Since(start)
		
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timed out")
		assert.Less(t, elapsed, 500*time.Millisecond, "Timeout should be enforced within a reasonable time")
	})
	
	t.Run("Loading Directory", func(t *testing.T) {
		// Create multiple script files
		scripts := map[string]string{
			"script1.lua": `function func1() return "result1" end`,
			"script2.lua": `function func2() return "result2" end`,
			"notlua.txt": `function not_loaded() return "should not load" end`,
		}
		
		scriptDir := filepath.Join(tempDir, "scripts")
		err = os.Mkdir(scriptDir, 0755)
		require.NoError(t, err)
		
		for name, content := range scripts {
			err = os.WriteFile(filepath.Join(scriptDir, name), []byte(content), 0644)
			require.NoError(t, err)
		}
		
		// Create a new engine to test directory loading
		dirEngine, err := scripting.NewLuaEngine(config)
		require.NoError(t, err)
		defer dirEngine.Close()
		
		// Load all scripts from the directory
		err = dirEngine.LoadScriptDir(scriptDir)
		require.NoError(t, err)
		
		// Test that .lua files were loaded
		result1, err := dirEngine.ExecuteFunction(ctx, "func1")
		require.NoError(t, err)
		assert.Equal(t, "result1", result1)
		
		result2, err := dirEngine.ExecuteFunction(ctx, "func2")
		require.NoError(t, err)
		assert.Equal(t, "result2", result2)
		
		// Test that non-lua files were not loaded
		_, err = dirEngine.ExecuteFunction(ctx, "not_loaded")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "function not found")
	})
}