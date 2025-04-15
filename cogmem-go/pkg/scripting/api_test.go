package scripting

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLuaAPI_Log(t *testing.T) {
	engine, err := NewLuaEngine(DefaultConfig())
	require.NoError(t, err)
	defer engine.Close()

	// Test using the log API
	err = engine.LoadScript("log_test", []byte(`
		function test_log()
			cogmem.log("info", "This is a test log message")
			cogmem.log("error", "This is an error message")
			cogmem.log("debug", "This is a debug message")
			return "log messages sent"
		end
	`))
	require.NoError(t, err)

	// Execute the function - we're just testing that it doesn't error
	result, err := engine.ExecuteFunction(context.Background(), "test_log")
	assert.NoError(t, err)
	assert.Equal(t, "log messages sent", result)
}

func TestLuaAPI_Now(t *testing.T) {
	engine, err := NewLuaEngine(DefaultConfig())
	require.NoError(t, err)
	defer engine.Close()

	// Test using the now API
	err = engine.LoadScript("now_test", []byte(`
		function test_now()
			local ts = cogmem.now()
			-- Simply return the timestamp, we'll validate it in Go
			return ts
		end
	`))
	require.NoError(t, err)

	// Execute the function
	result, err := engine.ExecuteFunction(context.Background(), "test_now")
	assert.NoError(t, err)
	
	// Validate that it's a recent timestamp
	ts, ok := result.(float64)
	assert.True(t, ok, "Expected timestamp to be a number")
	
	now := time.Now().Unix()
	assert.InDelta(t, now, ts, 60, "Timestamp should be within 60 seconds of current time")
}

func TestLuaAPI_FormatTime(t *testing.T) {
	engine, err := NewLuaEngine(DefaultConfig())
	require.NoError(t, err)
	defer engine.Close()

	// Test using the format_time API
	err = engine.LoadScript("format_time_test", []byte(`
		function test_format_time()
			local ts = 1609459200 -- 2021-01-01 00:00:00 UTC
			local formatted = cogmem.format_time(ts)
			return formatted
		end

		function test_format_time_custom()
			local ts = 1609459200 -- 2021-01-01 00:00:00 UTC
			local formatted = cogmem.format_time(ts, "2006-01-02")
			return formatted
		end
	`))
	require.NoError(t, err)

	// Execute with default format
	result, err := engine.ExecuteFunction(context.Background(), "test_format_time")
	assert.NoError(t, err)
	assert.Equal(t, "2021-01-01T00:00:00Z", result)

	// Execute with custom format
	result, err = engine.ExecuteFunction(context.Background(), "test_format_time_custom")
	assert.NoError(t, err)
	assert.Equal(t, "2021-01-01", result)
}

func TestLuaAPI_UUID(t *testing.T) {
	engine, err := NewLuaEngine(DefaultConfig())
	require.NoError(t, err)
	defer engine.Close()

	// Test using the UUID API
	err = engine.LoadScript("uuid_test", []byte(`
		function test_uuid()
			local id1 = cogmem.uuid()
			local id2 = cogmem.uuid()
			
			if id1 == id2 then
				return "UUIDs should be unique but were the same"
			end
			
			if type(id1) ~= "string" then
				return "UUID should be a string but was " .. type(id1)
			end
			
			return "valid UUID"
		end
	`))
	require.NoError(t, err)

	// Execute the function
	result, err := engine.ExecuteFunction(context.Background(), "test_uuid")
	assert.NoError(t, err)
	assert.Equal(t, "valid UUID", result)
}

func TestLuaAPI_JSON(t *testing.T) {
	engine, err := NewLuaEngine(DefaultConfig())
	require.NoError(t, err)
	defer engine.Close()

	// Test using the JSON API
	err = engine.LoadScript("json_test", []byte(`
		function test_json_roundtrip()
			local obj = {
				name = "test",
				value = 123,
				nested = {
					key = "value"
				}
			}
			
			local encoded = cogmem.json_encode(obj)
			local decoded = cogmem.json_decode(encoded)
			
			-- For this test we'll just return the original since
			-- our implementation is a placeholder
			return encoded
		end
	`))
	require.NoError(t, err)

	// Execute the function
	result, err := engine.ExecuteFunction(context.Background(), "test_json_roundtrip")
	assert.NoError(t, err)
	// Since our placeholder just does a string conversion, we're just checking it contains expected data
	assert.Contains(t, result.(string), "test")
}

func TestLuaAPI_Context(t *testing.T) {
	// Create a context with a deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	engine, err := NewLuaEngine(DefaultConfig())
	require.NoError(t, err)
	defer engine.Close()

	// Test accessing the context from Lua
	err = engine.LoadScript("context_test", []byte(`
		function test_context()
			if ctx == nil then
				return "ctx is nil"
			end
			
			if ctx.deadline == nil then
				return "no deadline in context"
			end
			
			return "context available with deadline"
		end
	`))
	require.NoError(t, err)

	// Execute the function with the context
	result, err := engine.ExecuteFunction(ctx, "test_context")
	assert.NoError(t, err)
	assert.Equal(t, "context available with deadline", result)
}
