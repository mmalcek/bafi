//go:build !(js && wasm)

package main

import (
	"bytes"
	"encoding/base64"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/clbanning/mxj/v2"
)

const bsonDump = `JgAAAAdfaWQAYQAVOXB0lMEH42tuAm5hbWUABgAAAEhlbGxvAAAmAAAAB19pZABhCmAZAkO5o4cVWbsCbmFtZQAGAAAAV29ybGQAAA==`

// runCLI runs the CLI in-process. stdin nil means "interactive terminal, nothing piped".
func runCLI(t *testing.T, stdin io.Reader, args ...string) (stdout, stderr string, code int) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, stdin, &out, &errOut)
	return out.String(), errOut.String(), code
}

// writeTempFile writes content to a file inside t.TempDir() and returns its path.
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return path
}

// TestVersionFormat guards version.txt: an empty or malformed file must not ship.
func TestVersionFormat(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`).MatchString(version) {
		t.Errorf("version.txt must contain N.N.N or N.N.N-suffix, got %q", version)
	}
}

func TestRun(t *testing.T) {
	description := `?Output template test: description = {{index .TOP_LEVEL "-description"}}`
	cases := []struct {
		name       string
		stdin      io.Reader
		args       []string
		wantCode   int
		wantStdout string // substring
		wantStderr string // substring
	}{
		{"no input and no pipe", nil, []string{"-t", "?x"}, 2, "", "bafi: no input"},
		{"missing template file", nil, []string{"-i", "testdata.xml", "-t", "hello.tmpl"}, 1, "", `cannot read template "hello.tmpl"`},
		{"template parse error", nil, []string{"-i", "testdata.xml", "-t", "?{{define content}}"}, 1, "", `unexpected "content" in define`},
		{"wrong -f for the data", nil, []string{"-i", "testdata.xml", "-f", "json", "-t", "?x"}, 1, "", "cannot parse JSON input"},
		{"xml by extension", nil, []string{"-i", "testdata.xml", "-t", description}, 0, "description = employeeData", ""},
		{"file list", nil, []string{"-i", "?filesTest.yaml", "-t", `?{{index .filesTest.TOP_LEVEL "-description"}}`}, 0, "employeeData", ""},
		{"file list missing", nil, []string{"-i", "?filesTest.yamlx", "-t", "?x"}, 1, "", `cannot read file list "filesTest.yamlx"`},
		{"version", nil, []string{"-v"}, 0, "Version: " + version, ""},
		{"no template", nil, []string{"-i", "testdata.xml"}, 2, "", "bafi: no template given"},
		{"help", nil, []string{"-h"}, 0, "Recipes and documentation:", ""},
		{"help long form", nil, []string{"--help"}, 0, "-strict", ""},
		{"unknown flag", nil, []string{"-x"}, 2, "", "flag provided but not defined: -x"},
		{"unquoted inline template", nil, []string{"-i", "testdata.xml", "-t", "?{{toJSON", ".}}"}, 2, "", `unexpected argument ".}}"`},
		{"stdin without -f", strings.NewReader(`{"a":1}`), []string{"-t", "?x"}, 2, "", "add -f json (the input looks like JSON)"},
		{"unknown -f", strings.NewReader(`{}`), []string{"-f", "jsn", "-t", "?x"}, 2, "", "did you mean -f json?"},
		{"stdin json", strings.NewReader(`{"a":"b"}`), []string{"-f", "json", "-t", "?{{.a}}"}, 0, "b", ""},
		{"output dir missing", nil, []string{"-i", "testdata.xml", "-t", "?x", "-o", filepath.Join(t.TempDir(), "no", "such", "out.txt")}, 1, "", "cannot create output file"},
		{"bad hex delimiter", strings.NewReader("a,b\n1,2"), []string{"-f", "csv", "-d", "0xZZ", "-t", "?x"}, 2, "", `invalid CSV delimiter "0xZZ"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, code := runCLI(t, tc.stdin, tc.args...)
			if code != tc.wantCode {
				t.Errorf("exit code %d, want %d (stderr: %s)", code, tc.wantCode, stderr)
			}
			if !strings.Contains(stdout, tc.wantStdout) {
				t.Errorf("stdout %q does not contain %q", stdout, tc.wantStdout)
			}
			if !strings.Contains(stderr, tc.wantStderr) {
				t.Errorf("stderr %q does not contain %q", stderr, tc.wantStderr)
			}
			if tc.wantStderr == "" && stderr != "" {
				t.Errorf("unexpected stderr: %s", stderr)
			}
			if strings.Contains(stderr, "\n20") && regexp.MustCompile(`\d{4}/\d\d/\d\d \d\d:`).MatchString(stderr) {
				t.Errorf("stderr must not carry log timestamps: %s", stderr)
			}
		})
	}
}

