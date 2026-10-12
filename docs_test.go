//go:build !(js && wasm)

package main

// Executable documentation: every example marked in docs/**/*.md is run here, so the docs cannot
// drift from what bafi actually does. Markers are HTML comments directly before a fenced block
// (invisible in the rendered site):
//
//	<!-- test:file NAME [crlf] --> the block is written to NAME in the page's temp working directory
//	                               (crlf: with CRLF line endings, e.g. for MT940)
//	<!-- test:cmd [quiet] -->      the block is one command line: [echo '...' | cat FILE |] bafi ARGS [< FILE];
//	                               quiet: must succeed silently (e.g. it writes -o), no result block
//	<!-- test:output -->           expected stdout of the preceding command
//	<!-- test:stderr exit=N -->    expected stderr and exit code of the preceding command
//	<!-- test:table -->            before a markdown table: Function | Example | Result; each
//	                               Example is rendered and must equal Result ("*" skips a row)
//
// Run with BAFI_DOCS_UPDATE=1 to rewrite expected output/stderr blocks from the actual results
// (review the diff before committing).

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
)

var markerRe = regexp.MustCompile(`^<!--\s*test:(\w+)\s*(.*?)\s*-->\s*$`)

type docBlock struct {
	kind    string // file, cmd, output, stderr, table
	arg     string
	content string
	start   int // first content line (0-based) in the file
	end     int // line after the last content line (the closing fence)
	line    int // 1-based line of the marker, for messages
}

type docCase struct {
	files    []docBlock
	cmd      *docBlock
	expected *docBlock
}

func TestDocs(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var pages []string
	err = filepath.WalkDir(filepath.Join(root, "docs"), func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".md") && d.Type()&fs.ModeSymlink == 0 {
			pages = append(pages, path)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	update := os.Getenv("BAFI_DOCS_UPDATE") != ""
	for _, page := range pages {
		rel, _ := filepath.Rel(root, page)
		t.Run(filepath.ToSlash(rel), func(t *testing.T) {
			src, err := os.ReadFile(page)
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(string(src), "\n")
			blocks, err := parseDocBlocks(lines)
			if err != nil {
				t.Fatal(err)
			}
			cases, tables, err := groupDocCases(blocks)
			if err != nil {
				t.Fatal(err)
			}
			isRecipe := strings.Contains(filepath.ToSlash(rel), "docs/recipes/") && filepath.Base(page) != "index.md"
			if isRecipe && len(cases) == 0 {
				t.Fatal("recipe pages need at least one tested example (<!-- test:cmd --> + <!-- test:output -->)")
			}
			// One working directory per page: later steps of a tutorial can use files earlier steps wrote.
			dir := t.TempDir()
			for _, sample := range []string{"testdata.xml", "template.tmpl", "lua/functions.lua", "lua/json.lua"} {
				copyFile(t, filepath.Join(root, sample), filepath.Join(dir, sample))
			}
			defer resetLuaState()
			replacements := map[int]string{}
			for _, c := range cases {
				got, ok := runDocCase(t, root, dir, c, update)
				if !ok && update {
					replacements[c.expected.start] = got
				}
			}
			for _, tbl := range tables {
				for _, problem := range checkFunctionTable(tbl) {
					t.Error(problem)
				}
			}
			if update && len(replacements) > 0 {
				writeDocUpdates(t, page, lines, blocks, replacements)
			}
		})
	}
}

// parseDocBlocks finds every marker and the content that follows it.
func parseDocBlocks(lines []string) ([]docBlock, error) {
	var blocks []docBlock
	for i := 0; i < len(lines); i++ {
		m := markerRe.FindStringSubmatch(strings.TrimSpace(lines[i]))
		if m == nil {
			continue
		}
		b := docBlock{kind: m[1], arg: m[2], line: i + 1}
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if b.kind == "table" {
			b.start = j
			for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), "|") {
				j++
			}
			b.end = j
			b.content = strings.Join(lines[b.start:b.end], "\n")
			if b.start == b.end {
				return nil, fmt.Errorf("line %d: test:table is not followed by a table", b.line)
			}
			blocks = append(blocks, b)
			i = j - 1
			continue
		}
		if j >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[j]), "```") {
			return nil, fmt.Errorf("line %d: marker test:%s is not followed by a fenced code block", b.line, b.kind)
		}
		fence := strings.TrimSpace(lines[j])
		fence = fence[:len(fence)-len(strings.TrimLeft(fence, "`"))]
		k := j + 1
		for k < len(lines) && strings.TrimSpace(lines[k]) != fence {
			k++
		}
		if k >= len(lines) {
			return nil, fmt.Errorf("line %d: fenced block is not closed", b.line)
		}
		b.start, b.end = j+1, k
		b.content = strings.Join(lines[j+1:k], "\n")
		blocks = append(blocks, b)
		i = k
	}
	return blocks, nil
}

