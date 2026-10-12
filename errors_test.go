package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// TestFriendlyErrors pins location, hint and troubleshooting anchor for every error class users hit.
// Parser messages change between Go versions (json/v2 in Go 1.25+), so each hint is asserted here.
func TestFriendlyErrors(t *testing.T) {
	cases := []struct {
		name       string
		format     string // input format; "" = template case
		input      string
		tmpl       string
		strict     bool
		wantLine   int
		wantCol    int
		wantHint   string
		wantAnchor string
	}{
		{name: "json unquoted value", format: "json", input: "{\n  \"a\": 1,\n  \"b\": oops\n}\n", wantLine: 3, wantCol: 8, wantHint: `text values must be in double quotes, e.g. "oops"`, wantAnchor: "invalid-json"},
		{name: "json trailing comma", format: "json", input: "{\n  \"a\": 1,\n}\n", wantLine: 3, wantCol: 1, wantHint: "remove the trailing comma before }", wantAnchor: "invalid-json"},
		{name: "json unquoted key", format: "json", input: `{a: 1}`, wantLine: 1, wantCol: 2, wantHint: `object keys must be in double quotes: "a"`, wantAnchor: "invalid-json"},
		{name: "json single quotes", format: "json", input: `{'a': 1}`, wantLine: 1, wantCol: 2, wantHint: "not single quotes", wantAnchor: "invalid-json"},
		{name: "json truncated", format: "json", input: `{"a": [1,2`, wantLine: 1, wantCol: 10, wantHint: "ends too early", wantAnchor: "invalid-json"},
		{name: "json empty", format: "json", input: " \n", wantHint: "did the previous command in the pipe fail?", wantAnchor: "input-is-empty"},
		{name: "json scalar array", format: "json", input: `[1,2]`, wantHint: "array of objects", wantAnchor: "invalid-json"},
		{name: "json multibyte column", format: "json", input: `{"ž": "č", x}`, wantLine: 1, wantCol: 12, wantHint: "object keys must be in double quotes", wantAnchor: "invalid-json"},
		{name: "yaml indentation", format: "yaml", input: "a: 1\n b: 2\n", wantLine: 2, wantHint: "check the indentation", wantAnchor: "invalid-yaml"},
		{name: "yaml tabs", format: "yaml", input: "a:\n\tb: 1\n", wantLine: 2, wantHint: "TAB", wantAnchor: "invalid-yaml"},
		{name: "yaml scalar", format: "yaml", input: "just text", wantLine: 1, wantHint: "YAML mapping", wantAnchor: "invalid-yaml"},
		{name: "xml mismatched tag", format: "xml", input: "<a>\n  <b>1</c>\n</a>", wantLine: 2, wantCol: 7, wantHint: "closing tag does not match", wantAnchor: "invalid-xml"},
		{name: "xml bare ampersand", format: "xml", input: "<a>\n  <b>Tom & Jerry</b>\n</a>", wantLine: 2, wantCol: 10, wantHint: "&amp;", wantAnchor: "invalid-xml"},
		{name: "xml empty", format: "xml", input: "", wantHint: "nothing arrived on stdin", wantAnchor: "input-is-empty"},
		{name: "csv field count", format: "csv", input: "a,b\n1,2\n3,4,5\n", wantLine: 3, wantHint: "as many fields as the header", wantAnchor: "csv-wrong-number-of-fields"},
		{name: "csv bare quote", format: "csv", input: "a,b\n1,x\"y\n", wantLine: 2, wantCol: 4, wantHint: "double the inner quotes", wantAnchor: "invalid-csv"},
		{name: "csv empty", format: "csv", input: "", wantHint: "nothing arrived on stdin", wantAnchor: "input-is-empty"},
		{name: "bson garbage", format: "bson", input: "not bson at all", wantHint: "mongodump", wantAnchor: "invalid-input"},
		{name: "bson decoder panic", format: "bson", input: "{\"\u017e\":\x00}", wantHint: "mongodump", wantAnchor: "invalid-input"},
		{name: "xml given json", format: "xml", input: `{"a": 1}`, wantHint: "use -f json", wantAnchor: "invalid-xml"},
		{name: "xml given text", format: "xml", input: "just text", wantHint: "root element", wantAnchor: "invalid-xml"},
		{name: "xml multibyte column", format: "xml", input: "<a>\u017e\u017e\u017e\u017e\u017e\u017e\u017e\u017e\u017e\u017e<1b/></a>", wantLine: 1, wantCol: 15, wantHint: "every element needs", wantAnchor: "invalid-xml"},
		{name: "json multibyte bad value", format: "json", input: `{"a": ž}`, wantLine: 1, wantCol: 7, wantHint: `"ž"`, wantAnchor: "invalid-json"},
		{name: "tmpl unclosed action", tmpl: "line one\n{{.a", wantLine: 2, wantHint: "missing its closing }}", wantAnchor: "template-syntax-error"},
		{name: "tmpl unknown function", tmpl: "{{uper .a}}", wantLine: 1, wantHint: `did you mean "upper"?`, wantAnchor: "unknown-function"},
		{name: "tmpl unknown function no match", tmpl: "{{frobnicate .a}}", wantLine: 1, wantHint: "see the function reference", wantAnchor: "unknown-function"},
		{name: "tmpl dash in key", tmpl: "{{.my-key}}", wantLine: 1, wantHint: `{{index . "my-key"}}`, wantAnchor: "keys-with-dashes"},
		{name: "tmpl missing end", tmpl: "Users:\n{{range .users}}\n- {{.name}}\n", wantLine: 2, wantHint: "missing its {{end}}", wantAnchor: "template-syntax-error"},
		{name: "tmpl stray end", tmpl: "a\n{{end}}", wantLine: 2, wantHint: "has no matching", wantAnchor: "template-syntax-error"},
		{name: "tmpl wrong type", tmpl: "Value: {{upper .n}}", wantLine: 1, wantCol: 16, wantHint: "(toString .value)", wantAnchor: "wrong-type"},
		{name: "tmpl list into text function", tmpl: "{{upper .l}}", wantLine: 1, wantCol: 9, wantHint: "the value is a list", wantAnchor: "wrong-type"},
		{name: "tmpl object into text function", tmpl: "{{upper .m}}", wantLine: 1, wantCol: 9, wantHint: "the value is an object", wantAnchor: "wrong-type"},
		{name: "tmpl field on number", tmpl: "xyz {{ .n.b }}", wantLine: 1, wantCol: 8, wantHint: "is not an object", wantAnchor: "wrong-type"},
		{name: "tmpl index nil", tmpl: `{{index .nope "x"}}`, wantLine: 1, wantCol: 3, wantHint: "{{toJSON .}}", wantAnchor: "missing-keys-and-no-value"},
		{name: "tmpl arg count", tmpl: "{{upper .s .s}}", wantLine: 1, wantCol: 3, wantHint: "upper takes 1 argument(s), 2 were given", wantAnchor: "unknown-function"},
		{name: "tmpl range over number", tmpl: "{{range .n}}x{{end}}", wantLine: 1, wantCol: 9, wantHint: "mustArray", wantAnchor: "wrong-type"},
		{name: "tmpl compare float64 with int", tmpl: "{{if gt .n 2}}x{{end}}", wantLine: 1, wantCol: 6, wantHint: "decimal point (2.0) or convert the value: (toInt .count)", wantAnchor: "wrong-type"},
		{name: "tmpl compare text with number", tmpl: "{{eq .s 3}}", wantLine: 1, wantCol: 3, wantHint: "one side is text", wantAnchor: "wrong-type"},
		{name: "tmpl compare object", tmpl: "{{lt .m 1}}", wantLine: 1, wantCol: 3, wantHint: "compare single values", wantAnchor: "wrong-type"},
		{name: "tmpl function error", tmpl: `{{replaceAllRegex "[a" "b" "c"}}`, wantLine: 1, wantCol: 3, wantHint: "replaceAllRegex rejected its arguments", wantAnchor: "unknown-function"},
		{name: "tmpl strict missing key", tmpl: "Hello {{.nmae}}", strict: true, wantLine: 1, wantCol: 9, wantHint: `{{with index . "nmae"}}`, wantAnchor: "missing-keys-and-no-value"},
		{name: "tmpl strict nested missing key", tmpl: "{{.m.nope}}", strict: true, wantLine: 1, wantCol: 3, wantHint: `no key "nope"`, wantAnchor: "missing-keys-and-no-value"},
	}
	data := map[string]any{"n": float64(1), "s": "text", "m": map[string]any{"k": "v"}, "l": []any{"a"}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.format != "" {
				_, err = mapInputData([]byte(tc.input), inputSpec{Source: "stdin", Format: tc.format}, nil)
			} else {
				_, err = render(&bytes.Buffer{}, data, []byte(tc.tmpl), "inline", tc.strict)
			}
			var be *bafiError
			if !errors.As(err, &be) {
				t.Fatalf("want *bafiError, got %T: %v", err, err)
			}
			if be.Line != tc.wantLine || be.Col != tc.wantCol {
				t.Errorf("location = line %d col %d, want line %d col %d\n%s", be.Line, be.Col, tc.wantLine, tc.wantCol, formatError(err))
			}
			if !strings.Contains(be.Hint, tc.wantHint) {
				t.Errorf("hint %q does not contain %q", be.Hint, tc.wantHint)
			}
			if be.Anchor != tc.wantAnchor {
				t.Errorf("anchor %q, want %q", be.Anchor, tc.wantAnchor)
			}
			if exitCode(err) != 1 {
				t.Errorf("exit code %d, want 1", exitCode(err))
			}
			if strings.Contains(err.Error(), "template: inline:") || strings.Contains(err.Error(), "xml.Decoder.Token()") {
				t.Errorf("nested prefix left in Error(): %s", err)
			}
		})
	}
}

