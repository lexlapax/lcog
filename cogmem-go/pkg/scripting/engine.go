package scripting

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lexlapax/cogmem/pkg/log"
	lua "github.com/yuin/gopher-lua"
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

// Errors
var (
	ErrScriptNotLoaded  = errors.New("script not loaded")
	ErrFunctionNotFound = errors.New("lua function not found")
	ErrExecutionTimeout = errors.New("script execution timed out")
	ErrInvalidArgument  = errors.New("invalid argument for lua function")
	ErrMemoryLimit      = errors.New("lua memory limit exceeded")
)

// LuaEngine implements the Engine interface using gopher-lua.
type LuaEngine struct {
	state       *lua.LState
	config      Config
	loadedFiles map[string]bool
	mutex       sync.Mutex
}

// NewLuaEngine creates a new LuaEngine with the given configuration.
func NewLuaEngine(config Config) (*LuaEngine, error) {
	// Create a new Lua state
	opts := lua.Options{
		SkipOpenLibs: config.EnableSandboxing,
	}
	
	L := lua.NewState(opts)
	
	// Initialize the engine
	engine := &LuaEngine{
		state:       L,
		config:      config,
		loadedFiles: make(map[string]bool),
	}
	
	// Setup the sandbox if enabled
	if config.EnableSandboxing {
		setupSandbox(L)
		log.Debug("Lua engine initialized with sandbox enabled", 
			"timeout_ms", config.ScriptTimeoutMs,
			"max_memory_mb", config.MaxMemoryMB)
	} else {
		L.OpenLibs()
		log.Debug("Lua engine initialized with sandbox disabled", 
			"timeout_ms", config.ScriptTimeoutMs,
			"max_memory_mb", config.MaxMemoryMB,
			"warning", "full Lua libraries available")
	}
	
	// Register API functions
	registerAPIFunctions(L)
	
	return engine, nil
}

// LoadScript loads a Lua script with the given name and content.
func (e *LuaEngine) LoadScript(name string, content []byte) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	log.Debug("Loading Lua script from string", "name", name, "size_bytes", len(content))
	
	if err := e.state.DoString(string(content)); err != nil {
		log.Error("Failed to load Lua script", "name", name, "error", err)
		return fmt.Errorf("failed to load script %s: %w", name, err)
	}
	
	e.loadedFiles[name] = true
	return nil
}

// LoadScriptFile loads a Lua script from a file path.
func (e *LuaEngine) LoadScriptFile(path string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	// Check if the file has already been loaded
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Error("Failed to get absolute path for script file", "path", path, "error", err)
		return fmt.Errorf("failed to get absolute path for %s: %w", path, err)
	}
	
	if e.loadedFiles[absPath] {
		log.Debug("Lua script file already loaded, skipping", "path", absPath)
		return nil // Already loaded
	}
	
	log.Debug("Loading Lua script from file", "path", absPath)
	
	// Load the file
	if err := e.state.DoFile(path); err != nil {
		log.Error("Failed to load Lua script file", "path", absPath, "error", err)
		return fmt.Errorf("failed to load script file %s: %w", path, err)
	}
	
	e.loadedFiles[absPath] = true
	return nil
}

// LoadScriptDir loads all Lua scripts from a directory.
func (e *LuaEngine) LoadScriptDir(dir string) error {
	log.Debug("Loading Lua scripts from directory", "dir", dir)
	
	fileCount := 0
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Error("Error walking script directory", "path", path, "error", err)
			return err
		}
		
		// Skip directories
		if d.IsDir() {
			return nil
		}
		
		// Only load .lua files
		if !strings.HasSuffix(path, ".lua") {
			return nil
		}
		
		fileCount++
		return e.LoadScriptFile(path)
	})
	
	if err != nil {
		log.Error("Failed to load Lua scripts from directory", "dir", dir, "error", err)
		return err
	}
	
	log.Debug("Successfully loaded Lua scripts from directory", "dir", dir, "file_count", fileCount)
	return nil
}