// groupDocCases pairs files and a command with the expected result that follows them.
func groupDocCases(blocks []docBlock) ([]docCase, []docBlock, error) {
	var cases []docCase
	var tables []docBlock
	var cur docCase
	for i := range blocks {
		b := blocks[i]
		switch b.kind {
		case "file":
			if b.arg == "" {
				return nil, nil, fmt.Errorf("line %d: test:file needs a file name", b.line)
			}
			cur.files = append(cur.files, b)
		case "cmd":
			if cur.cmd != nil {
				return nil, nil, fmt.Errorf("line %d: test:cmd without test:output/test:stderr for the command on line %d", b.line, cur.cmd.line)
			}
			if b.arg == "quiet" {
				cases = append(cases, docCase{files: cur.files, cmd: &blocks[i]})
				cur = docCase{}
				continue
			}
			cur.cmd = &blocks[i]
		case "output", "stderr":
			if cur.cmd == nil {
				return nil, nil, fmt.Errorf("line %d: test:%s without a preceding test:cmd", b.line, b.kind)
			}
			cur.expected = &blocks[i]
			cases = append(cases, cur)
			cur = docCase{}
		case "table":
			tables = append(tables, b)
		default:
			return nil, nil, fmt.Errorf("line %d: unknown marker test:%s", b.line, b.kind)
		}
	}
	if cur.cmd != nil {
		return nil, nil, fmt.Errorf("line %d: test:cmd has no test:output or test:stderr", cur.cmd.line)
	}
	return cases, tables, nil
}

// runDocCase writes the case's files into dir and runs its command there. It returns the actual result
// and whether it matched.
func runDocCase(t *testing.T, root, dir string, c docCase, update bool) (string, bool) {
	t.Helper()
	for _, f := range c.files {
		fields := strings.Fields(f.arg)
		content := f.content + "\n"
		if len(fields) > 1 && fields[1] == "crlf" {
			content = strings.ReplaceAll(content, "\n", "\r\n")
		}
		path := filepath.Join(dir, filepath.FromSlash(fields[0]))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(fields[0], "lua/") {
			resetLuaState() // a page that ships its own functions.lua must not see an earlier one
		}
	}
	args, stdin, err := parseDocCommand(c.cmd.content, dir)
	if err != nil {
		t.Errorf("line %d: %v", c.cmd.line, err)
		return "", true
	}
	t.Chdir(dir)
	var stdout, stderr bytes.Buffer
	code := run(args, stdin, &stdout, &stderr)
	t.Chdir(root)

	if c.expected == nil { // quiet
		if code != 0 || stdout.Len() > 0 || stderr.Len() > 0 {
			t.Errorf("line %d: quiet command %q: exit %d, stdout %q, stderr %q", c.cmd.line, c.cmd.content, code, stdout.String(), stderr.String())
		}
		return "", true
	}
	want := strings.TrimRight(c.expected.content, "\n")
	switch c.expected.kind {
	case "output":
		got := strings.TrimRight(stdout.String(), "\n")
		if code != 0 {
			t.Errorf("line %d: %q exited %d:\n%s", c.cmd.line, c.cmd.content, code, stderr.String())
			return got, true
		}
		if got != want {
			if !update {
				t.Errorf("line %d: output of %q differs\n--- got ---\n%s\n--- want ---\n%s", c.cmd.line, c.cmd.content, got, want)
			}
			return got, false
		}
	case "stderr":
		got := strings.TrimRight(stderr.String(), "\n")
		wantCode := 1
		if m := regexp.MustCompile(`exit=(\d+)`).FindStringSubmatch(c.expected.arg); m != nil {
			wantCode, _ = strconv.Atoi(m[1])
		}
		if code != wantCode {
			t.Errorf("line %d: %q exited %d, want %d\n%s", c.cmd.line, c.cmd.content, code, wantCode, stderr.String())
		}
		if got != want {
			if !update {
				t.Errorf("line %d: stderr of %q differs\n--- got ---\n%s\n--- want ---\n%s", c.cmd.line, c.cmd.content, got, want)
			}
			return got, false
		}
	}
	return "", true
}