// TestFormatErrorExact pins the complete rendering of three representative errors.
func TestFormatErrorExact(t *testing.T) {
	_, jsonErr := parseJSON([]byte("{\n  \"a\": 1,\n  \"b\": oops\n}\n"), "stdin")
	_, tmplErr := render(&bytes.Buffer{}, map[string]any{"a": float64(1)}, []byte("Value: {{upper .a}}"), "report.tmpl", false)
	usage := usageError("no template given", `pass a template file with -t report.tmpl`, "")
	cases := map[string]struct {
		err  error
		want string
	}{
		"json": {jsonErr, `bafi: cannot parse JSON input from stdin (line 3, column 8)

  3 |   "b": oops
    |        ^
  invalid character 'o' looking for beginning of value

hint: text values must be in double quotes, e.g. "oops" (only true, false and null are bare words)
docs: https://mmalcek.github.io/bafi/troubleshooting/#invalid-json
`},
		"template": {tmplErr, `bafi: template error in report.tmpl (line 1, column 16)

  1 | Value: {{upper .a}}
    |                ^
  wrong type for value; expected string; got float64 (at <.a>)

hint: the value is a float64, the function needs text; convert it first, e.g. (toString .value)
docs: https://mmalcek.github.io/bafi/troubleshooting/#wrong-type
`},
		"usage": {usage, `bafi: no template given

hint: pass a template file with -t report.tmpl

usage: bafi -i INPUT -t TEMPLATE [-o OUTPUT]   (bafi -h for help and examples)
`},
	}
	for name, tc := range cases {
		if got := formatError(tc.err); got != tc.want {
			t.Errorf("%s:\n--- got ---\n%s--- want ---\n%s", name, got, tc.want)
		}
	}
	if got := formatError(errors.New("plain")); got != "bafi: plain\n" {
		t.Errorf("plain error: %q", got)
	}
	if exitCode(usage) != 2 || exitCode(jsonErr) != 1 || exitCode(errors.New("x")) != 1 {
		t.Error("exit codes: usage must be 2, everything else 1")
	}
	if want := "cannot parse JSON input from stdin (line 3, column 8): invalid character 'o' looking for beginning of value"; jsonErr.Error() != want {
		t.Errorf("Error() = %q, want %q", jsonErr.Error(), want)
	}
}

