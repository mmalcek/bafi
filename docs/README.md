# BaFi

**Convert JSON, XML, CSV, YAML, BSON and MT940 into any text format with templates.**

bafi reads structured data from a file or a pipe and renders it through a template into whatever text
the next system needs: JSON, XML, CSV, HTML, SQL, a SOAP request. One small program without
dependencies for Windows, macOS and Linux.

[Install](install.md) · [Recipes](recipes/index.md) · [Function reference](functions.md) · [Troubleshooting](troubleshooting.md) · [GitHub](https://github.com/mmalcek/bafi)

<img src="img/scheme.svg" style="border: 0;" height="150px" alt="input formats to any output" />

## What do you want to do?

| I want to... | Recipe |
|---|---|
| turn XML into JSON | [Convert XML to JSON](recipes/xml-to-json.md) |
| turn JSON (e.g. a REST response) into XML | [Convert JSON to XML](recipes/json-to-xml.md) |
| turn a CSV export into JSON | [Convert CSV to JSON](recipes/csv-to-json.md) |
| get JSON or XML into Excel | [Convert JSON to CSV](recipes/json-to-csv.md) · [XML to CSV](recipes/xml-to-csv.md) |
| call a SOAP service with data from a REST API | [REST response to SOAP request](recipes/rest-to-soap.md) |
| give a SOAP response to a REST client | [SOAP response to JSON](recipes/soap-to-json.md) |
| produce an HTML report | [HTML report from JSON](recipes/json-to-html.md) |
| load a CSV into a database | [SQL INSERT statements from CSV](recipes/csv-to-sql.md) |
| process MT940 bank statements | [MT940 to CSV](recipes/mt940-to-csv.md) |
| combine several files into one document | [Merge several files](recipes/merge-files.md) |
| convert YAML, JSON, TOON, BSON | [Convert between formats](recipes/convert-formats.md) |

[All recipes](recipes/index.md)

## In thirty seconds

<!-- test:file users.json -->
```json
{"users": [{"name": "Alice", "city": "Prague"}, {"name": "Bob", "city": "Brno"}]}
```

Convert the whole document with one function:

<!-- test:cmd -->
```sh
bafi -i users.json -t '?{{toYAML .}}'
```

<!-- test:output -->
```yaml
users:
    - city: Prague
      name: Alice
    - city: Brno
      name: Bob
```

Or describe the output yourself. Text outside `{{ }}` is copied as it is; `range` repeats a block for
every item:

<!-- test:cmd -->
```sh
bafi -i users.json -t '?{{range .users}}{{.name}} lives in {{upper .city}}{{"\n"}}{{end}}'
```

<!-- test:output -->
```
Alice lives in PRAGUE
Bob lives in BRNO
```

Works in pipes too: `curl -fs https://example.com/api/users | bafi -f json -t report.tmpl -o report.html`.
Next: the [template guide](templates.md) and the [command line](cli.md).

## Try it online

Runs entirely in your browser; nothing is uploaded. The demo (a few megabytes) loads on the first click.

<div class="bafi-demo">
<p>
<button type="button" onclick="bafiExample('json')">JSON to text</button>
<button type="button" onclick="bafiExample('csv')">CSV to JSON</button>
<button type="button" onclick="bafiExample('xml')">XML to CSV</button>
</p>
<p>Input format:
<select id="bafiFormat"><option value="json">JSON</option><option value="xml">XML</option><option value="yaml">YAML</option><option value="csv">CSV</option></select>
</p>
<textarea id="bafiInput" rows="12" cols="44" spellcheck="false" aria-label="input data"></textarea>
<textarea id="bafiTemplate" rows="12" cols="44" spellcheck="false" aria-label="template"></textarea>
<p><button type="button" onclick="bafiRun()"><b>Run</b></button></p>
<pre id="bafiOutput" style="max-width: 720px; min-height: 120px; white-space: pre-wrap;"></pre>
</div>

<script src="js/wasm_exec.js"></script>
<script>
var bafiExamples = {
  json: {format: "json",
    input: '{\n  "user": {\n    "name": "John Doe",\n    "age": 25,\n    "address": {"city": "New York", "state": "NY"},\n    "favourite_colors": ["red", "green", "blue"]\n  }\n}',
    template: 'Hello {{upper .user.name}},\n\nyou are {{.user.age}} years old\nand live in {{.user.address.city}}, {{.user.address.state}}.\nYour favourite colors are:\n{{range .user.favourite_colors}} - {{.}}\n{{end}}'},
  csv: {format: "csv",
    input: 'id,name,city\n1,Alice,Prague\n2,"Brown, Bob",Brno',
    template: '{{toJSON .}}'},
  xml: {format: "xml",
    input: '<orders>\n  <order id="1001"><customer>Alice</customer><total>120.50</total></order>\n  <order id="1002"><customer>Bob</customer><total>80.00</total></order>\n</orders>',
    template: 'id,customer,total\n{{- range mustArray .orders.order}}\n{{index . "-id"}},{{.customer}},{{.total}}\n{{- end}}'}
};
function bafiExample(name) {
  var e = bafiExamples[name];
  document.getElementById("bafiFormat").value = e.format;
  document.getElementById("bafiInput").value = e.input;
  document.getElementById("bafiTemplate").value = e.template;
  document.getElementById("bafiOutput").textContent = "";
}
var bafiLoading = null;
function bafiLoad() {
  if (!bafiLoading) {
    var go = new Go();
    bafiLoading = WebAssembly.instantiateStreaming(fetch("js/bafi.wasm"), go.importObject)
      .then(function (result) { go.run(result.instance); });
  }
  return bafiLoading;
}
function bafiRun() {
  var out = document.getElementById("bafiOutput");
  if (typeof bafiRender !== "function") { out.textContent = "Loading the demo..."; }
  bafiLoad().then(function () {
    out.textContent = bafiRender(document.getElementById("bafiInput").value,
      document.getElementById("bafiTemplate").value,
      document.getElementById("bafiFormat").value);
  }).catch(function (err) {
    bafiLoading = null;
    out.textContent = "The demo could not be loaded: " + err;
  });
}
bafiExample("json");
</script>

## Why bafi

- **Many inputs:** JSON, BSON, YAML, CSV (any delimiter), XML and MT940 bank statements, from a file
  or standard input.
- **Any output:** the template decides. HTML, SQL, shell scripts, SOAP envelopes, fixed-width text.
- **Pipes:** read from `curl`, write to the next command. Bridges REST and SOAP services, databases
  and spreadsheets.
- **Several inputs at once:** merge files of different formats into one output.
- **Helpful errors:** problems are reported with the file, line and column, and a hint how to fix them.
- **Extensible:** 60+ template functions, plus your own in [Lua](recipes/lua.md).
- **TOON and OpenAI:** encode data compactly for LLM prompts, or [ask OpenAI](recipes/openai.md) about
  your data (experimental).

[![Go](https://github.com/mmalcek/bafi/actions/workflows/go.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/go.yml)
[![CodeQL](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/mmalcek/bafi/actions/workflows/codeql-analysis.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/mmalcek/bafi)](https://goreportcard.com/report/github.com/mmalcek/bafi)
[![License](https://img.shields.io/github/license/mmalcek/bafi)](https://github.com/mmalcek/bafi/blob/main/LICENSE)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge.svg)](https://github.com/avelino/awesome-go#text-processing)
[![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/mmalcek/bafi?label=latest%20release)](https://github.com/mmalcek/bafi/releases/latest)

**If you like this app you can buy me a coffee ;)**

<a href='https://ko-fi.com/mmalcek' target='_blank'>
	<img height='20' style='border:0;height:40px;' src='https://storage.ko-fi.com/cdn/brandasset/kofi_button_blue.png' alt='Buy Me a Coffee at ko-fi.com' />
</a>
