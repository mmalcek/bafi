# Troubleshooting

Every bafi error names the file, the line and usually the column, shows the offending line, and ends
with a hint and a link to the matching section below. The examples on this page are real output of the
current version.

Exit codes: `0` success, `1` the input, template or output failed, `2` the command line is wrong.

## Command line

### No input

<!-- test:cmd -->
```sh
bafi -t '?{{toJSON .}}'
```

<!-- test:stderr exit=2 -->
```
bafi: no input

hint: read a file with -i data.json, or pipe data in: cat data.json | bafi -f json -t '?{{toXML .}}'
docs: https://mmalcek.github.io/bafi/troubleshooting/#no-input

usage: bafi -i INPUT -t TEMPLATE [-o OUTPUT]   (bafi -h for help and examples)
```

Without `-i`, bafi reads standard input, and nothing was piped in. Give a file with `-i data.json`, or
pipe data: `cat data.json | bafi -f json -t ...`. Redirecting works as well: `bafi -f json -t ... < data.json`.

### Unexpected argument

The shell splits an inline template at spaces unless it is quoted:

<!-- test:file data.json -->
```json
{"name": "Alice", "count": 3}
```

<!-- test:cmd -->
```sh
bafi -i data.json -t ?{{toJSON .}}
```

<!-- test:stderr exit=2 -->
```
bafi: unexpected argument ".}}"

hint: flags must come first and inline templates with spaces must be quoted: -t '?{{toJSON .}}'
docs: https://mmalcek.github.io/bafi/troubleshooting/#unexpected-argument

usage: bafi -i INPUT -t TEMPLATE [-o OUTPUT]   (bafi -h for help and examples)
```

Quote inline templates: single quotes in bash/zsh (`-t '?{{toJSON .}}'`), double quotes in PowerShell
and cmd.exe (`-t "?{{toJSON .}}"`). All flags must come before any other argument. The output above is
from bash; zsh, the macOS default shell, stops even earlier with `zsh: no matches found`, because it
treats the unquoted `?` as a wildcard. The fix is the same: quote the template.

### Unknown input format

<!-- test:cmd -->
```sh
echo '{"a": 1}' | bafi -t '?{{.a}}'
```

<!-- test:stderr exit=2 -->
```
bafi: cannot tell the input format of stdin

hint: add -f json (the input looks like JSON); supported formats: json, bson, yaml, csv, xml, mt940
docs: https://mmalcek.github.io/bafi/troubleshooting/#unknown-input-format

usage: bafi -i INPUT -t TEMPLATE [-o OUTPUT]   (bafi -h for help and examples)
```

Standard input has no file name, so bafi cannot guess the format: add `-f json`, `-f xml`, `-f csv`,
`-f yaml`, `-f bson` or `-f mt940`. For files, the extension decides (`.json`, `.xml`, `.csv`, `.yaml`/`.yml`,
`.bson`, `.sta`); `-f` overrides it. A misspelled `-f` value gets a suggestion.

### File not found

<!-- test:cmd -->
```sh
bafi -i data.jsn -t '?{{.name}}'
```

<!-- test:stderr -->
```
bafi: cannot read input file "data.jsn"
  no such file or directory

hint: check the path; relative paths are resolved from the current directory
docs: https://mmalcek.github.io/bafi/troubleshooting/#file-not-found
```

Relative paths start in the current directory, not where bafi is installed. The same error for `-t`
usually means the inline template is missing its leading `?`.

### Cannot create output file

<!-- test:cmd -->
```sh
bafi -i data.json -t '?{{.name}}' -o reports/today.txt
```

<!-- test:stderr -->
```
bafi: cannot create output file "reports/today.txt"
  no such file or directory

hint: the directory does not exist; create it first
docs: https://mmalcek.github.io/bafi/troubleshooting/#cannot-create-output-file
```

bafi does not create directories; create `reports` first.

## Input data

### Input is empty

<!-- test:cmd -->
```sh
echo '' | bafi -f json -t '?{{.name}}'
```

<!-- test:stderr -->
```
bafi: JSON input from stdin is empty

hint: nothing arrived on stdin - did the previous command in the pipe fail? (with curl use -f so HTTP errors are reported)
docs: https://mmalcek.github.io/bafi/troubleshooting/#input-is-empty
```

