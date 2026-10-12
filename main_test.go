package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/clbanning/mxj/v2"
)

// newTestParams returns a tParams with every pointer set, so tests can tweak single fields.
func newTestParams() tParams {
	return tParams{
		inputFile:      new(string),
		outputFile:     new(string),
		textTemplate:   new(string),
		inputFormat:    new(string),
		inputDelimiter: new(string),
		getVersion:     new(bool),
		getHelp:        new(bool),
		chatGPTkey:     new(string),
		chatGPTmodel:   new(string),
		chatGPTquery:   new(string),
	}
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

const bsonDump = `JgAAAAdfaWQAYQAVOXB0lMEH42tuAm5hbWUABgAAAEhlbGxvAAAmAAAAB19pZABhCmAZAkO5o4cVWbsCbmFtZQAGAAAAV29ybGQAAA==`

// TestVersionFormat guards version.txt: an empty or malformed file must not ship.
func TestVersionFormat(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`).MatchString(version) {
		t.Errorf("version.txt must contain N.N.N or N.N.N-suffix, got %q", version)
	}
}

func TestProcessTemplate(t *testing.T) {
	inputFile := ""
	inputFormat := ""
	inputDelimiter := ","
	outputFile := ""
	textTemplate := `?{{define content}}`
	getVersion := false
	getHelp := false
	chatGPTkey := ""
	chatGPTmodel := ""
	chatGPTquery := ""

	params := tParams{
		inputFile:      &inputFile,
		inputFormat:    &inputFormat,
		inputDelimiter: &inputDelimiter,
		outputFile:     &outputFile,
		textTemplate:   &textTemplate,
		getVersion:     &getVersion,
		getHelp:        &getHelp,
		chatGPTkey:     &chatGPTkey,
		chatGPTmodel:   &chatGPTmodel,
		chatGPTquery:   &chatGPTquery,
	}
	err := processTemplate(params)
	if !strings.Contains(err.Error(), "stdin: Error-noPipe") {
		t.Errorf("result: %v", err.Error())
	}
	inputFile = "testdata.xml"
	textTemplate = "hello.tmpl"
	err = processTemplate(params)
	if !strings.Contains(err.Error(), `readFile: open hello.tmpl:`) {
		t.Errorf("result: %v", err.Error())
	}
	inputFile = "testdata.xml"
	textTemplate = "?{{define content}}"
	err = processTemplate(params)
	if !strings.Contains(err.Error(), `unexpected "content" in define`) {
		t.Errorf("result: %v", err.Error())
	}
	inputFormat = "json"
	err = processTemplate(params)
	if !strings.Contains(err.Error(), `mapJSON: invalid character`) {
		t.Errorf("result: %v", err.Error())
	}
	textTemplate = `?Output template test: description = {{index .TOP_LEVEL "-description"}} {{print "\r\n"}}`
	inputFormat = ""
	err = processTemplate(params)
	if err != nil {
		t.Errorf("result: %v", err.Error())
	}

	textTemplate = `?Output template test: description = {{index .filesTest.TOP_LEVEL "-description"}} {{print "\r\n"}}`
	inputFile = "?filesTest.yaml"
	err = processTemplate(params)
	if err != nil {
		t.Errorf("result: %v", err.Error())
	}

	inputFile = "?filesTest.yamlx"
	err = processTemplate(params)
	if !strings.Contains(err.Error(), "readFileList: open filesTest.yamlx") {
		t.Errorf("result: %v", err.Error())
	}

	getVersion = true
	err = processTemplate(params)
	if err != nil {
		t.Errorf("result: %v", err.Error())
	}
	textTemplate = ""
	getVersion = false
	err = processTemplate(params)
	if err != nil {
		t.Errorf("result: %v", err.Error())
	}
}

// TestFileListValidation covers the "-i ?files.yaml" path: malformed records must produce a
// descriptive error (they used to panic) and "format" may be omitted when the extension is known.
func TestFileListValidation(t *testing.T) {
	testdata, err := filepath.Abs("testdata.xml")
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name    string
		yaml    string
		wantErr string // substring of the expected error, "" for success
	}{
		{"valid explicit format", "- file: " + testdata + "\n  format: xml\n  label: x\n", ""},
		{"format omitted, detected from extension", "- file: " + testdata + "\n  label: x\n", ""},
		{"format empty string, detected from extension", "- file: " + testdata + "\n  format: \"\"\n  label: x\n", ""},
		{"file is int", "- file: 123\n  format: xml\n  label: x\n", `fileList[0]: "file" must be a string (got int)`},
		{"label missing", "- file: " + testdata + "\n  format: xml\n", `fileList[0]: missing "label"`},
		{"format is list", "- file: " + testdata + "\n  format: [xml]\n  label: x\n", `fileList[0]: "format" must be a string`},
		{"second record broken", "- file: " + testdata + "\n  label: a\n- file: {}\n  label: b\n", `fileList[1]: "file" must be a string`},
		{"unknown extension without format", "- file: " + testdata + ".unknown\n  label: x\n", "unknown input format"},
		{"empty label", "- file: " + testdata + "\n  label: \"\"\n", `fileList[0]: "label" must not be empty`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "unknown extension without format" {
				// The file must exist so the error comes from the format detection, not from ReadFile.
				if err := os.WriteFile(testdata+".unknown", []byte("<a/>"), 0o600); err != nil {
					t.Fatal(err)
				}
				defer os.Remove(testdata + ".unknown")
			}
			params := newTestParams()
			*params.inputFile = "?" + writeTempFile(t, "files.yaml", tc.yaml)
			*params.textTemplate = `?{{index .x.TOP_LEVEL "-description"}}`
			err := processTemplate(params)
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Errorf("want error containing %q, got %v", tc.wantErr, err)
			}
		})
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

// TestOpenAIMode checks the mode switch and key resolution without reaching the network:
// the resolved key is validated before any request, and a dummy key must fail inside the OpenAI
// client (not in the template guard).
func TestOpenAIMode(t *testing.T) {
	t.Setenv(openAIKeyEnv, "")
	params := newTestParams()
	*params.inputFile = "testdata.xml"
	*params.chatGPTquery = "hello"
	err := processTemplate(params)
	if err == nil || !strings.Contains(err.Error(), "OpenAI API key must be defined") {
		t.Errorf("-gq without any key: want key error, got %v", err)
	}

	// Key from the environment: the request is attempted (and fails on the dummy key / network).
	t.Setenv(openAIKeyEnv, "dummy-key")
	err = processTemplate(params)
	if err == nil {
		t.Fatal("dummy key must not succeed")
	}
	for _, forbidden := range []string{"OpenAI API key must be defined", "template file must be defined", "readTemplate"} {
		if strings.Contains(err.Error(), forbidden) {
			t.Errorf("env key not picked up, got %v", err)
		}
	}

	// -gk alone still asks for a query (unchanged behaviour), and -t alone still runs the template
	// even if the environment holds a key.
	*params.chatGPTquery = ""
	*params.chatGPTkey = "flag-key"
	if err := processTemplate(params); err != nil {
		t.Errorf("-gk without -gq: want nil (prints usage hint), got %v", err)
	}
	*params.chatGPTkey = ""
	*params.textTemplate = `?{{index .TOP_LEVEL "-description"}}`
	if err := processTemplate(params); err != nil {
		t.Errorf("template mode with OPENAI_API_KEY exported: want nil, got %v", err)
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
	// Reaches the caller as a regular input error instead of log.Fatalf.
	if _, err := parseCSV([]byte("a,b\n1,2"), "0xZZ"); err == nil || !strings.Contains(err.Error(), "mapCSV: invalid delimiter") {
		t.Errorf("want mapCSV error, got %v", err)
	}
}

func TestGetInputData(t *testing.T) {
	inputFile := "testdata.xml"
	data, _, _ := getInputData(&inputFile)
	if !strings.Contains(string(data), `<?xml version="1.0" encoding="utf-8"?>`) {
		t.Errorf("result: %v", string(data))
	}
	inputFile = "Hello.xml"
	_, _, err := getInputData(&inputFile)
	if !strings.Contains(err.Error(), `readFile: open Hello.xml:`) {
		t.Errorf("result: %v", err.Error())
	}
	inputFile = ""
	_, _, err = getInputData(&inputFile)
	if !strings.Contains(err.Error(), `stdin: Error-noPipe`) {
		t.Errorf("result: %v", err.Error())
	}
}

func TestMapInputData(t *testing.T) {
	inputFile := ""
	inputFormat := ""
	inputDelimiter := ","
	outputFile := ""
	textTemplate := `?{{define content}}`
	getVersion := false
	params := tParams{
		inputFile:      &inputFile,
		inputFormat:    &inputFormat,
		inputDelimiter: &inputDelimiter,
		outputFile:     &outputFile,
		textTemplate:   &textTemplate,
		getVersion:     &getVersion,
	}
	// Test map json
	input := []byte(`{"name": "John","age": 30}`)
	inputFormat = "json"
	result, _ := mapInputData(input, params)
	if result.(map[string]interface{})["name"] != "John" || result.(map[string]interface{})["age"] != float64(30) {
		t.Errorf("resultJSON: %v", result)
	}
	input = []byte(`[{"name": "John","age": 30}, {"name": "Hanz","age": 28}]`)
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != "John" {
		t.Errorf("resultJSONarray: %v", result.([]map[string]interface{})[0]["name"])
	}
	input = []byte(`[{"name": "John","age": 30}, {"name": Hanz","age": 28}]`)
	_, err := mapInputData(input, params)
	if !strings.Contains(err.Error(), "invalid character 'H'") {
		t.Errorf("resultJSONerr: %v", err.Error())
	}
	input = []byte(`{"name" John","age": 30}`)
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "invalid character 'J'") {
		t.Errorf("resultJSONerr: %v", err.Error())
	}
	// Test map bson
	input = []byte{14, 0, 0, 0, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0}
	inputFormat = "bson"
	result, _ = mapInputData(input, params)
	if result.(map[string]interface{})["h"] != "w" {
		t.Errorf("resultBSON: %v", result)
	}
	input, err = base64.StdEncoding.DecodeString(bsonDump)
	if err != nil {
		t.Errorf("base64BSONdump: %v", err.Error())
	}
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != `Hello` {
		t.Errorf("resultBSONdump: %v", result.([]map[string]interface{})[0]["name"])
	}
	input = []byte{14, 0, 0, 1, 2, 104, 0, 2, 0, 0, 0, 119, 0, 0}
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "EOF") {
		t.Errorf("resultBSONerr: %v", err.Error())
	}
	// Test map yaml
	input = []byte(`name: John`)
	inputFormat = "yaml"
	result, _ = mapInputData(input, params)
	if result.(map[string]interface{})["name"] != "John" {
		t.Errorf("resultYAML: %v", result)
	}
	input = []byte("- name: John\r\n- name: Peter")
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != "John" {
		t.Errorf("resultYAMLarray: %v", result)
	}
	input = []byte("- name John\r\n- name: Peter")
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "cannot unmarshal !!str `name John`") {
		t.Errorf("resultYAMLarrayErr: %v", err.Error())
	}
	input = []byte(`name John`)
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "cannot unmarshal !!str `name John`") {
		t.Errorf("resultYAMLerr: %v", err.Error())
	}
	// Test map csv
	input = []byte("name,surname\r\nHello,World")
	inputFormat = "csv"
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != "Hello" {
		t.Errorf("result: %v", result.([]map[string]interface{})[0]["name"])
	}
	input = []byte("name,surname\r\nHello,World")
	inputDelimiter = "0x2C"
	result, _ = mapInputData(input, params)
	if result.([]map[string]interface{})[0]["name"] != "Hello" {
		t.Errorf("result: %v", result.([]map[string]interface{})[0]["name"])
	}
	input = []byte("name,surname\r\nHello,World,!!!")
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "wrong number of fields") {
		t.Errorf("result: %v", err.Error())
	}
	input = []byte("")
	_, err = mapInputData(input, params)
	if err == nil || !strings.Contains(err.Error(), "CSV has no rows") {
		t.Errorf("result: %v", err)
	}

	// Test map xml
	input = []byte(`<name>John</name>`)
	inputFormat = "xml"
	result, _ = mapInputData(input, params)
	if result.(mxj.Map)["name"] != "John" {
		t.Errorf("result: %v", result)
	}
	input = []byte(`<name>John<name>`)
	_, err = mapInputData(input, params)
	if !strings.Contains(err.Error(), "xml.Decoder.Token() - XML syntax error on line 1") {
		t.Errorf("result: %v", err.Error())
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
			got, err := parseCSV([]byte(tt.data), tt.delimiter)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("parseCSV() = %#v, want %#v", got, want)
			}
		})
	}

	t.Run("quoted delimiter and empty field", func(t *testing.T) {
		got, err := parseCSV([]byte("name§surname\n\"Hello§there\"§World\nHello§"), "§")
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
	// Test inline template
	result, err := readTemplate("?{{toXML .}}")
	if string(result) != "{{toXML .}}" {
		t.Errorf("result: %v", string(result))
	}
	if err != nil {
		t.Errorf("err: %v", err)
	}
	// Test template file
	result, err = readTemplate("template.tmpl")
	if !strings.Contains(string(result), "CSV formatted data:") {
		t.Errorf("result: %v", string(result))
	}
	if err != nil {
		t.Errorf("err: %v", err)
	}
	// Test nonExisting file
	_, err = readTemplate("hello.tmpl")
	if !strings.Contains(err.Error(), "readFile: open hello.tmpl:") {
		t.Errorf("err: %v", err)
	}
	// Empty template name must be an error, not a slice-bounds panic
	if _, err = readTemplate(""); err == nil || !strings.Contains(err.Error(), "template must be defined") {
		t.Errorf("err: %v", err)
	}
}

func TestWriteOutputData(t *testing.T) {
	testData := make(map[string]interface{})
	testData["Hello"] = "World"
	outputFile := ""
	templateFile := []byte(`{{define content}}`)
	err := writeOutputData(testData, &outputFile, templateFile)
	if !strings.Contains(err.Error(), `new:1: unexpected "content"`) {
		t.Errorf("result: %v", err.Error())
	}
	templateFile = []byte(`Output test: Hello {{.Hello}} {{print "\r\n"}}`)
	if err := writeOutputData(testData, &outputFile, templateFile); err != nil {
		t.Errorf("result: %v", err.Error())
	}
	outputFile = "output.txt"
	if err := writeOutputData(testData, &outputFile, templateFile); err != nil {
		t.Errorf("result: %v", err.Error())
	}
	testData["Hello"] = make(chan int, 1)
	err = writeOutputData(testData, &outputFile, templateFile)
	if !strings.Contains(err.Error(), "can't print {{.Hello}} of type chan int") {
		t.Errorf("result: %v", err.Error())
	}
	outputFile = "out*he\\ll//o/./txt"
	err = writeOutputData(testData, &outputFile, templateFile)
	if !strings.Contains(err.Error(), "createOutputFile:") {
		t.Errorf("result: %v", err.Error())
	}
}

func TestCleanBOM(t *testing.T) {
	input := "\xef\xbb\xbf" + "Hello"
	result := string(cleanBOM([]byte(input)))
	if result != "Hello" {
		t.Errorf("result: %v", result)
	}
	input = "Hello"
	result = string(cleanBOM([]byte(input)))
	if result != "Hello" {
		t.Errorf("result: %v", result)
	}
}
