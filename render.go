package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"
)

// render executes tmpl against data into w. strict turns a missing map key into an error instead of
// "<no value>". It returns how many times "<no value>" was written, so callers can warn about it.
// Errors are *bafiError values pointing at the template line.
func render(w io.Writer, data interface{}, tmpl []byte, name string, strict bool) (int, error) {
	t := template.New(name).Funcs(templateFunctions())
	if strict {
		t = t.Option("missingkey=error")
	}
	if _, err := t.Parse(string(tmpl)); err != nil {
		return 0, templateError(err, name, tmpl)
	}
	counter := &noValueCounter{w: w}
	if err := t.Execute(counter, data); err != nil {
		return counter.n, templateError(err, name, tmpl)
	}
	return counter.n, nil
}

// demoSource and demoTemplate name the online demo's input and template boxes in error messages.
const (
	demoSource   = "the input box"
	demoTemplate = "demo"
)

// renderString parses input in the given format and renders tmpl, returning the output (preceded by any
// warnings) or the formatted error. It is what the online demo calls; the CLI uses run().
func renderString(input, tmpl, format string) string {
	var warnings []string
	warn := func(msg string) { warnings = append(warnings, "bafi: warning: "+msg) }
	if format != "" && !knownFormat(format) {
		return formatError(unknownFormatError(format))
	}
	data, err := mapInputData(cleanBOM([]byte(input)), inputSpec{Source: demoSource, Format: format}, warn)
	if err != nil {
		return formatError(err)
	}
	var out bytes.Buffer
	noValues, err := render(&out, data, []byte(tmpl), demoTemplate, false)
	if err != nil {
		return formatError(err)
	}
	if noValues > 0 {
		warn(fmt.Sprintf("output contains <no value> %d time(s) - a key used in the template is missing from the input (or is null)", noValues))
	}
	if len(warnings) > 0 {
		return strings.Join(warnings, "\n") + "\n\n" + out.String()
	}
	return out.String()
}

// noValueMarker is what text/template prints for a missing key or a nil value.
var noValueMarker = []byte("<no value>")

// noValueCounter passes writes through and counts noValueMarker occurrences, including ones split
// across two writes. The marker's first byte appears only once in it, so the carried tail can never
// be counted twice.
type noValueCounter struct {
	w    io.Writer
	tail []byte
	n    int
}

func (c *noValueCounter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	buf := append(c.tail, p[:n]...)
	c.n += bytes.Count(buf, noValueMarker)
	keep := len(noValueMarker) - 1
	if len(buf) < keep {
		keep = len(buf)
	}
	c.tail = append(c.tail[:0], buf[len(buf)-keep:]...)
	return n, err
}