Usually the command before bafi failed and printed its error to the terminal instead of the pipe. With
curl, add `-f` so HTTP errors are reported, and `-s` to hide the progress bar: `curl -fs URL | bafi ...`.

### Invalid JSON

<!-- test:file broken.json -->
```json
{
  "name": "Alice",
  "tags": ["a", "b",],
}
```

<!-- test:cmd -->
```sh
bafi -i broken.json -t '?{{.name}}'
```

<!-- test:stderr -->
```
bafi: cannot parse JSON input from "broken.json" (line 3, column 21)

  3 |   "tags": ["a", "b",],
    |                     ^
  invalid character ']' looking for beginning of value

hint: remove the trailing comma before ] - JSON does not allow it
docs: https://mmalcek.github.io/bafi/troubleshooting/#invalid-json
```

The most common causes: a trailing comma, single quotes instead of double quotes, keys or text values
without quotes, or a file cut off in the middle. bafi expects an object `{...}` or a list of objects
`[{...}, ...]` at the top level.

### Invalid YAML

<!-- test:file broken.yaml -->
```yaml
name: Alice
 role: admin
```

<!-- test:cmd -->
```sh
bafi -i broken.yaml -t '?{{.name}}'
```

<!-- test:stderr -->
```
bafi: cannot parse YAML input from "broken.yaml" (near line 2)

  1 | name: Alice
  2 |  role: admin
  mapping values are not allowed in this context

hint: check the indentation; a value containing ": " must be quoted, e.g. title: "Note: read me"
docs: https://mmalcek.github.io/bafi/troubleshooting/#invalid-yaml
```

YAML reports problems at or just before the line where it got confused, so check the line above as well.
Indent with spaces only; quote values that contain `: ` or start with special characters.

### Invalid XML

<!-- test:file broken.xml -->
```xml
<order>
  <customer>Tom & Jerry</customer>
</order>
```

<!-- test:cmd -->
```sh
bafi -i broken.xml -t '?{{.order.customer}}'
```

<!-- test:stderr -->
```
bafi: cannot parse XML input from "broken.xml" (line 2, column 17)

  2 |   <customer>Tom & Jerry</customer>
    |                 ^
  XML syntax error on line 2: invalid character entity & (no semicolon)

hint: a bare & must be written as &amp; (and < as &lt;) inside XML text
docs: https://mmalcek.github.io/bafi/troubleshooting/#invalid-xml
```

Inside XML text `&` must be written `&amp;` and `<` must be `&lt;`. Other frequent causes are a closing
tag that does not match (`<b>...</c>`) and a document that was cut off. Only UTF-8 XML is supported.

### Invalid CSV

<!-- test:file quotes.csv -->
```
name,comment
Alice,said "hi"
```

<!-- test:cmd -->
```sh
bafi -i quotes.csv -t '?{{len .}}'
```

<!-- test:stderr -->
```
bafi: cannot parse CSV input from "quotes.csv" (line 2, column 12)

  2 | Alice,said "hi"
    |            ^
  bare " in non-quoted-field

hint: a quote inside an unquoted field: quote the whole field and double the inner quotes, e.g. "He said ""hi"""
docs: https://mmalcek.github.io/bafi/troubleshooting/#invalid-csv
```

A field that contains quotes must itself be quoted, with the inner quotes doubled:
`Alice,"said ""hi"""`.

### CSV: wrong number of fields

<!-- test:file fields.csv -->
```
name,city
Alice,Prague
Bob,Brno,CZ
```

<!-- test:cmd -->
```sh
bafi -i fields.csv -t '?{{len .}}'
```

<!-- test:stderr -->
```
bafi: cannot parse CSV input from "fields.csv" (line 3)

  3 | Bob,Brno,CZ
  wrong number of fields: the header has 2, this record has 3

hint: every row needs as many fields as the header; quote values that contain the delimiter ("Smith, John"), or set the delimiter with -d
docs: https://mmalcek.github.io/bafi/troubleshooting/#csv-wrong-number-of-fields
```

