package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

// docsURL is the documentation site; troubleshooting anchors are appended to it in hints.
const docsURL = "https://mmalcek.github.io/bafi/"

// errStage classifies a failure. It decides the exit code: usage mistakes exit 2, everything else 1.
type errStage string

const (
	stageUsage    errStage = "usage"
	stageInput    errStage = "input"
	stageTemplate errStage = "template"
	stageOutput   errStage = "output"
	stageOpenAI   errStage = "openai"
)

// srcLine is one numbered line of the input or template shown under an error.
type srcLine struct {
	N    int
	Text string
}

// bafiError is the single error type bafi reports to users. Error() is one greppable line that always
// contains the underlying parser message; formatError renders the multi-line form with the source
// excerpt, a caret, a hint and a link to the troubleshooting page.
type bafiError struct {
	Stage   errStage
	Summary string    // what failed, e.g. `cannot parse JSON input from stdin`
	Line    int       // 1-based; 0 = unknown
	Col     int       // 1-based rune column; 0 = no caret
	Near    bool      // the location is approximate (YAML): "near line N", no caret
	Snippet []srcLine // lines to show; the one numbered Line gets the caret
	Detail  string    // underlying message without nested "template: x:1:" style prefixes
	Hint    string    // how to fix it
	Anchor  string    // heading id on docs/troubleshooting.md
	Err     error     // original error, for errors.Is/As
}

func (e *bafiError) Error() string {
	var b strings.Builder
	b.WriteString(e.Summary)
	if loc := e.location(); loc != "" {
		fmt.Fprintf(&b, " (%s)", loc)
	}
	if e.Detail != "" {
		fmt.Fprintf(&b, ": %s", strings.ReplaceAll(e.Detail, "\n", "; "))
	}
	return b.String()
}

func (e *bafiError) Unwrap() error { return e.Err }

func (e *bafiError) location() string {
	switch {
	case e.Line <= 0:
		return ""
	case e.Near:
		return fmt.Sprintf("near line %d", e.Line)
	case e.Col > 0:
		return fmt.Sprintf("line %d, column %d", e.Line, e.Col)
	default:
		return fmt.Sprintf("line %d", e.Line)
	}
}

// exitCode maps an error to the process exit code: 2 for usage mistakes, 1 for everything else.
func exitCode(err error) int {
	var be *bafiError
	if errors.As(err, &be) && be.Stage == stageUsage {
		return 2
	}
	return 1
}

// shortUsage is appended to usage errors.
const shortUsage = "usage: bafi -i INPUT -t TEMPLATE [-o OUTPUT]   (bafi -h for help and examples)"

// formatError renders err for humans (stderr, the online demo). Plain errors get a "bafi: " prefix.
func formatError(err error) string {
	var be *bafiError
	if !errors.As(err, &be) {
		return "bafi: " + err.Error() + "\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "bafi: %s", be.Summary)
	if loc := be.location(); loc != "" {
		fmt.Fprintf(&b, " (%s)", loc)
	}
	b.WriteString("\n")
	if len(be.Snippet) > 0 {
		b.WriteString("\n")
		writeSnippet(&b, be.Snippet, be.Line, be.Col, be.Near)
	}
	for l := range strings.SplitSeq(be.Detail, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			fmt.Fprintf(&b, "  %s\n", l)
		}
	}
	if be.Hint != "" {
		fmt.Fprintf(&b, "\nhint: %s\n", be.Hint)
	}
	if be.Anchor != "" {
		if be.Hint == "" {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "docs: %stroubleshooting/#%s\n", docsURL, be.Anchor)
	}
	if be.Stage == stageUsage {
		b.WriteString("\n" + shortUsage + "\n")
	}
	return b.String()
}

// snippetWindow is how many runes are kept either side of the caret on very long lines (minified JSON).
const snippetWindow = 40