// parseDocCommand understands `bafi ARGS`, `echo '...' | bafi ARGS`, `cat FILE | bafi ARGS` and
// `bafi ARGS < FILE`.
func parseDocCommand(cmd, dir string) ([]string, io.Reader, error) {
	cmd = strings.TrimSpace(cmd)
	if strings.Contains(cmd, "\n") {
		return nil, nil, fmt.Errorf("test:cmd must be a single line: %q", cmd)
	}
	var stdin io.Reader
	if i := strings.Index(cmd, "| bafi "); i >= 0 {
		left, err := splitShellWords(strings.TrimSpace(cmd[:i]))
		if err != nil {
			return nil, nil, err
		}
		switch {
		case len(left) == 2 && left[0] == "echo":
			stdin = strings.NewReader(left[1] + "\n")
		case len(left) == 2 && left[0] == "cat":
			data, err := os.ReadFile(filepath.Join(dir, left[1]))
			if err != nil {
				return nil, nil, err
			}
			stdin = bytes.NewReader(data)
		default:
			return nil, nil, fmt.Errorf("only `echo '...' | bafi` and `cat FILE | bafi` pipes are supported: %q", cmd)
		}
		cmd = cmd[i+2:]
	}
	if i := strings.LastIndex(cmd, " < "); i >= 0 && stdin == nil {
		if target, err := splitShellWords(cmd[i+3:]); err == nil && len(target) == 1 && !strings.ContainsAny(cmd[i+3:], `'"{}`) {
			data, err := os.ReadFile(filepath.Join(dir, target[0]))
			if err != nil {
				return nil, nil, err
			}
			stdin = bytes.NewReader(data)
			cmd = cmd[:i]
		}
	}
	words, err := splitShellWords(cmd)
	if err != nil {
		return nil, nil, err
	}
	if len(words) == 0 || (words[0] != "bafi" && words[0] != "./bafi") {
		return nil, nil, fmt.Errorf("test:cmd must start with bafi: %q", cmd)
	}
	return words[1:], stdin, nil
}