Every row needs as many values as the header line. Values containing the delimiter must be quoted
(`"Brno, CZ"`).

### CSV in one column

<!-- test:file semicolon.csv -->
```
name;city
Alice;Prague
```

<!-- test:cmd -->
```sh
bafi -i semicolon.csv -t '?{{range .}}{{.name}}{{end}}'
```

<!-- test:stderr exit=0 -->
```
bafi: warning: the CSV header has a single column containing ';' - did you mean -d ';'?
bafi: warning: output contains <no value> 1 time(s) - a key used in the template is missing from the input (or is null); run with -strict to see which one
```

The file uses `;`, but bafi split at commas, so the whole line became one column named `name;city`.
Set the delimiter: `-d ';'`, or `-d 0x09` for TAB-separated files.

### Invalid file list

<!-- test:file files.yaml -->
```yaml
- file: data.json
  label: DATA
- file: 42
  label: OTHER
```

<!-- test:cmd -->
```sh
bafi -i '?files.yaml' -t '?{{toJSON .}}'
```

<!-- test:stderr -->
```
bafi: invalid entry in file list "files.yaml"
  fileList[1]: "file" must be a string (got int)

hint: every entry needs file and label as text, format is optional: - file: rates.xml / label: RATES
docs: https://mmalcek.github.io/bafi/troubleshooting/#invalid-file-list
```

Each entry of a [file list](recipes/merge-files.md) needs `file` and `label` as text; `format` is
optional when the extension is known.

### Invalid input

Errors in BSON and MT940 input have no line numbers.

<!-- test:file statement.sta -->
```
this is not a SWIFT message
```

<!-- test:cmd -->
```sh
bafi -i statement.sta -t '?{{.Fields.F_20}}'
```

<!-- test:stderr -->
```
bafi: cannot parse MT940 input from "statement.sta"
  no {1: header block found (headerStartNotFound)

hint: an MT940 message starts with the {1:...}{2:...}{4: header, uses CRLF line endings and ends with -}; for files with several messages (e.g. Multicash) set the separator, e.g. -d "-}\r\n"
docs: https://mmalcek.github.io/bafi/troubleshooting/#invalid-input
```

MT940 files need the `{1:...}{2:...}{4:` header, CRLF line endings and `-}` at the end. BSON input must
be a BSON document or a `mongodump` file; MongoDB extended JSON is JSON (`-f json`).

## Templates

### Template syntax error

<!-- test:file report.tmpl -->
```
Users:
{{range .users}}
- {{.name}}
```

<!-- test:cmd -->
```sh
bafi -i data.json -t report.tmpl
```

<!-- test:stderr -->
```
bafi: template error in report.tmpl (line 2)

  2 | {{range .users}}
  unexpected EOF: the {{range}} started here has no {{end}}

hint: a {{if}}, {{range}} or {{with}} block is missing its {{end}}
docs: https://mmalcek.github.io/bafi/troubleshooting/#template-syntax-error
```

Every `{{if}}`, `{{range}}` and `{{with}}` needs its `{{end}}`, and every `{{` needs `}}`.

### Unknown function

<!-- test:cmd -->
```sh
bafi -i data.json -t '?{{uper .name}}'
```

<!-- test:stderr -->
```
bafi: template error in the inline template (line 1)

  1 | {{uper .name}}
  function "uper" not defined

hint: there is no function "uper" - did you mean "upper"?
docs: https://mmalcek.github.io/bafi/troubleshooting/#unknown-function
```

Function names are case-sensitive. See the [function reference](functions.md).

### Keys with dashes

<!-- test:file dashes.json -->
```json
{"order-id": 1001}
```

<!-- test:cmd -->
```sh
bafi -i dashes.json -t '?{{.order-id}}'
```

<!-- test:stderr -->
```
bafi: template error in the inline template (line 1)

  1 | {{.order-id}}
  bad character U+002D '-'

hint: keys containing "-" cannot be used with a dot; use index: {{index . "my-key"}} (XML attributes are "-name")
docs: https://mmalcek.github.io/bafi/troubleshooting/#keys-with-dashes
```

A dot path cannot contain `-` (or spaces). Use `index`, which also reaches XML attributes, because those
are stored with a leading `-`:

