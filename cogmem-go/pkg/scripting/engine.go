package scripting

import (
	"context"
	"io/ioutil"
	"path/filepath"
)

// Engine is the interface for the Lua scripting engine.
type Engine interface {
	// LoadScript loads a Lua script with the given name and content.
	LoadScript(name string, content []byte) error
	
	// LoadScriptFile loads a Lua script from a file path.
	LoadScriptFile(path string) error
	
	// LoadScriptDir loads all Lua scripts from a directory.
	LoadScriptDir(dir string) error
	
	// ExecuteFunction calls a Lua function with the given arguments.
	// The function should be previously loaded via LoadScript or LoadScriptFile.
	ExecuteFunction(ctx context.Context, funcName string, args ...interface{}) (interface{}, error)
	
	// Close releases resources associated with the engine.
	Close() error
}

// Config contains configuration options for the scripting engine.
type Config struct {
	// EnableSandboxing restricts access to potentially dangerous Lua modules like os and io
	EnableSandboxing bool
	
	// ScriptTimeoutMs sets a maximum execution time for scripts in milliseconds
	ScriptTimeoutMs int
	
	// MaxMemoryMB sets a maximum memory limit for the Lua state in megabytes
	MaxMemoryMB int
}

// DefaultConfig returns the default configuration for the scripting engine.
func DefaultConfig() Config {
	return Config{
		EnableSandboxing: true,
		ScriptTimeoutMs:  1000,  // 1 second
		MaxMemoryMB:      100,   // 100 MB
	}
}

// Helper function to load all Lua scripts from a directory
func LoadAllScripts(engine Engine, dir string) error {
	return engine.LoadScriptDir(dir)
}
