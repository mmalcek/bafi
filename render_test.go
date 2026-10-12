package main

import (
	"strings"
	"testing"
)

// TestRenderString covers what the online demo shows.
func TestRenderString(t *testing.T) {
	cases := []struct {
		name, input, tmpl, format string
		want                      string // substring
	}{
		{"ok", `{"a":"x"}`, `{{upper .a}}`, "json", "X"},
		{"input error", `{"a":}`, `{{.a}}`, "json", `bafi: cannot parse JSON input from the input box (line 1, column 6)`},
		{"template error", `{"a":1}`, `{{upper .a}}`, "json", "bafi: template error in the template (line 1, column 9)"},
		{"unknown format", `{}`, `x`, "jsn", "did you mean -f json?"},
		{"warning before output", `{}`, `[{{.a}}]`, "json", "bafi: warning: output contains <no value> 1 time(s)"},
		{"csv delimiter warning", "a;b\n1;2", `{{len .}}`, "csv", "did you mean -d ';'?"},
	}
	for _, tc := range cases {
		if got := renderString(tc.input, tc.tmpl, tc.format); !strings.Contains(got, tc.want) {
			t.Errorf("%s: %q does not contain %q", tc.name, got, tc.want)
		}
	}
	if got := renderString(`{}`, `[{{.a}}]`, "json"); !strings.HasSuffix(got, "\n\n[<no value>]") {
		t.Errorf("output must follow the warnings: %q", got)
	}
}

func TestLuaUnavailable(t *testing.T) {
	luaUnavailable = "not here"
	defer func() { luaUnavailable = "" }()
	if got := luaF("sum", 1, 2); got != "luaError: not here" {
		t.Errorf("got %q", got)
	}
}