<!-- test:cmd -->
```sh
bafi -i dashes.json -t '?{{index . "order-id"}}'
```

<!-- test:output -->
```
1001
```

### Wrong type

<!-- test:cmd -->
```sh
bafi -i data.json -t '?{{upper .count}}'
```

<!-- test:stderr -->
```
bafi: template error in the inline template (line 1, column 9)

  1 | {{upper .count}}
    |         ^
  wrong type for value; expected string; got float64 (at <.count>)

hint: the value is a float64, the function needs text; convert it first, e.g. (toString .value)
docs: https://mmalcek.github.io/bafi/troubleshooting/#wrong-type
```

JSON numbers are numbers (`float64`), while CSV and XML values are always text. Convert where the
function expects something else: `{{upper (toString .count)}}`, `{{add (toInt .qty) 1}}`. `range` needs a
list: an XML element that occurs once is not a list, wrap it with `mustArray`.

The same applies to comparisons, where it is easy to miss: a number written in the template is an
integer, a JSON or YAML number is a `float64`, and the two cannot be compared directly.

<!-- test:cmd -->
```sh
bafi -i data.json -t '?{{if gt .count 2}}many{{end}}'
```

<!-- test:stderr -->
```
bafi: template error in the inline template (line 1, column 6)

  1 | {{if gt .count 2}}many{{end}}
    |      ^
  error calling gt: incompatible types for comparison: float64 and int (at <gt .count 2>)

hint: numbers from JSON and YAML are float64, numbers written in the template are int; write the number with a decimal point (2.0) or convert the value: (toInt .count)
docs: https://mmalcek.github.io/bafi/troubleshooting/#wrong-type
```

Write the number with a decimal point, or convert the value:

<!-- test:cmd -->
```sh
bafi -i data.json -t '?{{if gt .count 2.0}}many{{end}} / {{if gt (toInt .count) 2}}many{{end}}'
```

<!-- test:output -->
```
many / many
```

### Missing keys and `<no value>`

A key that is not in the input prints `<no value>`. bafi keeps the output but warns:

<!-- test:cmd -->
```sh
bafi -i data.json -t '?Hello {{.nmae}}'
```

<!-- test:stderr exit=0 -->
```
bafi: warning: output contains <no value> 1 time(s) - a key used in the template is missing from the input (or is null); run with -strict to see which one
```

`-strict` turns it into an error that names the key and the line:

<!-- test:cmd -->
```sh
bafi -i data.json -strict -t '?Hello {{.nmae}}'
```

<!-- test:stderr -->
```
bafi: template error in the inline template (line 1, column 9)

  1 | Hello {{.nmae}}
    |         ^
  map has no entry for key "nmae" (at <.nmae>)

hint: the input has no key "nmae" here (-strict is on); check spelling and case - XML attributes start with "-" (index . "-id"); for keys that may be absent use {{with index . "nmae"}}...{{end}}
docs: https://mmalcek.github.io/bafi/troubleshooting/#missing-keys-and-no-value
```

Check spelling and case (`Name` and `name` are different keys); XML attributes start with `-`. To see
what bafi parsed, print it: `bafi -i data.json -t '?{{toJSON .}}'`.

Under `-strict`, keys that may legitimately be absent need `index`, which returns nothing instead of
failing: `{{with index . "nickname"}}({{.}}){{end}}`.

## Installation

### macOS: bafi cannot be opened

macOS blocks programs downloaded with a browser that are not signed by an Apple developer
("cannot be opened because the developer cannot be verified"). Remove the quarantine flag once:

```sh
xattr -d com.apple.quarantine ./bafi
```

Files downloaded with `curl` are not quarantined. See [Install](install.md).

### PowerShell: bafi is not recognized

PowerShell does not run programs from the current directory by name. Use `.\bafi.exe`, or put bafi in a
directory listed in `PATH`. Windows SmartScreen may warn the first time; choose **More info**, then
**Run anyway**.

### Lua errors in the output

Lua problems do not stop bafi; the output contains `luaError: ...` where the function's result should
be. See [Custom logic with Lua](recipes/lua.md#errors).