// ExecuteFunction calls a Lua function with the given arguments.
func (e *LuaEngine) ExecuteFunction(ctx context.Context, funcName string, args ...interface{}) (interface{}, error) {
	// Lock the mutex before accessing the Lua state
	e.mutex.Lock()
	
	log.DebugContext(ctx, "Executing Lua function", 
		"function", funcName, 
		"arg_count", len(args),
	)
	
	// Get the function from global environment
	fn := e.state.GetGlobal(funcName)
	if fn.Type() != lua.LTFunction {
		// Unlock before returning
		e.mutex.Unlock()
		log.WarnContext(ctx, "Lua function not found", "function", funcName)
		return nil, fmt.Errorf("%w: %s", ErrFunctionNotFound, funcName)
	}
	
	// Convert arguments to Lua values before starting the goroutine
	luaArgs, err := convertArgsToLua(e.state, args...)
	if err != nil {
		// Unlock before returning
		e.mutex.Unlock()
		log.ErrorContext(ctx, "Error converting arguments to Lua", 
			"function", funcName, 
			"error", err,
		)
		return nil, err
	}
	
	// Create a channel to handle timeout
	done := make(chan struct{})
	resultCh := make(chan lua.LValue, 1)
	errCh := make(chan error, 1)
	
	startTime := time.Now()
	
	// Push context to Lua state
	pushContext(e.state, ctx)
	
	// Create a new mutex that we'll use just for this execution
	// to avoid blocking other operations if this one times out
	execMutex := &sync.Mutex{}
	execMutex.Lock()
	
	// Execute the function in a goroutine
	go func() {
		defer close(done)
		defer e.mutex.Unlock() // Always unlock the main mutex
		defer execMutex.Unlock() // Signal that execution is complete
		
		// Call the function
		err := e.state.CallByParam(lua.P{
			Fn:      fn,
			NRet:    1,
			Protect: true,
		}, luaArgs...)
		
		if err != nil {
			log.ErrorContext(ctx, "Error executing Lua function", 
				"function", funcName, 
				"error", err,
			)
			errCh <- err
			return
		}
		
		// Get the result
		var result lua.LValue
		if e.state.GetTop() > 0 {
			result = e.state.Get(-1)
			e.state.Pop(1)
		}
		resultCh <- result
	}()
	
	// Wait for execution to complete or timeout
	select {
	case <-done:
		// Wait for execution mutex to be unlocked (should already be unlocked)
		execMutex.Lock()
		execMutex.Unlock()
		
		// Check for errors
		select {
		case err := <-errCh:
			return nil, err
		case result := <-resultCh:
			executionTime := time.Since(startTime)
			log.DebugContext(ctx, "Lua function executed successfully", 
				"function", funcName, 
				"execution_time_ms", executionTime.Milliseconds(),
			)
			return convertLuaToGo(result), nil
		default:
			return nil, errors.New("lua function execution completed without result or error")
		}
	case <-time.After(time.Duration(e.config.ScriptTimeoutMs) * time.Millisecond):
		// Force unlock the execution mutex to indicate timeout
		execMutex.Unlock()
		
		// For testing purposes, let's create a new goroutine to handle the cleanup
		// after the timeout. This avoids deadlocks when the main goroutine is blocked.
		// In production code, you'd want a more robust mechanism.
		go func() {
			// Wait a bit to let the execution goroutine reach a safe point
			time.Sleep(100 * time.Millisecond)
			
			// Try to cancel the operation or clean up state
			// This is a simplified approach - in production, consider 
			// using a separate Lua state for each function execution
			
			// The main mutex might still be locked - we'll try to unlock it
			// but ignore errors if it's already been unlocked
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error("Recovered from panic while unlocking mutex after timeout", "error", r)
					}
				}()
				e.mutex.Unlock()
			}()
		}()
		
		log.ErrorContext(ctx, "Lua function execution timed out", 
			"function", funcName, 
			"timeout_ms", e.config.ScriptTimeoutMs,
		)
		return nil, ErrExecutionTimeout
	case <-ctx.Done():
		// Similar approach for context cancellation
		execMutex.Unlock()
		
		go func() {
			time.Sleep(100 * time.Millisecond)
			func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error("Recovered from panic while unlocking mutex after context cancellation", "error", r)
					}
				}()
				e.mutex.Unlock()
			}()
		}()
		
		log.WarnContext(ctx, "Lua function execution canceled by context", 
			"function", funcName, 
			"error", ctx.Err(),
		)
		return nil, ctx.Err()
	}
}

// Close releases resources associated with the engine.
func (e *LuaEngine) Close() error {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.state.Close()
	return nil
}

// Helper function to convert Go values to Lua values
func convertArgsToLua(L *lua.LState, args ...interface{}) ([]lua.LValue, error) {
	lArgs := make([]lua.LValue, 0, len(args))
	
	for _, arg := range args {
		lv, err := convertToLua(L, arg)
		if err != nil {
			return nil, err
		}
		lArgs = append(lArgs, lv)
	}
	
	return lArgs, nil
}

// Helper function to convert a Go value to a Lua value
func convertToLua(L *lua.LState, val interface{}) (lua.LValue, error) {
	if val == nil {
		return lua.LNil, nil
	}
	
	switch v := val.(type) {
	case string:
		return lua.LString(v), nil
	case int:
		return lua.LNumber(v), nil
	case int64:
		return lua.LNumber(v), nil
	case float64:
		return lua.LNumber(v), nil
	case bool:
		return lua.LBool(v), nil
	case []interface{}:
		tbl := L.NewTable()
		for i, item := range v {
			lv, err := convertToLua(L, item)
			if err != nil {
				return nil, err
			}
			tbl.RawSetInt(i+1, lv)
		}
		return tbl, nil
	case map[string]interface{}:
		tbl := L.NewTable()
		for k, item := range v {
			lv, err := convertToLua(L, item)
			if err != nil {
				return nil, err
			}
			tbl.RawSetString(k, lv)
		}
		return tbl, nil
	default:
		return nil, fmt.Errorf("%w: unsupported type %T", ErrInvalidArgument, val)
	}
}

// Helper function to convert a Lua value to a Go value
func convertLuaToGo(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		// Check if it's an array-like table
		maxn := v.MaxN()
		if maxn > 0 {
			slice := make([]interface{}, 0, maxn)
			for i := 1; i <= maxn; i++ {
				item := v.RawGetInt(i)
				if item.Type() != lua.LTNil {
					slice = append(slice, convertLuaToGo(item))
				}
			}
			return slice
		}
		
		// It's a map-like table
		result := make(map[string]interface{})
		v.ForEach(func(key, value lua.LValue) {
			if k, ok := key.(lua.LString); ok {
				result[string(k)] = convertLuaToGo(value)
			}
		})
		return result
	default:
		return fmt.Sprintf("unsupported Lua type: %s", lv.Type().String())
	}
}

// Helper function to push context to Lua state
func pushContext(L *lua.LState, ctx context.Context) {
	// Create a context table
	ctxTable := L.NewTable()
	
	// You can add context values here if needed
	// For example, adding a deadline
	if deadline, ok := ctx.Deadline(); ok {
		L.SetField(ctxTable, "deadline", lua.LNumber(deadline.Unix()))
	}
	
	// Set the context in global environment
	L.SetGlobal("ctx", ctxTable)
}