// splitShellWords splits like a POSIX shell for the subset used in docs: single quotes are literal,
// double quotes allow \" and \\, a backslash outside quotes escapes the next character.
func splitShellWords(s string) ([]string, error) {
	var words []string
	var cur strings.Builder
	inWord := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '\'':
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return nil, fmt.Errorf("unclosed single quote in %q", s)
			}
			cur.WriteString(s[i+1 : i+1+end])
			i += end + 1
			inWord = true
		case c == '"':
			i++
			for ; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' && i+1 < len(s) && (s[i+1] == '"' || s[i+1] == '\\') {
					i++
				}
				cur.WriteByte(s[i])
			}
			if i >= len(s) {
				return nil, fmt.Errorf("unclosed double quote in %q", s)
			}
			inWord = true
		case c == '\\' && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
			inWord = true
		case c == ' ' || c == '\t':
			if inWord {
				words = append(words, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	if inWord {
		words = append(words, cur.String())
	}
	return words, nil
}

var tableCodeRe = regexp.MustCompile("`([^`]*)`")

// checkFunctionTable renders every Example cell of a Function | Example | Result table and returns
// one message per mismatch.
func checkFunctionTable(tbl docBlock) []string {
	var problems []string
	rows := strings.Split(tbl.content, "\n")
	checked := 0
	for n, row := range rows {
		cells := strings.Split(strings.Trim(strings.TrimSpace(row), "|"), " | ")
		if n < 2 || len(cells) < 3 { // header and separator
			continue
		}
		line := tbl.start + n + 1
		example := tableCodeRe.FindStringSubmatch(cells[1])
		result := strings.TrimSpace(cells[2])
		if example == nil {
			problems = append(problems, fmt.Sprintf("table line %d: Example must be a `code span`: %s", line, row))
			continue
		}
		if result == "*" {
			continue
		}
		want := tableCodeRe.FindStringSubmatch(result)
		if want == nil {
			problems = append(problems, fmt.Sprintf("table line %d: Result must be a `code span` or *: %s", line, row))
			continue
		}
		tmpl := strings.ReplaceAll(example[1], `\|`, "|")
		var out bytes.Buffer
		if _, err := render(&out, nil, []byte(tmpl), "inline", false); err != nil {
			problems = append(problems, fmt.Sprintf("table line %d: %s: %v", line, tmpl, err))
			continue
		}
		if got := strings.TrimRight(out.String(), "\n"); got != strings.ReplaceAll(want[1], `\|`, "|") {
			problems = append(problems, fmt.Sprintf("table line %d: %s = %q, documented %q", line, tmpl, got, want[1]))
		}
		checked++
	}
	if checked == 0 {
		problems = append(problems, fmt.Sprintf("table at line %d checked no rows", tbl.line))
	}
	return problems
}

func TestFunctionTableChecker(t *testing.T) {
	tbl := docBlock{line: 1, content: strings.Join([]string{
		"| Function | Example | Result |",
		"|---|---|---|",
		"| ok | `{{add1 41}}` | `42` |",
		"| wrong | `{{add1 41}}` | `43` |",
		"| broken | `{{nope}}` | `x` |",
		"| skipped | `{{uuid}}` | * |",
	}, "\n")}
	problems := checkFunctionTable(tbl)
	if len(problems) != 2 || !strings.Contains(problems[0], `documented "43"`) || !strings.Contains(problems[1], "nope") {
		t.Errorf("checker must report exactly the wrong and the broken row, got %q", problems)
	}
}

// writeDocUpdates replaces expected blocks with actual results (BAFI_DOCS_UPDATE=1).
func writeDocUpdates(t *testing.T, page string, lines []string, blocks []docBlock, replacements map[int]string) {
	t.Helper()
	var out []string
	pos := 0
	for _, b := range blocks {
		repl, ok := replacements[b.start]
		if !ok {
			continue
		}
		out = append(out, lines[pos:b.start]...)
		out = append(out, strings.Split(repl, "\n")...)
		pos = b.end
	}
	out = append(out, lines[pos:]...)
	if err := os.WriteFile(page, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("updated %d block(s) in %s", len(replacements), page)
}

// resetLuaState drops the process-wide Lua state so the next {{lua}} call loads ./lua/functions.lua
// from the current directory again. The CLI loads it once per process; tests need a fresh one per page.
func resetLuaState() {
	if luaData != nil {
		luaData.Close()
	}
	luaData, luaLoadErr, luaLoadOnce = nil, nil, sync.Once{}
}

func copyFile(t *testing.T, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestSplitShellWords(t *testing.T) {
	got, err := splitShellWords(`bafi -i a.json -t '?{{index . "my key"}}' -d ";" x\ y`)
	want := []string{"bafi", "-i", "a.json", "-t", `?{{index . "my key"}}`, "-d", ";", "x y"}
	if err != nil || strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("got %q, %v", got, err)
	}
}

// TestTroubleshootingAnchors: every docs link printed by an error must land on a real heading.
func TestTroubleshootingAnchors(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("docs", "troubleshooting.md"))
	if err != nil {
		t.Fatal(err)
	}
	headings := map[string]bool{}
	for _, line := range strings.Split(string(src), "\n") {
		if strings.HasPrefix(line, "#") {
			headings[mkdocsSlug(strings.TrimSpace(strings.TrimLeft(line, "#")))] = true
		}
	}
	anchors := collectAnchors(t, "errors.go", "input.go", "main.go")
	for anchor, pos := range anchors {
		if !headings[anchor] {
			t.Errorf("%s: anchor %q has no heading in docs/troubleshooting.md", pos, anchor)
		}
	}
	if len(anchors) < 15 {
		t.Errorf("found only %d anchors - did the way errors set Anchor change?", len(anchors))
	}
}

// collectAnchors finds every troubleshooting anchor literal in the given files: `Anchor: "x"` fields,
// the last argument of usageError(...), and the second result of the (hint, anchor) helpers.
func collectAnchors(t *testing.T, files ...string) map[string]string {
	t.Helper()
	anchors := map[string]string{}
	fset := token.NewFileSet()
	add := func(e ast.Expr) {
		if lit, ok := e.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			if s, err := strconv.Unquote(lit.Value); err == nil && s != "" {
				anchors[s] = fset.Position(lit.Pos()).String()
			}
		}
	}
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.KeyValueExpr:
				if id, ok := n.Key.(*ast.Ident); ok && id.Name == "Anchor" {
					add(n.Value)
				}
			case *ast.CallExpr:
				if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "usageError" && len(n.Args) == 3 {
					add(n.Args[2])
				}
			case *ast.FuncDecl:
				if n.Name.Name == "parseHint" || n.Name.Name == "execHint" {
					ast.Inspect(n.Body, func(m ast.Node) bool {
						if ret, ok := m.(*ast.ReturnStmt); ok && len(ret.Results) == 2 {
							add(ret.Results[1])
						}
						return true
					})
				}
			}
			return true
		})
	}
	return anchors
}

// mkdocsSlug mirrors Python-Markdown's toc slugify for plain ASCII headings.
func mkdocsSlug(heading string) string {
	heading = strings.ToLower(heading)
	heading = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(heading, "")
	return regexp.MustCompile(`[-\s]+`).ReplaceAllString(strings.TrimSpace(heading), "-")
}
