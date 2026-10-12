package main

import (
	"fmt"
	"os"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

// luaFunctionsPath is where user-defined Lua functions are looked up, relative to the
// current working directory. The file is loaded lazily on the first {{lua ...}} call,
// never at startup, so `bafi -v` or a template without Lua never executes it.
const luaFunctionsPath = "./lua/functions.lua"

var (
	luaData     *lua.LState
	luaLoadOnce sync.Once
	luaLoadErr  error
)

// luaRestrictedFunctions lists Lua standard-library functions that are removed from the
// state before functions.lua runs: anything that spawns processes, terminates bafi, or
// deletes/renames files. Everything else (os.date, os.time, io.open, require, ...) stays.
// The lib tables are shared between _G and package.loaded, so niling fields on them also
// covers `require("os").execute`.
var luaRestrictedFunctions = map[string][]string{
	"os": {"execute", "exit", "remove", "rename", "tmpname"},
	"io": {"popen"},
}

// luaState returns the shared Lua state, loading ./lua/functions.lua on first use.
// A missing file is reported as an error wrapping fs.ErrNotExist; any other failure
// (unreadable file, syntax error, runtime error in top-level code) is wrapped with the path.
func luaState() (*lua.LState, error) {
	luaLoadOnce.Do(func() {
		if _, err := os.Stat(luaFunctionsPath); err != nil {
			luaLoadErr = err
			return
		}
		luaData, luaLoadErr = loadLuaState(luaFunctionsPath)
	})
	return luaData, luaLoadErr
}

// loadLuaState creates a restricted Lua state and executes the given file in it.
// On failure the state is closed and nil is returned together with the error.
func loadLuaState(path string) (*lua.LState, error) {
	L := lua.NewState()
	restrictLuaState(L)
	if err := L.DoFile(path); err != nil {
		L.Close()
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	return L, nil
}

// restrictLuaState removes every function listed in luaRestrictedFunctions from the state.
func restrictLuaState(L *lua.LState) {
	for lib, names := range luaRestrictedFunctions {
		tbl, ok := L.GetGlobal(lib).(*lua.LTable)
		if !ok {
			continue
		}
		for _, name := range names {
			tbl.RawSetString(name, lua.LNil)
		}
	}
}
