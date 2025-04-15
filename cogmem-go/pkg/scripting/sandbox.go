package scripting

import (
	"log"

	lua "github.com/yuin/gopher-lua"
)

// setupSandbox configures a restricted sandbox environment for Lua scripts.
// It selectively opens only safe libraries and removes dangerous functions.
func setupSandbox(L *lua.LState) {
	// Selectively open only safe libraries
	
	// Open the basic library
	L.OpenLibs()
	removeUnsafeFunctions(L)
	
	// String library is safe
	L.Push(lua.LString("string"))
	lua.OpenString(L)
	L.SetGlobal("string", L.Get(-1))
	L.Pop(1)
	
	// Table library is safe
	L.Push(lua.LString("table"))
	lua.OpenTable(L)
	L.SetGlobal("table", L.Get(-1))
	L.Pop(1)
	
	// Math library is safe
	L.Push(lua.LString("math"))
	lua.OpenMath(L)
	L.SetGlobal("math", L.Get(-1))
	L.Pop(1)
	
	// Explicitly make unsafe modules nil or empty
	L.SetGlobal("io", lua.LNil)
	L.SetGlobal("os", lua.LNil)
	L.SetGlobal("package", lua.LNil)
	L.SetGlobal("require", lua.LNil)
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	
	// Set up print to log to our logger instead
	L.SetGlobal("print", L.NewFunction(safePrint))
}

// removeUnsafeFunctions removes potentially dangerous functions from the base library
func removeUnsafeFunctions(L *lua.LState) {
	// Get the _G table
	g := L.Get(-1)
	if t, ok := g.(*lua.LTable); ok {
		// Remove unsafe functions
		t.RawSetString("dofile", lua.LNil)
		t.RawSetString("loadfile", lua.LNil)
		t.RawSetString("load", lua.LNil)
		t.RawSetString("os", lua.LNil)
		t.RawSetString("io", lua.LNil)
		t.RawSetString("require", lua.LNil)
		t.RawSetString("package", lua.LNil)
	}
}

// safePrint redirects Lua's print to our logger
func safePrint(L *lua.LState) int {
	top := L.GetTop()
	args := make([]interface{}, top)
	
	for i := 1; i <= top; i++ {
		args[i-1] = convertLuaToGo(L.Get(i))
	}
	
	log.Println("[LUA]", args)
	return 0
}