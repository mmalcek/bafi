package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

// writeLuaFile writes content into a temp directory and returns the file path.
func writeLuaFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "functions.lua")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp lua file: %v", err)
	}
	return path
}

func TestLoadLuaStateErrors(t *testing.T) {
	if L, err := loadLuaState(filepath.Join(t.TempDir(), "missing.lua")); err == nil || L != nil {
		t.Errorf("expected error for missing file, got state=%v err=%v", L, err)
	}
	if L, err := loadLuaState(writeLuaFile(t, "function broken( return 1 end")); err == nil || L != nil {
		t.Errorf("expected error for syntax error, got state=%v err=%v", L, err)
	}
}

func TestLuaSandboxRemovesFunctions(t *testing.T) {
	L, err := loadLuaState(writeLuaFile(t, "function noop(d) return d end"))
	if err != nil {
		t.Fatalf("loadLuaState: %v", err)
	}
	defer L.Close()
	for lib, names := range luaRestrictedFunctions {
		tbl, ok := L.GetGlobal(lib).(*lua.LTable)
		if !ok {
			t.Fatalf("library %q is not a table", lib)
		}
		for _, name := range names {
			if v := tbl.RawGetString(name); v != lua.LNil {
				t.Errorf("%s.%s should be nil, got %s", lib, name, v.Type())
			}
		}
	}
	// package.loaded shares the same table as _G, so require() must not resurrect anything.
	if err := L.DoString(`assert(require("os").execute == nil and require("io").popen == nil)`); err != nil {
		t.Errorf("restricted functions reachable via require: %v", err)
	}
}

func TestLuaSandboxKeepsAllowedFunctions(t *testing.T) {
	L, err := loadLuaState(writeLuaFile(t, "function noop(d) return d end"))
	if err != nil {
		t.Fatalf("loadLuaState: %v", err)
	}
	defer L.Close()
	allowed := []string{
		"os.date", "os.time", "os.clock", "os.getenv",
		"io.open", "io.read", "io.write", "io.lines",
		"string.format", "table.concat", "math.floor",
	}
	for _, name := range allowed {
		if err := L.DoString("assert(type(" + name + ") == 'function', '" + name + " missing')"); err != nil {
			t.Errorf("%s should still be available: %v", name, err)
		}
	}
	for _, name := range []string{"require", "load", "loadstring", "dofile", "loadfile", "pcall"} {
		if L.GetGlobal(name).Type() != lua.LTFunction {
			t.Errorf("global %s should still be a function, got %s", name, L.GetGlobal(name).Type())
		}
	}
	if err := L.DoString(`assert(tonumber(os.date("%Y")) > 2000)`); err != nil {
		t.Errorf("os.date should work: %v", err)
	}
}

func TestLuaSandboxBlocksTopLevelCode(t *testing.T) {
	L, err := loadLuaState(writeLuaFile(t, `os.execute("true")`))
	if err == nil {
		L.Close()
		t.Fatal("expected top-level os.execute to fail")
	}
	// gopher-lua wording for calling a nil global
	if !strings.Contains(err.Error(), "attempt to call a non-function object") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestLuaFRejectsRestrictedCall(t *testing.T) {
	L, err := luaState()
	if err != nil {
		t.Fatalf("luaState: %v", err)
	}
	if err := L.DoString(`function pwn(d) local f = io.popen("id") return f:read("*a") end`); err != nil {
		t.Fatalf("define pwn: %v", err)
	}
	got := luaF("pwn", "x")
	if !strings.Contains(got, "luaError:") || !strings.Contains(got, "attempt to call a non-function object") {
		t.Errorf("expected luaError about calling a removed function, got %q", got)
	}
	if strings.Contains(got, "uid=") {
		t.Errorf("io.popen executed a command: %q", got)
	}
}

func TestLuaFStackBalanced(t *testing.T) {
	L, err := luaState()
	if err != nil {
		t.Fatalf("luaState: %v", err)
	}
	if err := L.DoString(`function retnum(d) return 42 end`); err != nil {
		t.Fatalf("define retnum: %v", err)
	}
	var got string
	for n := 0; n < 200; n++ {
		got = luaF("retnum", "x")
	}
	if !strings.Contains(got, "luaError: getResult") {
		t.Errorf("expected getResult error, got %q", got)
	}
	if top := L.GetTop(); top != 0 {
		t.Errorf("Lua stack leaked: top=%d after repeated calls", top)
	}
	for n := 0; n < 200; n++ {
		luaF("sum", "1", "2")
	}
	if top := L.GetTop(); top != 0 {
		t.Errorf("Lua stack leaked on string results: top=%d", top)
	}
}

func TestLuaFBadFunctionName(t *testing.T) {
	for _, got := range []string{luaF(), luaF(5, "x")} {
		if !strings.HasPrefix(got, "luaError: function name must be a string") {
			t.Errorf("expected function-name error, got %q", got)
		}
	}
}