// writeSnippet prints numbered source lines and a caret under column col of line caretLine.
func writeSnippet(b *strings.Builder, lines []srcLine, caretLine, col int, near bool) {
	width := 1
	for _, l := range lines {
		if w := len(strconv.Itoa(l.N)); w > width {
			width = w
		}
	}
	for _, l := range lines {
		text, c := clipLine(l.Text, col)
		if l.N != caretLine {
			text, _ = clipLine(l.Text, 0)
		}
		fmt.Fprintf(b, "  %*d | %s\n", width, l.N, text)
		if l.N == caretLine && c > 0 && !near {
			fmt.Fprintf(b, "  %*s | %s^\n", width, "", caretPadding(text, c))
		}
	}
}

// clipLine shortens a long line to a window around the 1-based rune column col (col 0: from the start)
// and returns the column adjusted to the clipped text.
func clipLine(line string, col int) (string, int) {
	line = strings.TrimRight(line, "\r")
	runes := []rune(line)
	limit := 2*snippetWindow + 20
	if len(runes) <= limit {
		return line, col
	}
	center := col - 1
	if center < 0 {
		center = 0
	}
	start := center - snippetWindow
	if start < 0 {
		start = 0
	}
	end := start + 2*snippetWindow
	if end > len(runes) {
		end = len(runes)
		if start = end - 2*snippetWindow; start < 0 {
			start = 0
		}
	}
	var b strings.Builder
	newCol := col - start
	if start > 0 {
		b.WriteString("…")
		newCol++
	}
	b.WriteString(string(runes[start:end]))
	if end < len(runes) {
		b.WriteString("…")
	}
	if col == 0 {
		newCol = 0
	}
	return b.String(), newCol
}

// caretPadding returns the whitespace that places a caret under rune column col, copying tabs so the
// caret lines up however the terminal renders them.
func caretPadding(line string, col int) string {
	var b strings.Builder
	i := 1
	for _, r := range line {
		if i >= col {
			break
		}
		if r == '\t' {
			b.WriteRune('\t')
		} else {
			b.WriteRune(' ')
		}
		i++
	}
	for ; i < col; i++ {
		b.WriteRune(' ')
	}
	return b.String()
}

// splitLines splits source text into lines without their line endings.
func splitLines(data []byte) []string {
	return strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
}

// excerpt returns line n (1-based) plus `context` lines either side, within bounds.
func excerpt(data []byte, n, context int) []srcLine {
	lines := splitLines(data)
	if n < 1 || n > len(lines) {
		return nil
	}
	var out []srcLine
	for i := n - context; i <= n+context; i++ {
		if i >= 1 && i <= len(lines) {
			if i != n && strings.TrimSpace(lines[i-1]) == "" {
				continue
			}
			out = append(out, srcLine{N: i, Text: lines[i-1]})
		}
	}
	return out
}

// offsetToLineCol converts a 0-based byte offset into a 1-based line and 1-based rune column.
func offsetToLineCol(data []byte, offset int) (line, col int) {
	if offset < 0 {
		offset = 0
	}
	if offset > len(data) {
		offset = len(data)
	}
	before := data[:offset]
	line = bytes.Count(before, []byte("\n")) + 1
	lineStart := bytes.LastIndexByte(before, '\n') + 1
	col = utf8.RuneCount(before[lineStart:]) + 1
	return line, col
}

// byteColToRuneCol converts a 0-based byte offset within line into a 1-based rune column.
func byteColToRuneCol(line string, byteCol int) int {
	if byteCol < 0 {
		return 0
	}
	if byteCol > len(line) {
		byteCol = len(line)
	}
	return utf8.RuneCountInString(line[:byteCol]) + 1
}

// sourcePhrase words where input came from: "from stdin" or `from "data.json"`.
func sourcePhrase(source string) string {
	if source == "stdin" || source == demoSource {
		return "from " + source
	}
	return fmt.Sprintf("from %q", source)
}

// ---------------------------------------------------------------------------------------------------
// Input errors

