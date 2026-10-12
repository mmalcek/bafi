//go:build !(js && wasm)

package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/alpkeskin/gotoon"
	"github.com/sashabaranov/go-openai"
	"gopkg.in/yaml.v3"
)

// options holds the parsed command line.
type options struct {
	inputFile   string
	outputFile  string
	template    string
	format      string
	delimiter   string
	strict      bool
	version     bool
	help        bool
	openAIKey   string
	openAIModel string
	openAIQuery string
}

// openAIKeyEnv is the environment variable consulted for the OpenAI API key when -gk is not given.
// Preferred over -gk: command-line arguments are visible in process listings and shell history.
const openAIKeyEnv = "OPENAI_API_KEY"

func main() {
	code := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
	if luaData != nil {
		luaData.Close()
	}
	os.Exit(code)
}

// run is the whole CLI: it parses args, reads stdin when no -i is given, writes results to stdout and
// errors/warnings to stderr, and returns the exit code (0 ok, 1 runtime error, 2 usage error).
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var o options
	fs := newFlagSet(&o)
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(stdout, fs)
			return 0
		}
		fmt.Fprint(stderr, formatError(usageError(err.Error(), "run bafi -h to see all flags", "")))
		return 2
	}
	if o.help {
		printHelp(stdout, fs)
		return 0
	}
	// A single trailing "-" is the common "read standard input" convention; bafi reads stdin anyway when
	// -i is not set, and versions before 1.5.0 ignored the argument, so keep accepting it.
	stdinMarker := fs.NArg() == 1 && fs.Arg(0) == "-" && o.inputFile == ""
	if fs.NArg() > 0 && !stdinMarker {
		hint := `flags must come first and inline templates with spaces must be quoted: -t '?{{toJSON .}}'`
		if fs.Arg(0) == "-" {
			hint = `"-" means standard input, but -i already names the input file; remove one of them`
		}
		fmt.Fprint(stderr, formatError(usageError(fmt.Sprintf("unexpected argument %q", fs.Arg(0)), hint, "unexpected-argument")))
		return 2
	}
	warn := func(msg string) { fmt.Fprintln(stderr, "bafi: warning: "+msg) }
	if err := execute(o, stdin, stdout, warn); err != nil {
		fmt.Fprint(stderr, formatError(err))
		return exitCode(err)
	}
	return 0
}

