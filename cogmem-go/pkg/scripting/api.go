package scripting

import (
	"fmt"
	"time"

	"github.com/lexlapax/cogmem/pkg/log"
	lua "github.com/yuin/gopher-lua"
)

// registerAPIFunctions registers Go functions that are available to Lua scripts.
func registerAPIFunctions(L *lua.LState) {
	// Create a cogmem table
	cogmem := L.NewTable()
	
	// Log function
	L.SetField(cogmem, "log", L.NewFunction(apiLog))
	
	// Current time function
	L.SetField(cogmem, "now", L.NewFunction(apiNow))
	
	// Format time function
	L.SetField(cogmem, "format_time", L.NewFunction(apiFormatTime))
	
	// UUID generation
	L.SetField(cogmem, "uuid", L.NewFunction(apiUUID))
	
	// JSON encoding/decoding
	L.SetField(cogmem, "json_encode", L.NewFunction(apiJSONEncode))
	L.SetField(cogmem, "json_decode", L.NewFunction(apiJSONDecode))
	
	// Register the cogmem table in the global namespace
	L.SetGlobal("cogmem", cogmem)
	
	// Register dostring function for testing purposes
	L.SetGlobal("dostring", L.NewFunction(apiDoString))
}

// apiLog is a function to log messages from Lua
func apiLog(L *lua.LState) int {
	level := L.CheckString(1)
	message := L.CheckString(2)
	
	switch level {
	case "debug":
		log.Debug("Lua script message", "message", message)
	case "info":
		log.Info("Lua script message", "message", message)
	case "warn", "warning":
		log.Warn("Lua script message", "message", message)
	case "error":
		log.Error("Lua script message", "message", message)
	default:
		log.Info("Lua script message", "message", message)
	}
	
	return 0
}

// apiNow returns the current time as a Unix timestamp
func apiNow(L *lua.LState) int {
	L.Push(lua.LNumber(time.Now().Unix()))
	return 1
}

// apiFormatTime formats a Unix timestamp as a string
func apiFormatTime(L *lua.LState) int {
	timestamp := L.CheckNumber(1)
	format := L.OptString(2, time.RFC3339)
	
	t := time.Unix(int64(timestamp), 0).UTC() // Use UTC to ensure consistent results
	L.Push(lua.LString(t.Format(format)))
	return 1
}

// apiUUID generates a UUID string
func apiUUID(L *lua.LState) int {
	// This is a placeholder implementation
	// In a real implementation, you'd use a proper UUID package
	// such as github.com/google/uuid
	
	// Use more randomness to ensure UUIDs are unique even when called in rapid succession
	uuid := fmt.Sprintf("uuid-%d-%d", time.Now().UnixNano(), time.Now().Add(time.Nanosecond).UnixNano())
	L.Push(lua.LString(uuid))
	return 1
}

// apiJSONEncode encodes a Lua table to a JSON string
func apiJSONEncode(L *lua.LState) int {
	value := L.CheckAny(1)
	
	// Convert Lua value to Go
	goValue := convertLuaToGo(value)
	
	// For now, just return a placeholder
	// In a real implementation, you'd use encoding/json
	L.Push(lua.LString(fmt.Sprintf("%v", goValue)))
	return 1
}

// apiJSONDecode decodes a JSON string to a Lua table
func apiJSONDecode(L *lua.LState) int {
	jsonStr := L.CheckString(1)
	
	// For now, just create an empty table
	// In a real implementation, you'd use encoding/json
	table := L.NewTable()
	table.RawSetString("original", lua.LString(jsonStr))
	
	L.Push(table)
	return 1
}

// apiDoString executes a Lua code string, primarily for testing purposes
func apiDoString(L *lua.LState) int {
	code := L.CheckString(1)
	
	log.Debug("Executing Lua code via dostring", "code_length", len(code))
	
	err := L.DoString(code)
	if err != nil {
		log.Error("Error in dostring execution", "error", err)
		L.Push(lua.LBool(false))  // Return false to indicate failure
		L.Push(lua.LString(err.Error()))
		return 2
	}
	
	L.Push(lua.LBool(true))  // Return true to indicate success
	return 1
}