func emptyInputError(source, format string) error {
	hint := "the file is empty"
	if source == "stdin" {
		hint = "nothing arrived on stdin - did the previous command in the pipe fail? (with curl use -f so HTTP errors are reported)"
	}
	return &bafiError{
		Stage:   stageInput,
		Summary: fmt.Sprintf("%s input %s is empty", format, sourcePhrase(source)),
		Hint:    hint,
		Anchor:  "input-is-empty",
	}
}

// unknownFormatError reports an -f value bafi does not support, suggesting the closest one.
func unknownFormatError(format string) error {
	hint := "supported formats: " + strings.Join(inputFormats, ", ")
	if s := didYouMean(strings.ToLower(format), inputFormats); s != "" {
		hint = fmt.Sprintf("did you mean -f %s? (%s)", s, hint)
	}
	return &bafiError{
		Stage:   stageUsage,
		Summary: fmt.Sprintf("unknown input format %q", format),
		Hint:    hint,
		Anchor:  "unknown-input-format",
	}
}

// missingFormatError is reported when -f is not given and the format cannot be derived from a file name.
func missingFormatError(source string, data []byte) error {
	summary := "cannot tell the input format of stdin"
	if source != "stdin" {
		summary = fmt.Sprintf("cannot tell the input format of %q from its file extension", source)
	}
	hint := "add -f with one of: " + strings.Join(inputFormats, ", ")
	if guess := sniffFormat(data); guess != "" {
		hint = fmt.Sprintf("add -f %s (the input looks like %s); supported formats: %s",
			guess, strings.ToUpper(guess), strings.Join(inputFormats, ", "))
	}
	return &bafiError{Stage: stageUsage, Summary: summary, Hint: hint, Anchor: "unknown-input-format"}
}

func jsonError(data []byte, source string, err error) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return emptyInputError(source, "JSON")
	}
	be := &bafiError{
		Stage:   stageInput,
		Summary: "cannot parse JSON input " + sourcePhrase(source),
		Detail:  strings.TrimPrefix(err.Error(), "json: "),
		Anchor:  "invalid-json",
		Err:     err,
	}
	var syntax *json.SyntaxError
	var typeErr *json.UnmarshalTypeError
	switch {
	case errors.As(err, &syntax):
		offset := int(syntax.Offset) - 1 // Offset counts the bad byte itself
		if strings.Contains(err.Error(), "unexpected end of JSON input") {
			offset = len(bytes.TrimRight(data, " \t\r\n")) - 1
			be.Hint = "the JSON ends too early - a closing } or ] is missing, or the download was cut off"
		}
		if offset < 0 {
			offset = 0
		}
		// For a multibyte character the offset can land inside it; step back to its first byte.
		for offset > 0 && offset < len(data) && !utf8.RuneStart(data[offset]) {
			offset--
		}
		be.Line, be.Col = offsetToLineCol(data, offset)
		be.Snippet = excerpt(data, be.Line, 0)
		if be.Hint == "" {
			be.Hint = jsonSyntaxHint(data, offset)
		}
	case errors.As(err, &typeErr):
		be.Hint = "bafi needs a JSON object {...} or an array of objects [{...}, ...] at the top level"
	}
	return be
}