// TestRunRedirectedStdin: `bafi -f xml ... < file` used to fail with "noPipe"; any non-terminal stdin is input.
func TestRunRedirectedStdin(t *testing.T) {
	f, err := os.Open("testdata.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	stdout, stderr, code := runCLI(t, f, "-f", "xml", "-t", `?{{index .TOP_LEVEL "-description"}}`)
	if code != 0 || stdout != "employeeData" {
		t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestNoValueWarningAndStrict(t *testing.T) {
	stdout, stderr, code := runCLI(t, strings.NewReader(`{"name":"x"}`), "-f", "json", "-t", "?Hello {{.nmae}}")
	if code != 0 || stdout != "Hello <no value>" {
		t.Errorf("default mode must keep the output unchanged: code=%d stdout=%q", code, stdout)
	}
	if !strings.Contains(stderr, "bafi: warning: output contains <no value> 1 time(s)") || !strings.Contains(stderr, "-strict") {
		t.Errorf("missing <no value> warning: %q", stderr)
	}
	stdout, stderr, code = runCLI(t, strings.NewReader(`{"name":"x"}`), "-f", "json", "-strict", "-t", "?Hello {{.nmae}}")
	if code != 1 || stdout != "" {
		t.Errorf("-strict must fail without output: code=%d stdout=%q", code, stdout)
	}
	if !strings.Contains(stderr, `map has no entry for key "nmae"`) || !strings.Contains(stderr, "(line 1, column 9)") {
		t.Errorf("-strict error should name key and position: %s", stderr)
	}
	// The warning is also raised when writing to a file.
	out := filepath.Join(t.TempDir(), "out.txt")
	_, stderr, code = runCLI(t, strings.NewReader(`{}`), "-f", "json", "-o", out, "-t", "?{{.a}}{{.b}}")
	if code != 0 || !strings.Contains(stderr, "<no value> 2 time(s)") {
		t.Errorf("file output: code=%d stderr=%q", code, stderr)
	}
}

func TestCSVDelimiterWarning(t *testing.T) {
	_, stderr, code := runCLI(t, strings.NewReader("a;b\n1;2\n"), "-f", "csv", "-t", "?{{len .}}")
	if code != 0 || !strings.Contains(stderr, "did you mean -d ';'?") {
		t.Errorf("code=%d stderr=%q", code, stderr)
	}
	_, stderr, _ = runCLI(t, strings.NewReader("a;b\n1;2\n"), "-f", "csv", "-d", ";", "-t", "?{{len .}}")
	if stderr != "" {
		t.Errorf("no warning expected with the right delimiter: %q", stderr)
	}
}

// TestFileListValidation covers the "-i ?files.yaml" path: malformed records produce a descriptive
// error and "format" may be omitted when the extension is known.
func TestFileListValidation(t *testing.T) {
	testdata, err := filepath.Abs("testdata.xml")
	if err != nil {
		t.Fatal(err)
	}
	unknownExt := writeTempFile(t, "data.unknown", "<a/>")
	cases := []struct {
		name    string
		yaml    string
		wantErr string // substring of stderr, "" for success
	}{
		{"valid explicit format", "- file: " + testdata + "\n  format: xml\n  label: x\n", ""},
		{"format omitted, detected from extension", "- file: " + testdata + "\n  label: x\n", ""},
		{"format empty string, detected from extension", "- file: " + testdata + "\n  format: \"\"\n  label: x\n", ""},
		{"file is int", "- file: 123\n  format: xml\n  label: x\n", `fileList[0]: "file" must be a string (got int)`},
		{"label missing", "- file: " + testdata + "\n  format: xml\n", `fileList[0]: missing "label"`},
		{"format is list", "- file: " + testdata + "\n  format: [xml]\n  label: x\n", `fileList[0]: "format" must be a string`},
		{"second record broken", "- file: " + testdata + "\n  label: a\n- file: {}\n  label: b\n", `fileList[1]: "file" must be a string`},
		{"unknown extension without format", "- file: " + unknownExt + "\n  label: x\n", "cannot tell the format of"},
		{"misspelled format", "- file: " + testdata + "\n  format: xlm\n  label: x\n", "did you mean format: xml?"},
		{"empty label", "- file: " + testdata + "\n  label: \"\"\n", `fileList[0]: "label" must not be empty`},
		{"not a list", "file: x\n", "cannot parse file list"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := writeTempFile(t, "files.yaml", tc.yaml)
			_, stderr, code := runCLI(t, nil, "-i", "?"+list, "-t", `?{{index .x.TOP_LEVEL "-description"}}`)
			switch {
			case tc.wantErr == "" && code != 0:
				t.Errorf("unexpected error: %s", stderr)
			case tc.wantErr != "" && (code != 1 || !strings.Contains(stderr, tc.wantErr)):
				t.Errorf("want exit 1 and %q, got %d: %s", tc.wantErr, code, stderr)
			}
		})
	}
}

// TestFileListBOM: file-list entries are BOM-cleaned like single files (they were not before 1.5.0).
func TestFileListBOM(t *testing.T) {
	data := writeTempFile(t, "bom.json", "\xef\xbb\xbf{\"a\":\"ok\"}")
	list := writeTempFile(t, "files.yaml", "- file: "+data+"\n  label: x\n")
	stdout, stderr, code := runCLI(t, nil, "-i", "?"+list, "-t", "?{{.x.a}}")
	if code != 0 || stdout != "ok" {
		t.Errorf("code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestDetectFormat(t *testing.T) {
	cases := map[string]string{
		"a.json": "json", "a.JSON": "json", "a.bson": "bson", "a.yaml": "yaml", "a.yml": "yaml",
		"a.csv": "csv", "a.sta": "mt940", "a.xml": "xml", "a.cdf": "xml", "a.cdf3": "xml",
		"a.txt": "", "a": "", "": "",
	}
	for path, want := range cases {
		if got := detectFormat(path); got != want {
			t.Errorf("detectFormat(%q) = %q, want %q", path, got, want)
		}
	}
}

// TestOpenAIMode checks the mode switch and key resolution. Usage problems are caught before any
// network access; a dummy key must fail inside the OpenAI client, not in the template guard.
func TestOpenAIMode(t *testing.T) {
	t.Setenv(openAIKeyEnv, "")
	_, stderr, code := runCLI(t, nil, "-i", "testdata.xml", "-gq", "hello")
	if code != 2 || !strings.Contains(stderr, "OpenAI API key must be defined") {
		t.Errorf("-gq without any key: code=%d stderr=%s", code, stderr)
	}
	_, stderr, code = runCLI(t, nil, "-i", "testdata.xml", "-gk", "flag-key")
	if code != 2 || !strings.Contains(stderr, "OpenAI query is missing") {
		t.Errorf("-gk without -gq: code=%d stderr=%s", code, stderr)
	}

	t.Setenv(openAIKeyEnv, "dummy-key")
	_, stderr, code = runCLI(t, nil, "-i", "testdata.xml", "-gq", "hello")
	if code != 1 || !strings.Contains(stderr, "OpenAI request failed") {
		t.Errorf("env key not picked up: code=%d stderr=%s", code, stderr)
	}
	// Template mode is unaffected by an exported key.
	stdout, stderr, code := runCLI(t, nil, "-i", "testdata.xml", "-t", `?{{index .TOP_LEVEL "-description"}}`)
	if code != 0 || stdout != "employeeData" {
		t.Errorf("template mode with %s exported: code=%d stdout=%q stderr=%s", openAIKeyEnv, code, stdout, stderr)
	}
}

func TestPrepareDelimiter(t *testing.T) {
	for input, want := range map[string]rune{"": ',', ";": ';', "0x09": '\t', "0x2C": ',', "§x": '§'} {
		got, err := prepareDelimiter(input)
		if err != nil || got != want {
			t.Errorf("prepareDelimiter(%q) = %q, %v; want %q", input, got, err, want)
		}
	}
	if _, err := prepareDelimiter("0xZZ"); err == nil || !strings.Contains(err.Error(), "invalid delimiter") {
		t.Errorf("want invalid delimiter error, got %v", err)
	}
	_, err := parseCSV([]byte("a,b\n1,2"), "0xZZ", "stdin", nil)
	if err == nil || exitCode(err) != 2 || !strings.Contains(err.Error(), `invalid CSV delimiter "0xZZ"`) {
		t.Errorf("want usage error for the delimiter, got %v", err)
	}
}

func TestLoadInput(t *testing.T) {
	data, err := loadInput(options{inputFile: "testdata.xml"}, nil, nil)
	if err != nil || data.(mxj.Map)["TOP_LEVEL"] == nil {
		t.Errorf("testdata.xml: %v %v", data, err)
	}
	if _, err = loadInput(options{inputFile: "Hello.xml"}, nil, nil); err == nil || !strings.Contains(err.Error(), `cannot read input file "Hello.xml"`) {
		t.Errorf("missing file: %v", err)
	}
	if _, err = loadInput(options{}, nil, nil); err == nil || exitCode(err) != 2 {
		t.Errorf("no input: %v", err)
	}
	// Explicit -f wins over the extension.
	if _, err = loadInput(options{inputFile: "testdata.xml", format: "json"}, nil, nil); err == nil || !strings.Contains(err.Error(), "cannot parse JSON input") {
		t.Errorf("-f json on XML: %v", err)
	}
}

func TestMapInputData(t *testing.T) {
	parse := func(format, delimiter string, input []byte) (interface{}, error) {
		return mapInputData(input, inputSpec{Source: "stdin", Format: format, Delimiter: delimiter}, nil)
	}
	// JSON
	result, _ := parse("json", "", []byte(`{"name": "John","age": 30}`))
	if result.(map[string]interface{})["name"] != "John" || result.(map[string]interface{})["age"] != float64(30) {
		t.Errorf("resultJSON: %v", result)
	}
	result, _ = parse("JSON", "", []byte(` [{"name": "John","age": 30}, {"name": "Hanz","age": 28}]`))
	if result.([]map[string]interface{})[0]["name"] != "John" {
		t.Errorf("resultJSONarray: %v", result)
	}
	_, err := parse("json", "", []byte(`[{"name": "John","age": 30}, {"name": Hanz","age": 28}]`))
	if err == nil || !strings.Contains(err.Error(), "invalid character 'H'") {
		t.Errorf("resultJSONerr: %v", err)
	}
	_, err = parse("json", "", []byte(`{"name" John","age": 30}`))
	if err == nil || !strings.Contains(err.Error(), "invalid character 'J'") {
		t.Errorf("resultJSONerr: %v", err)
	}
	if result, err = parse("json", "", []byte(`null`)); err != nil || result.(map[string]interface{}) != nil {
		t.Errorf("JSON null keeps parsing to a nil map: %v %v", result, err)
	}
	// BSON
	result, _ = parse("bson", "", []byte{14, 0, 0, 0, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0})
	if result.(map[string]interface{})["h"] != "w" {
		t.Errorf("resultBSON: %v", result)
	}
	input, err := base64.StdEncoding.DecodeString(bsonDump)
	if err != nil {
		t.Fatal(err)
	}
	result, _ = parse("bson", "", input)
	if result.([]map[string]interface{})[0]["name"] != `Hello` {
		t.Errorf("resultBSONdump: %v", result)
	}
	_, err = parse("bson", "", []byte{14, 0, 0, 1, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0})
	if err == nil || !strings.Contains(err.Error(), "EOF") {
		t.Errorf("resultBSONerr: %v", err)
	}
	// YAML
	result, _ = parse("yaml", "", []byte(`name: John`))
	if result.(map[string]interface{})["name"] != "John" {
		t.Errorf("resultYAML: %v", result)
	}
	result, _ = parse("yaml", "", []byte("- name: John\r\n- name: Peter"))
	if result.([]map[string]interface{})[0]["name"] != "John" {
		t.Errorf("resultYAMLarray: %v", result)
	}
	_, err = parse("yaml", "", []byte("- name John\r\n- name: Peter"))
	if err == nil || !strings.Contains(err.Error(), "cannot unmarshal !!str `name John`") {
		t.Errorf("resultYAMLarrayErr: %v", err)
	}
	_, err = parse("yaml", "", []byte(`name John`))
	if err == nil || !strings.Contains(err.Error(), "cannot unmarshal !!str `name John`") {
		t.Errorf("resultYAMLerr: %v", err)
	}
	// CSV
	result, _ = parse("csv", ",", []byte("name,surname\r\nHello,World"))
	if result.([]map[string]interface{})[0]["name"] != "Hello" {
		t.Errorf("result: %v", result)
	}
	result, _ = parse("csv", "0x2C", []byte("name,surname\r\nHello,World"))
	if result.([]map[string]interface{})[0]["name"] != "Hello" {
		t.Errorf("result: %v", result)
	}
	_, err = parse("csv", "0x2C", []byte("name,surname\r\nHello,World,!!!"))
	if err == nil || !strings.Contains(err.Error(), "wrong number of fields") {
		t.Errorf("result: %v", err)
	}
	_, err = parse("csv", "", []byte(""))
	if err == nil || !strings.Contains(err.Error(), "CSV input from stdin is empty") {
		t.Errorf("result: %v", err)
	}
	// XML
	result, _ = parse("xml", "", []byte(`<name>John</name>`))
	if result.(mxj.Map)["name"] != "John" {
		t.Errorf("result: %v", result)
	}
	_, err = parse("xml", "", []byte(`<name>John<name>`))
	if err == nil || !strings.Contains(err.Error(), "XML syntax error on line 1") {
		t.Errorf("result: %v", err)
	}
	// Unknown / missing format
	if _, err = parse("jsn", "", []byte(`{}`)); err == nil || exitCode(err) != 2 {
		t.Errorf("unknown format: %v", err)
	}
	if _, err = parse("", "", []byte(`{}`)); err == nil || !strings.Contains(err.Error(), "cannot tell the input format of stdin") {
		t.Errorf("missing format: %v", err)
	}
}

func TestParseCSVDelimiters(t *testing.T) {
	tests := []struct {
		name      string
		delimiter string
		data      string
	}{
		{"default", "", "name,surname\nHello,World"},
		{"comma", ",", "name,surname\nHello,World"},
		{"semicolon", ";", "name;surname\nHello;World"},
		{"tab", "\t", "name\tsurname\nHello\tWorld"},
		{"hex tab", "0x09", "name\tsurname\nHello\tWorld"},
		{"hex comma", "0x2C", "name,surname\nHello,World"},
		{"section sign", "§", "name§surname\nHello§World"},
		{"fullwidth comma", "，", "name，surname\r\nHello，World"},
		{"supplementary rune", "🦊", "name🦊surname\nHello🦊World"},
		{"first Unicode rune", "§extra", "name§surname\nHello§World"},
		{"first ASCII rune", ";extra", "name;surname\nHello;World"},
	}
	want := []map[string]interface{}{{"name": "Hello", "surname": "World"}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCSV([]byte(tt.data), tt.delimiter, "stdin", nil)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("parseCSV() = %#v, want %#v", got, want)
			}
		})
	}

	t.Run("quoted delimiter and empty field", func(t *testing.T) {
		got, err := parseCSV([]byte("name§surname\n\"Hello§there\"§World\nHello§"), "§", "stdin", nil)
		if err != nil {
			t.Fatal(err)
		}
		want := []map[string]interface{}{
			{"name": "Hello§there", "surname": "World"},
			{"name": "Hello", "surname": ""},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("parseCSV() = %#v, want %#v", got, want)
		}
	})
}

func TestReadTemplate(t *testing.T) {
	result, err := readTemplate("?{{toXML .}}")
	if string(result) != "{{toXML .}}" || err != nil {
		t.Errorf("inline: %q %v", result, err)
	}
	result, err = readTemplate("template.tmpl")
	if !strings.Contains(string(result), "CSV formatted data:") || err != nil {
		t.Errorf("file: %v", err)
	}
	if _, err = readTemplate("hello.tmpl"); err == nil || !strings.Contains(err.Error(), `cannot read template "hello.tmpl"`) {
		t.Errorf("missing: %v", err)
	}
	if _, err = readTemplate("{{toJSON .}}"); err == nil || !strings.Contains(formatError(err), `inline templates must start with "?"`) {
		t.Errorf("forgotten ?: %v", err)
	}
	if _, err = readTemplate(""); err == nil || exitCode(err) != 2 {
		t.Errorf("empty: %v", err)
	}
}

func TestExecuteOutput(t *testing.T) {
	testData := map[string]interface{}{"Hello": "World"}
	var buf bytes.Buffer
	if _, err := render(&buf, testData, []byte(`{{define content}}`), "inline", false); err == nil || !strings.Contains(err.Error(), `unexpected "content"`) {
		t.Errorf("parse error: %v", err)
	}
	buf.Reset()
	if _, err := render(&buf, testData, []byte(`Output test: Hello {{.Hello}}`), "inline", false); err != nil || buf.String() != "Output test: Hello World" {
		t.Errorf("render: %q %v", buf.String(), err)
	}
	testData["Hello"] = make(chan int, 1)
	if _, err := render(&buf, testData, []byte(`{{.Hello}}`), "inline", false); err == nil || !strings.Contains(err.Error(), "can't print {{.Hello}} of type chan int") {
		t.Errorf("exec error: %v", err)
	}
	// -o writes the file; a failing template leaves no stdout output.
	out := filepath.Join(t.TempDir(), "out.txt")
	if _, stderr, code := runCLI(t, nil, "-i", "testdata.xml", "-o", out, "-t", `?{{index .TOP_LEVEL "-description"}}`); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr)
	}
	if got, _ := os.ReadFile(out); string(got) != "employeeData" {
		t.Errorf("output file: %q", got)
	}
	stdout, _, code := runCLI(t, nil, "-i", "testdata.xml", "-t", `?partial {{index .nope "x"}}`)
	if code != 1 || stdout != "" {
		t.Errorf("failing template must not print partial output: code=%d stdout=%q", code, stdout)
	}
	// A failing template must leave an existing -o file untouched: a parse error (the file used to be
	// truncated before parsing) and an execution error (the file used to be half-written).
	for _, tmpl := range []string{"?{{.a", `?partial {{index .nope "x"}}`} {
		if err := os.WriteFile(out, []byte("last good report"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, _, code := runCLI(t, nil, "-i", "testdata.xml", "-o", out, "-t", tmpl); code != 1 {
			t.Errorf("%s: want exit 1, got %d", tmpl, code)
		}
		if got, _ := os.ReadFile(out); string(got) != "last good report" {
			t.Errorf("%s: -o file changed to %q", tmpl, got)
		}
	}
}

// TestStdinDashArgument: a trailing "-" (read stdin) was ignored before 1.5.0 and keeps working.
func TestStdinDashArgument(t *testing.T) {
	stdout, stderr, code := runCLI(t, strings.NewReader(`{"a":"b"}`), "-f", "json", "-t", "?{{.a}}", "-")
	if code != 0 || stdout != "b" {
		t.Errorf("trailing -: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	_, stderr, code = runCLI(t, nil, "-i", "testdata.xml", "-t", "?x", "-")
	if code != 2 || !strings.Contains(stderr, "-i already names the input file") {
		t.Errorf("- together with -i: code=%d stderr=%q", code, stderr)
	}
}

func TestCleanBOM(t *testing.T) {
	if result := string(cleanBOM([]byte("\xef\xbb\xbf" + "Hello"))); result != "Hello" {
		t.Errorf("result: %v", result)
	}
	if result := string(cleanBOM([]byte("Hello"))); result != "Hello" {
		t.Errorf("result: %v", result)
	}
}
