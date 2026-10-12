//go:build js && wasm

package main

// The online demo on the documentation home page. Built with `task wasm` into docs/js/bafi.wasm; the
// page calls bafiRender(input, template, format) and shows the returned text. (Not plain "bafi": the
// page's <h1 id="bafi"> already owns that global name.)

import (
	"fmt"
	"syscall/js"
)

func main() {
	luaUnavailable = "Lua functions are not available in the online demo; run bafi on your computer to use them"
	js.Global().Set("bafiRender", js.FuncOf(bafiJS))
	js.Global().Set("bafiVersion", js.ValueOf(version))
	select {} // keep the Go runtime alive for later calls
}

// bafiJS renders synchronously (pure CPU, nothing to wait for). A panic must not kill the instance,
// otherwise every later click would fail.
func bafiJS(_ js.Value, args []js.Value) (result any) {
	defer func() {
		if r := recover(); r != nil {
			result = fmt.Sprintf("bafi: internal error: %v", r)
		}
	}()
	if len(args) != 3 {
		return "bafi: the demo must call bafiRender(input, template, format)"
	}
	return renderString(args[0].String(), args[1].String(), args[2].String())
}