// jsonSyntaxHint explains the most common JSON mistakes based on the byte where parsing stopped.
func jsonSyntaxHint(data []byte, offset int) string {
	if offset >= len(data) {
		return ""
	}
	bad, _ := utf8.DecodeRune(data[offset:])
	prev := bytes.TrimRight(data[:offset], " \t\r\n")
	switch {
	case (bad == '}' || bad == ']') && len(prev) > 0 && prev[len(prev)-1] == ',':
		return fmt.Sprintf("remove the trailing comma before %c - JSON does not allow it", bad)
	case bad == '\'':
		return `JSON strings need double quotes ("text"), not single quotes`
	case unicode.IsLetter(bad):
		word := data[offset:]
		end := bytes.IndexFunc(word, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' })
		if end > 0 {
			word = word[:end]
		}
		if len(prev) > 0 && (prev[len(prev)-1] == '{' || prev[len(prev)-1] == ',') {
			return fmt.Sprintf(`object keys must be in double quotes: "%s"`, word)
		}
		return fmt.Sprintf(`text values must be in double quotes, e.g. "%s" (only true, false and null are bare words)`, word)
	case bad == '"' && len(prev) > 0 && prev[len(prev)-1] != ',' && prev[len(prev)-1] != '{' && prev[len(prev)-1] != '[' && prev[len(prev)-1] != ':':
		return "a comma is missing between two values"
	}
	return "check the JSON at the marked position; `jq . file.json` shows the same error with more context"
}

var yamlLineRe = regexp.MustCompile(`line (\d+):`)

func yamlError(data []byte, source string, err error) error {
	msg := strings.TrimPrefix(err.Error(), "yaml: ")
	be := &bafiError{
		Stage:   stageInput,
		Summary: "cannot parse YAML input " + sourcePhrase(source),
		Detail:  msg,
		Anchor:  "invalid-yaml",
		Err:     err,
		Near:    true,
	}
	var typeErr *yaml.TypeError
	isTypeErr := errors.As(err, &typeErr)
	if isTypeErr {
		be.Detail = strings.TrimPrefix(strings.TrimPrefix(msg, "unmarshal errors:\n"), "unmarshal errors:")
		be.Detail = strings.TrimSpace(be.Detail)
	}
	if m := yamlLineRe.FindStringSubmatch(msg); m != nil {
		be.Line, _ = strconv.Atoi(m[1])
		// yaml.v3 parser errors report the line before the problem, so show one line either side.
		be.Snippet = excerpt(data, be.Line, 1)
		be.Detail = strings.Replace(be.Detail, m[0]+" ", "", 1)
	}
	switch {
	case isTypeErr:
		be.Hint = "bafi needs a YAML mapping (key: value) or a list of mappings (- key: value) at the top level"
	case bytes.Contains(data, []byte("\n\t")) || bytes.HasPrefix(data, []byte("\t")):
		be.Hint = "YAML does not allow TAB characters for indentation - indent with spaces"
	case strings.Contains(msg, "mapping values are not allowed"):
		be.Hint = `check the indentation; a value containing ": " must be quoted, e.g. title: "Note: read me"`
	case strings.Contains(msg, "did not find expected"):
		be.Hint = "an opening [ { or quote is not closed, or a list item is indented differently from its siblings"
	default:
		be.Hint = "check the indentation around this line; YAML is indentation-sensitive"
	}
	return be
}

var xmlLineRe = regexp.MustCompile(`on line (\d+)`)

func xmlError(data []byte, source string, err error) error {
	if len(bytes.TrimSpace(data)) == 0 {
		return emptyInputError(source, "XML")
	}
	if errors.Is(err, io.EOF) {
		// mxj reports io.EOF when the data contains no XML element at all, e.g. plain text or JSON.
		hint := "XML input needs a root element, e.g. <data>...</data>"
		if guess := sniffFormat(data); guess != "" && guess != "xml" {
			hint = fmt.Sprintf("the input looks like %s - use -f %s (or the .%s file extension)", strings.ToUpper(guess), guess, guess)
		}
		return &bafiError{
			Stage:   stageInput,
			Summary: "cannot parse XML input " + sourcePhrase(source),
			Detail:  "no XML element found",
			Hint:    hint,
			Anchor:  "invalid-xml",
			Err:     err,
		}
	}
	msg := strings.TrimPrefix(err.Error(), "xml.Decoder.Token() - ")
	be := &bafiError{
		Stage:   stageInput,
		Summary: "cannot parse XML input " + sourcePhrase(source),
		Detail:  msg,
		Anchor:  "invalid-xml",
		Err:     err,
	}
	// mxj only returns the message text, so re-run encoding/xml on the same bytes to get the exact position.
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = true
	for {
		if _, derr := d.Token(); derr != nil {
			if derr != io.EOF {
				be.Line, be.Col = d.InputPos()
			}
			break
		}
	}
	if be.Line == 0 {
		if m := xmlLineRe.FindStringSubmatch(msg); m != nil {
			be.Line, _ = strconv.Atoi(m[1])
		}
	}
	if be.Line > 0 {
		be.Snippet = excerpt(data, be.Line, 0)
		if len(be.Snippet) == 1 && be.Col > 0 {
			// InputPos counts bytes; the caret works in runes.
			be.Col = byteColToRuneCol(be.Snippet[0].Text, be.Col-1)
			be.Col = xmlCaretColumn(be.Snippet[0].Text, msg, be.Col)
		} else {
			be.Col = 0
		}
	}
	switch {
	case strings.Contains(msg, "closed by"):
		be.Hint = "a closing tag does not match the element that is open; check for a typo or a missing closing tag"
	case strings.Contains(msg, "invalid character entity") || strings.Contains(msg, "&"):
		be.Hint = "a bare & must be written as &amp; (and < as &lt;) inside XML text"
	case strings.Contains(msg, "unexpected EOF"):
		be.Hint = "the document ends before all elements are closed, or the download was cut off"
	case strings.Contains(msg, "encoding") && strings.Contains(msg, "CharsetReader"):
		be.Hint = "only UTF-8 XML is supported; convert the file first, e.g. iconv -f WINDOWS-1250 -t UTF-8"
	default:
		be.Hint = "check the XML at the marked position; every element needs a matching closing tag"
	}
	return be
}

var (
	xmlClosedByRe    = regexp.MustCompile(`closed by (</[^>]*>)`)
	xmlInvalidNameRe = regexp.MustCompile(`invalid XML name: (\S+)`)
)

// xmlCaretColumn moves the caret from where encoding/xml stopped (just past the problem) onto the
// problem itself: the mismatched closing tag, the bare &, or one character back.
func xmlCaretColumn(line, msg string, col int) int {
	runes := []rune(line)
	if col > len(runes)+1 {
		col = len(runes) + 1
	}
	before := string(runes[:col-1])
	if m := xmlClosedByRe.FindStringSubmatch(msg); m != nil {
		if i := strings.LastIndex(before, m[1]); i >= 0 {
			return utf8.RuneCountInString(before[:i]) + 1
		}
	}
	if m := xmlInvalidNameRe.FindStringSubmatch(msg); m != nil {
		// The decoder stops inside the name; point at its first character.
		upTo := col
		if upTo > len(runes) {
			upTo = len(runes)
		}
		if i := strings.LastIndex(string(runes[:upTo]), m[1]); i >= 0 {
			return utf8.RuneCountInString(string(runes[:upTo])[:i]) + 1
		}
	}
	if strings.Contains(msg, "entity") {
		if i := strings.LastIndex(before, "&"); i >= 0 {
			return utf8.RuneCountInString(before[:i]) + 1
		}
	}
	if col > 1 {
		return col - 1
	}
	return col
}

func csvError(data []byte, source string, comma rune, err error) error {
	be := &bafiError{
		Stage:   stageInput,
		Summary: "cannot parse CSV input " + sourcePhrase(source),
		Detail:  err.Error(),
		Anchor:  "invalid-csv",
		Err:     err,
	}
	var pe *csv.ParseError
	if !errors.As(err, &pe) {
		return be
	}
	be.Detail = pe.Err.Error()
	be.Line = pe.Line
	be.Snippet = excerpt(data, pe.Line, 0)
	if len(be.Snippet) == 1 && !errors.Is(pe.Err, csv.ErrFieldCount) {
		be.Col = byteColToRuneCol(be.Snippet[0].Text, pe.Column-1)
	}
	switch {
	case errors.Is(pe.Err, csv.ErrFieldCount):
		be.Anchor = "csv-wrong-number-of-fields"
		header, got := csvFieldCounts(data, comma, pe.StartLine)
		if header > 0 && got > 0 {
			be.Detail = fmt.Sprintf("wrong number of fields: the header has %d, this record has %d", header, got)
		}
		be.Hint = "every row needs as many fields as the header; quote values that contain the delimiter " +
			`("Smith, John"), or set the delimiter with -d`
	case errors.Is(pe.Err, csv.ErrBareQuote):
		be.Hint = `a quote inside an unquoted field: quote the whole field and double the inner quotes, e.g. "He said ""hi"""`
	case errors.Is(pe.Err, csv.ErrQuote):
		be.Hint = `a quoted field is not closed, or a quote inside it is not doubled ("")`
	}
	return be
}

// csvFieldCounts re-reads the CSV leniently and returns the header width and the width of the record
// that starts on startLine.
func csvFieldCounts(data []byte, comma rune, startLine int) (header, got int) {
	r := csv.NewReader(bytes.NewReader(data))
	r.Comma = comma
	r.FieldsPerRecord = -1
	for i := 0; ; i++ {
		rec, err := r.Read()
		if err != nil {
			return header, 0
		}
		if i == 0 {
			header = len(rec)
		}
		if line, _ := r.FieldPos(0); line == startLine {
			return header, len(rec)
		}
	}
}

// ---------------------------------------------------------------------------------------------------
// Template errors

var (
	tmplParseLineRe   = regexp.MustCompile(`^(\d+): `)
	tmplStartedAtRe   = regexp.MustCompile(`started at [^:]*:(\d+)`)
	tmplExecLocRe     = regexp.MustCompile(`^(\d+):(\d+): executing "[^"]*" at <(.*?)>: `)
	tmplUnknownFuncRe = regexp.MustCompile(`function "([^"]+)" not defined`)
	tmplWrongTypeRe   = regexp.MustCompile(`wrong type for value; expected (\S+); got (\S+)`)
	tmplArgCountRe    = regexp.MustCompile(`wrong number of args for (\S+): want (\d+) got (\d+)`)
	tmplMissingKeyRe  = regexp.MustCompile(`map has no entry for key "([^"]*)"`)
	tmplNoFieldRe     = regexp.MustCompile(`can't evaluate field (\S+) in type`)
	tmplErrCallingRe  = regexp.MustCompile(`^error calling (\S+): `)
)

// builtinTemplateFunctions are the text/template built-ins, used for "did you mean" suggestions.
var builtinTemplateFunctions = []string{"and", "call", "html", "index", "slice", "js", "len", "not", "or",
	"print", "printf", "println", "urlquery", "eq", "ge", "gt", "le", "lt", "ne", "block", "break",
	"continue", "define", "else", "end", "if", "range", "template", "with"}

// templateError converts a text/template parse or execution error into a bafiError pointing at the
// template line. name is the template name used for parsing (a base file name or "inline").
func templateError(err error, name string, src []byte) error {
	msg := err.Error()
	prefix := "template: " + name + ":"
	rest := strings.TrimPrefix(msg, prefix)
	be := &bafiError{
		Stage:   stageTemplate,
		Summary: "template error in " + templateLabel(name),
		Detail:  rest,
		Anchor:  "template-syntax-error",
		Err:     err,
	}
	if _, isExec := errors.AsType[template.ExecError](err); isExec {
		if m := tmplExecLocRe.FindStringSubmatch(rest); m != nil {
			be.Line, _ = strconv.Atoi(m[1])
			byteCol, _ := strconv.Atoi(m[2])
			ctx := m[3]
			be.Detail = strings.TrimPrefix(rest, m[0])
			if ctx != "" {
				be.Detail += " (at <" + ctx + ">)"
			}
			be.Snippet = excerpt(src, be.Line, 0)
			if len(be.Snippet) == 1 {
				be.Col = locateInLine(be.Snippet[0].Text, ctx, byteCol)
			}
		} else {
			be.Detail = strings.TrimSpace(strings.TrimPrefix(rest, " "))
		}
		be.Hint, be.Anchor = execHint(be.Detail)
		return be
	}
	// Parse error: "LINE: message", never a column.
	rest = strings.TrimSpace(rest)
	if m := tmplParseLineRe.FindStringSubmatch(rest); m != nil {
		be.Line, _ = strconv.Atoi(m[1])
		be.Detail = strings.TrimPrefix(rest, m[0])
	}
	if m := tmplStartedAtRe.FindStringSubmatch(be.Detail); m != nil {
		// "unclosed action started at inline:1" - the start is where the reader has to look.
		be.Line, _ = strconv.Atoi(m[1])
	}
	if strings.Contains(be.Detail, "unexpected EOF") {
		// The parser only knows the template ended; point at the block that was never closed.
		if line, keyword := unclosedBlock(src); line > 0 {
			be.Line = line
			be.Detail = fmt.Sprintf("unexpected EOF: the {{%s}} started here has no {{end}}", keyword)
		}
	}
	if be.Line > 0 {
		be.Snippet = excerpt(src, be.Line, 0)
	}
	be.Hint, be.Anchor = parseHint(be.Detail)
	return be
}

var tmplBlockRe = regexp.MustCompile(`\{\{-?\s*(if|range|with|define|block|end)\b`)

// unclosedBlock returns the line and keyword of the innermost block opener without a matching
// {{end}}. A plain scan, good enough to point a reader at the right line.
func unclosedBlock(src []byte) (int, string) {
	type opener struct {
		line    int
		keyword string
	}
	var stack []opener
	for i, line := range splitLines(src) {
		for _, m := range tmplBlockRe.FindAllStringSubmatch(line, -1) {
			if m[1] == "end" {
				if len(stack) > 0 {
					stack = stack[:len(stack)-1]
				}
				continue
			}
			stack = append(stack, opener{i + 1, m[1]})
		}
	}
	if len(stack) == 0 {
		return 0, ""
	}
	top := stack[len(stack)-1]
	return top.line, top.keyword
}

func templateLabel(name string) string {
	switch name {
	case "inline":
		return "the inline template"
	case demoTemplate:
		return "the template"
	}
	return name
}

// locateInLine places the caret on the action text text/template reported (<ctx>). The reported byte
// column points at the last field of a chain (.a.b -> .b), so the context text is searched first,
// choosing the occurrence closest to the reported column.
func locateInLine(line, ctx string, byteCol int) int {
	if ctx != "" {
		best := -1
		for from := 0; from <= len(line); {
			i := strings.Index(line[from:], ctx)
			if i < 0 {
				break
			}
			pos := from + i
			if best < 0 || abs(pos-byteCol) < abs(best-byteCol) {
				best = pos
			}
			from = pos + 1
		}
		if best >= 0 {
			return byteColToRuneCol(line, best)
		}
	}
	return byteColToRuneCol(line, byteCol)
}

func abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

// parseHint explains template syntax errors.
func parseHint(detail string) (hint, anchor string) {
	switch {
	case tmplUnknownFuncRe.MatchString(detail):
		fn := tmplUnknownFuncRe.FindStringSubmatch(detail)[1]
		names := templateFunctionNames()
		if s := didYouMean(fn, names); s != "" {
			return fmt.Sprintf("there is no function %q - did you mean %q?", fn, s), "unknown-function"
		}
		return fmt.Sprintf("there is no function %q; see the function reference", fn), "unknown-function"
	case strings.Contains(detail, "bad character U+002D '-'"):
		return `keys containing "-" cannot be used with a dot; use index: {{index . "my-key"}} (XML attributes are "-name")`, "keys-with-dashes"
	case strings.Contains(detail, "unclosed action"):
		return "an action opened with {{ is missing its closing }}", "template-syntax-error"
	case strings.Contains(detail, "unexpected {{end}}"):
		return "{{end}} has no matching {{if}}, {{range}} or {{with}}", "template-syntax-error"
	case strings.Contains(detail, "unexpected EOF"):
		return "a {{if}}, {{range}} or {{with}} block is missing its {{end}}", "template-syntax-error"
	case strings.Contains(detail, "bad character"):
		return "this character is not allowed inside {{ }}; quote strings with double quotes", "template-syntax-error"
	}
	return "", "template-syntax-error"
}

// execHint explains template execution errors.
func execHint(detail string) (hint, anchor string) {
	if m := tmplMissingKeyRe.FindStringSubmatch(detail); m != nil {
		return fmt.Sprintf(`the input has no key %q here (-strict is on); check spelling and case - XML attributes start with "-" (index . "-id"); `+
			`for keys that may be absent use {{with index . %q}}...{{end}}`, m[1], m[1]), "missing-keys-and-no-value"
	}
	if m := tmplWrongTypeRe.FindStringSubmatch(detail); m != nil {
		switch {
		case strings.HasPrefix(m[2], "[]"):
			return "the value is a list; loop over it with {{range ...}}, or pick one item with (index .list 0)", "wrong-type"
		case strings.HasPrefix(m[2], "map[") || strings.HasPrefix(m[2], "mxj.Map"):
			return "the value is an object; pick one of its keys, e.g. .order.customer, or print it with toJSON", "wrong-type"
		case m[1] == "string":
			return fmt.Sprintf("the value is a %s, the function needs text; convert it first, e.g. (toString .value)", m[2]), "wrong-type"
		case m[1] == "int" || m[1] == "int64" || m[1] == "float64":
			return fmt.Sprintf("the value is a %s, the function needs a number; convert it first, e.g. (toInt .value) or (toFloat64 .value)", m[2]), "wrong-type"
		}
		return "convert the value to the expected type first (toString, toInt, toFloat64, toBool)", "wrong-type"
	}
	if m := tmplArgCountRe.FindStringSubmatch(detail); m != nil {
		return fmt.Sprintf("%s takes %s argument(s), %s were given; see the function reference", m[1], m[2], m[3]), "unknown-function"
	}
	if strings.Contains(detail, "index of untyped nil") || strings.Contains(detail, "nil pointer evaluating") {
		return `the value does not exist in the input; print the parsed input with -t '?{{toJSON .}}' to check the path`, "missing-keys-and-no-value"
	}
	if m := tmplNoFieldRe.FindStringSubmatch(detail); m != nil {
		return fmt.Sprintf("the value before %s is not an object; print the parsed input with -t '?{{toJSON .}}' to check the path", m[1]), "wrong-type"
	}
	if strings.Contains(detail, "range can't iterate over") {
		return "range needs a list or a map; XML nodes with a single child are not lists - wrap them with mustArray", "wrong-type"
	}
	if m := tmplErrCallingRe.FindStringSubmatch(detail); m != nil {
		return fmt.Sprintf("the function %s rejected its arguments; see the function reference", m[1]), "unknown-function"
	}
	return "", "template-syntax-error"
}

// templateFunctionNames lists bafi's and text/template's functions, sorted, for suggestions.
func templateFunctionNames() []string {
	names := append([]string{}, builtinTemplateFunctions...)
	for name := range templateFunctions() {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// didYouMean returns the candidate closest to word when it is plausibly a typo, else "".
func didYouMean(word string, candidates []string) string {
	best, bestDist := "", -1
	for _, c := range candidates {
		d := editDistance(strings.ToLower(word), strings.ToLower(c))
		if bestDist < 0 || d < bestDist {
			best, bestDist = c, d
		}
	}
	limit := 2
	if len(word) <= 3 {
		limit = 1
	}
	if bestDist >= 0 && bestDist <= limit && best != word {
		return best
	}
	return ""
}

// editDistance is the optimal-string-alignment distance: insertions, deletions, substitutions and
// swaps of two adjacent letters ("xlm" -> "xml") each cost 1.
func editDistance(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	d := make([][]int, len(ra)+1)
	for i := range d {
		d[i] = make([]int, len(rb)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(ra); i++ {
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			d[i][j] = min3(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && ra[i-1] == rb[j-2] && ra[i-2] == rb[j-1] && d[i-2][j-2]+1 < d[i][j] {
				d[i][j] = d[i-2][j-2] + 1
			}
		}
	}
	return d[len(ra)][len(rb)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// usageError builds an exit-code-2 error for a wrong invocation.
func usageError(summary, hint, anchor string) error {
	return &bafiError{Stage: stageUsage, Summary: summary, Hint: hint, Anchor: anchor}
}