func TestClipLine(t *testing.T) {
	short := "short line"
	if got, col := clipLine(short, 3); got != short || col != 3 {
		t.Errorf("short line changed: %q %d", got, col)
	}
	long := strings.Repeat("a", 200) + "X" + strings.Repeat("b", 200)
	got, col := clipLine(long, 201)
	if !strings.HasPrefix(got, "…") || !strings.HasSuffix(got, "…") {
		t.Errorf("long line not clipped on both sides: %q", got)
	}
	if r := []rune(got); r[col-1] != 'X' {
		t.Errorf("caret column %d does not point at X in %q", col, got)
	}
	multi := strings.Repeat("ž", 150) + "X"
	got, col = clipLine(multi, 151)
	if r := []rune(got); r[col-1] != 'X' {
		t.Errorf("multibyte clipping broke the column: %d in %q", col, got)
	}
}

func TestCaretPadding(t *testing.T) {
	if got := caretPadding("\tab", 3); got != "\t " {
		t.Errorf("tabs must be copied: %q", got)
	}
	if got := caretPadding("žč", 3); got != "  " {
		t.Errorf("runes, not bytes: %q", got)
	}
}

func TestOffsetToLineCol(t *testing.T) {
	data := []byte("ab\nčd\nef")
	for _, tc := range []struct{ off, line, col int }{{0, 1, 1}, {3, 2, 1}, {5, 2, 2}, {7, 3, 1}} {
		if l, c := offsetToLineCol(data, tc.off); l != tc.line || c != tc.col {
			t.Errorf("offset %d: got %d:%d want %d:%d", tc.off, l, c, tc.line, tc.col)
		}
	}
}

func TestNoValueCounter(t *testing.T) {
	var out bytes.Buffer
	c := &noValueCounter{w: &out}
	for _, chunk := range []string{"a <no ", "value> b <no value><no", " value>", "<no value"} {
		c.Write([]byte(chunk))
	}
	if c.n != 3 {
		t.Errorf("counted %d, want 3", c.n)
	}
	if out.String() != "a <no value> b <no value><no value><no value" {
		t.Errorf("output altered: %q", out.String())
	}
}

func TestDidYouMean(t *testing.T) {
	cases := map[string]string{"jsn": "json", "xlm": "xml", "ymal": "yaml", "uper": "upper", "toJson": "toJSON", "zzzzzz": "", "json": ""}
	for word, want := range cases {
		candidates := append(append([]string{}, inputFormats...), "upper", "toJSON")
		if got := didYouMean(word, candidates); got != want {
			t.Errorf("didYouMean(%q) = %q, want %q", word, got, want)
		}
	}
}

func TestUnclosedBlock(t *testing.T) {
	src := []byte("{{if .a}}\n{{range .b}}{{end}}\n{{- with .c}}\nx\n{{end}}\n")
	if line, kw := unclosedBlock(src); line != 1 || kw != "if" {
		t.Errorf("got line %d %q, want 1 if", line, kw)
	}
}