func newFlagSet(o *options) *flag.FlagSet {
	fs := flag.NewFlagSet("bafi", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // run() reports parse errors itself
	fs.StringVar(&o.inputFile, "i", "", "input file (default: read stdin)\nprefix with ? to merge several files listed in a YAML file: -i '?files.yaml'")
	fs.StringVar(&o.outputFile, "o", "", "output file (default: write stdout)")
	fs.StringVar(&o.template, "t", "", "template file, or an inline template starting with ?: -t '?{{toJSON .}}'")
	fs.StringVar(&o.format, "f", "", "input format: "+strings.Join(inputFormats, ", ")+" (default: from the -i file extension)")
	fs.StringVar(&o.delimiter, "d", "", "CSV delimiter, a character or a hex byte: -d ';' or -d 0x09 (default ,)\nMT940: message separator for files with several messages")
	fs.BoolVar(&o.strict, "strict", false, "fail on a key that is missing from the input instead of printing <no value>")
	fs.BoolVar(&o.version, "v", false, "show version")
	fs.BoolVar(&o.help, "h", false, "show this help")
	fs.StringVar(&o.openAIKey, "gk", "", "OpenAI API key (default: environment variable "+openAIKeyEnv+")")
	fs.StringVar(&o.openAIModel, "gm", "gpt35", "OpenAI model: gpt35, gpt4, gpt4o, gpt4o-mini, gpt5, gpt5mini, gpt5nano")
	fs.StringVar(&o.openAIQuery, "gq", "", "OpenAI query; sends the query and the whole parsed input to OpenAI")
	return fs
}

const helpHeader = `bafi - convert JSON, XML, CSV, YAML, BSON and MT940 into any text format with templates

Usage:
  bafi -i INPUT -t TEMPLATE [-o OUTPUT]
  COMMAND | bafi -f FORMAT -t TEMPLATE

Examples:
  bafi -i data.xml -t '?{{toJSON .}}'
      convert an XML file to JSON
  bafi -i users.csv -d ';' -t report.tmpl -o report.html
      render a semicolon-separated CSV with a template file
  curl -fs https://example.com/api/users | bafi -f json -t '?{{toXML .}}'
      convert a REST response to XML

Flags:
`

func printHelp(w io.Writer, fs *flag.FlagSet) {
	fmt.Fprint(w, helpHeader)
	fs.SetOutput(w)
	fs.PrintDefaults()
	fs.SetOutput(io.Discard)
	fmt.Fprintf(w, "\nRecipes and documentation: %s\n", docsURL)
}

// execute runs one invocation. warn receives non-fatal findings for stderr.
func execute(o options, stdin io.Reader, stdout io.Writer, warn func(string)) error {
	if o.version {
		fmt.Fprintf(stdout, "Version: %s\r\nProject page: https://github.com/mmalcek/bafi\r\n", version)
		return nil
	}
	// OpenAI mode is selected by either OpenAI flag; the key itself may also come from the environment.
	openAIMode := o.openAIKey != "" || o.openAIQuery != ""
	if o.template == "" && !openAIMode {
		return usageError("no template given",
			`pass a template file with -t report.tmpl, or an inline template with -t '?{{toJSON .}}'`, "")
	}
	var apiKey string
	if openAIMode {
		if o.openAIQuery == "" {
			return usageError("-gk is set but the OpenAI query is missing", `add the question with -gq "What is the weather like?"`, "")
		}
		if apiKey = o.openAIKey; apiKey == "" {
			apiKey = os.Getenv(openAIKeyEnv)
		}
		if apiKey == "" {
			return usageError("OpenAI API key must be defined",
				"set the environment variable "+openAIKeyEnv+" (preferred) or pass -gk <key>", "")
		}
	}
	if o.format != "" && !knownFormat(o.format) {
		return unknownFormatError(o.format)
	}

	mapData, err := loadInput(o, stdin, warn)
	if err != nil {
		return err
	}
	if openAIMode {
		return askOpenAI(mapData, o, apiKey, stdout)
	}

	tmpl, err := readTemplate(o.template)
	if err != nil {
		return err
	}
	name := templateName(o.template)
	warnNoValue := func(n int) {
		if n > 0 && !o.strict {
			warn(fmt.Sprintf("output contains <no value> %d time(s) - a key used in the template is missing from the input "+
				"(or is null); run with -strict to see which one", n))
		}
	}
	// Render completely before touching the destination: a failing template leaves stdout empty and an
	// existing -o file untouched (a scheduled job keeps its last good report). The warning goes out
	// first so it does not get glued to output that lacks a final newline in a terminal.
	var buf bytes.Buffer
	noValues, err := render(&buf, mapData, tmpl, name, o.strict)
	if err != nil {
		return err
	}
	warnNoValue(noValues)
	if o.outputFile == "" {
		if _, err := stdout.Write(buf.Bytes()); err != nil {
			return &bafiError{Stage: stageOutput, Summary: "cannot write to stdout", Detail: err.Error(), Err: err}
		}
		return nil
	}
	output, err := os.Create(o.outputFile)
	if err != nil {
		return outputError(o.outputFile, err)
	}
	if _, err := output.Write(buf.Bytes()); err != nil {
		output.Close()
		return outputError(o.outputFile, err)
	}
	if err := output.Close(); err != nil {
		return outputError(o.outputFile, err)
	}
	return nil
}

// hasInput reports whether stdin carries data: a pipe or a redirected file, not an interactive terminal.
func hasInput(r io.Reader) bool {
	if r == nil {
		return false
	}
	f, ok := r.(*os.File)
	if !ok {
		return true
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice == 0
}

// loadInput reads and parses stdin, the -i file, or every file of a "?files.yaml" list.
func loadInput(o options, stdin io.Reader, warn func(string)) (interface{}, error) {
	switch {
	case o.inputFile == "":
		if !hasInput(stdin) {
			return nil, usageError("no input",
				"read a file with -i data.json, or pipe data in: cat data.json | bafi -f json -t '?{{toXML .}}'", "no-input")
		}
		data, err := io.ReadAll(stdin)
		if err != nil {
			return nil, &bafiError{Stage: stageInput, Summary: "cannot read stdin", Detail: err.Error(), Err: err}
		}
		data = cleanBOM(data)
		if o.format == "" {
			return nil, missingFormatError("stdin", data)
		}
		return mapInputData(data, inputSpec{Source: "stdin", Format: o.format, Delimiter: o.delimiter}, warn)
	case strings.HasPrefix(o.inputFile, "?"):
		return loadFileList(o.inputFile[1:], o.delimiter, warn)
	default:
		data, err := os.ReadFile(o.inputFile)
		if err != nil {
			return nil, fileError(stageInput, "input file", o.inputFile, err)
		}
		data = cleanBOM(data)
		format := o.format
		if format == "" {
			if format = detectFormat(o.inputFile); format == "" {
				return nil, missingFormatError(o.inputFile, data)
			}
		}
		return mapInputData(data, inputSpec{Source: o.inputFile, Format: format, Delimiter: o.delimiter}, warn)
	}
}

// loadFileList parses every file named in a YAML file list into a map keyed by the entries' labels.
// -f is ignored here: each entry has its own format (or its file extension decides).
func loadFileList(listPath, delimiter string, warn func(string)) (interface{}, error) {
	raw, err := os.ReadFile(listPath)
	if err != nil {
		return nil, fileError(stageInput, "file list", listPath, err)
	}
	files := make([]map[string]interface{}, 0)
	if err := yaml.Unmarshal(raw, &files); err != nil {
		be := yamlError(raw, listPath, err).(*bafiError)
		be.Summary = fmt.Sprintf("cannot parse file list %q", listPath)
		be.Hint = "the file list is a YAML list of entries with file, label and an optional format, e.g. - file: rates.xml"
		be.Anchor = "invalid-file-list"
		return nil, be
	}
	filesStruct := make(map[string]interface{})
	for i, file := range files {
		entry, err := parseFileListEntry(i, file)
		if err != nil {
			return nil, &bafiError{
				Stage:   stageInput,
				Summary: fmt.Sprintf("invalid entry in file list %q", listPath),
				Detail:  err.Error(),
				Hint:    "every entry needs file and label as text, format is optional: - file: rates.xml / label: RATES",
				Anchor:  "invalid-file-list",
			}
		}
		if entry.format != "" && !knownFormat(entry.format) {
			be := unknownFormatError(entry.format).(*bafiError)
			be.Stage = stageInput
			be.Summary = fmt.Sprintf("unknown format %q in file list %q (fileList[%d])", entry.format, listPath, i)
			be.Hint = strings.Replace(be.Hint, "did you mean -f ", "did you mean format: ", 1)
			return nil, be
		}
		if entry.format == "" {
			return nil, &bafiError{
				Stage:   stageInput,
				Summary: fmt.Sprintf("cannot tell the format of %q (fileList[%d] in %q)", entry.file, i, listPath),
				Hint:    "add format: with one of " + strings.Join(inputFormats, ", ") + " to the entry",
				Anchor:  "unknown-input-format",
			}
		}
		data, err := os.ReadFile(entry.file)
		if err != nil {
			return nil, fileError(stageInput, "input file", entry.file, err)
		}
		spec := inputSpec{Source: entry.file, Format: entry.format, Delimiter: delimiter}
		if filesStruct[entry.label], err = mapInputData(cleanBOM(data), spec, warn); err != nil {
			return nil, err
		}
	}
	return &filesStruct, nil
}

// fileListEntry is one validated record of the "-i ?files.yaml" file list.
type fileListEntry struct {
	file   string
	format string
	label  string
}

// parseFileListEntry validates one record of the file list. "file" and "label" are required strings;
// "format" is optional and falls back to the file extension (an unknown extension is reported later,
// same as for a single input file). A malformed record yields an error naming the record index and key.
func parseFileListEntry(index int, entry map[string]interface{}) (fileListEntry, error) {
	getString := func(key string, required bool) (string, error) {
		raw, ok := entry[key]
		if !ok || raw == nil {
			if required {
				return "", fmt.Errorf("fileList[%d]: missing %q", index, key)
			}
			return "", nil
		}
		value, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("fileList[%d]: %q must be a string (got %T)", index, key, raw)
		}
		if required && value == "" {
			return "", fmt.Errorf("fileList[%d]: %q must not be empty", index, key)
		}
		return value, nil
	}
	var result fileListEntry
	var err error
	if result.file, err = getString("file", true); err != nil {
		return result, err
	}
	if result.label, err = getString("label", true); err != nil {
		return result, err
	}
	if result.format, err = getString("format", false); err != nil {
		return result, err
	}
	if result.format == "" {
		result.format = detectFormat(result.file)
	}
	return result, nil
}

// templateName is the name templates are parsed under; it appears in error messages.
func templateName(templateArg string) string {
	if templateArg == "" || templateArg[0] == '?' {
		return "inline"
	}
	return filepath.Base(templateArg)
}

// fileError words a failed read of an input, template or file-list file.
func fileError(stage errStage, what, path string, err error) error {
	be := &bafiError{
		Stage:   stage,
		Summary: fmt.Sprintf("cannot read %s %q", what, path),
		Detail:  err.Error(),
		Anchor:  "file-not-found",
		Err:     err,
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		be.Detail = pe.Err.Error()
	}
	switch {
	case errors.Is(err, fs.ErrNotExist):
		be.Hint = "check the path; relative paths are resolved from the current directory"
		if what == "template" && !strings.HasPrefix(path, "?") && strings.Contains(path, "{{") {
			be.Hint = `inline templates must start with "?", e.g. -t '?{{toJSON .}}'`
		}
	case errors.Is(err, fs.ErrPermission):
		be.Hint = "bafi is not allowed to read this file; check its permissions"
	}
	return be
}

// outputError words a failed creation of the -o file.
func outputError(path string, err error) error {
	be := &bafiError{
		Stage:   stageOutput,
		Summary: fmt.Sprintf("cannot create output file %q", path),
		Detail:  err.Error(),
		Anchor:  "cannot-create-output-file",
		Err:     err,
	}
	var pe *fs.PathError
	if errors.As(err, &pe) {
		be.Detail = pe.Err.Error()
	}
	switch {
	case errors.Is(err, fs.ErrNotExist):
		be.Hint = "the directory does not exist; create it first"
	case errors.Is(err, fs.ErrPermission):
		be.Hint = "bafi may not write there; choose another directory or check permissions"
	}
	return be
}

// readTemplate get template from file or from input
func readTemplate(textTemplate string) ([]byte, error) {
	if textTemplate == "" {
		return nil, usageError("no template given", `pass -t template.tmpl or an inline template -t '?{{toJSON .}}'`, "")
	}
	if textTemplate[0] == '?' {
		return []byte(textTemplate[1:]), nil
	}
	templateFile, err := os.ReadFile(textTemplate)
	if err != nil {
		return nil, fileError(stageTemplate, "template", textTemplate, err)
	}
	return templateFile, nil
}

// askOpenAI sends the query with the parsed input and writes the answer to stdout or -o.
func askOpenAI(mapData interface{}, o options, apiKey string, stdout io.Writer) error {
	response, err := chatGPTprocess(mapData, o.openAIModel, o.openAIQuery, apiKey)
	if err != nil {
		be := &bafiError{Stage: stageOpenAI, Summary: "OpenAI request failed", Detail: err.Error(), Err: err,
			Hint: "check the network connection and the model name (-gm)"}
		switch msg := err.Error(); {
		case strings.Contains(msg, "401"):
			be.Hint = "the API key was rejected; check " + openAIKeyEnv + " or -gk"
		case strings.Contains(msg, "429"):
			be.Hint = "rate limit or quota exceeded on your OpenAI account"
		}
		return be
	}
	if len(response.Choices) == 0 {
		return &bafiError{Stage: stageOpenAI, Summary: "OpenAI returned an empty response (no choices)"}
	}
	content := response.Choices[0].Message.Content
	if o.outputFile == "" {
		fmt.Fprintln(stdout, content)
		return nil
	}
	output, err := os.Create(o.outputFile)
	if err != nil {
		return outputError(o.outputFile, err)
	}
	defer output.Close()
	if _, err := output.WriteString(content); err != nil {
		return outputError(o.outputFile, err)
	}
	return nil
}

// chatGPTprocess sends the query and the whole parsed input (TOON encoded) to OpenAI.
func chatGPTprocess(mapData interface{}, modelName, query, apiKey string) (response openai.ChatCompletionResponse, err error) {
	toonData, err := gotoon.Encode(mapData)
	if err != nil {
		return response, fmt.Errorf("cannot encode the input for OpenAI: %w", err)
	}
	model := openai.GPT3Dot5Turbo
	switch modelName {
	case "gpt35":
		model = openai.GPT3Dot5Turbo
	case "gpt4":
		model = openai.GPT4
	case "gpt4o":
		model = openai.GPT4o
	case "gpt4o-mini":
		model = openai.GPT4oMini
	case "gpt5":
		model = openai.GPT5
	case "gpt5mini":
		model = openai.GPT5Mini
	case "gpt5nano":
		model = openai.GPT5Nano
	}

	client := openai.NewClient(apiKey)
	return client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: query + "\n" + string(toonData),
				},
			},
		},
	)
}
