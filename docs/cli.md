# Command line

```
bafi -i INPUT -t TEMPLATE [-o OUTPUT]
COMMAND | bafi -f FORMAT -t TEMPLATE
```

`bafi -h` prints the same overview with all flags.

## Flags

| Flag | Meaning |
|---|---|
| `-i file` | Input file. Without `-i`, bafi reads standard input (a pipe or `< file`). `-i '?list.yaml'` reads [several files](#several-input-files). |
| `-t template` | Template file, or an inline template starting with `?`: `-t '?{{toJSON .}}'`. |
| `-o file` | Output file. Without `-o`, the result goes to standard output. |
| `-f format` | Input format: `json`, `bson`, `yaml`, `csv`, `xml`, `mt940`. Default: from the `-i` file extension. Required for standard input. |
| `-d delimiter` | CSV delimiter: one character (`-d ';'`) or a hex byte (`-d 0x09` for TAB). Default `,`. For MT940, the separator between messages. |
| `-strict` | Fail on a key that is missing from the input instead of printing `<no value>`. |
| `-v` | Print the version. |
| `-h` | Print help. |
| `-gq question` | Ask OpenAI about the input instead of using a template. See [Ask OpenAI](recipes/openai.md). |
| `-gk key` | OpenAI API key. Default: the `OPENAI_API_KEY` environment variable (preferred). |
| `-gm model` | OpenAI model: `gpt35` (default), `gpt4`, `gpt4o`, `gpt4o-mini`, `gpt5`, `gpt5mini`, `gpt5nano`. |

Flags must come before any other argument.

## Input

<!-- test:cmd -->
```sh
bafi -i testdata.xml -t '?{{index .TOP_LEVEL "-description"}}'
```

<!-- test:output -->
```
employeeData
```

The format is taken from the extension: `.json`, `.bson`, `.yaml`/`.yml`, `.csv`, `.xml` (also `.cdf`,
`.cdf3`), `.sta` (MT940). `-f` overrides it. Standard input has no name, so `-f` is required:

```sh
curl -fs https://example.com/api/users | bafi -f json -t users.tmpl
```

Redirecting a file works the same way:

<!-- test:file users.json -->
```json
{"users": [{"name": "Alice"}, {"name": "Bob"}]}
```

<!-- test:cmd -->
```sh
bafi -f json -t '?{{len .users}} users' < users.json
```

<!-- test:output -->
```
2 users
```

A single `-` after the flags, the usual "read standard input" marker, is accepted and changes nothing.

A UTF-8 byte order mark at the start of the input is ignored.

### Several input files

`-i '?list.yaml'` reads every file named in a YAML list and makes each available under its label (quote it: zsh, the macOS
default shell, treats an unquoted `?` as a wildcard):

```yaml
- file: rates.xml
  label: RATES          # {{ .RATES }}
- file: orders.json
  label: ORDERS
  format: json          # optional; the extension decides when omitted
```

`-f` is ignored in this mode; `-d` applies to every CSV file. Full example:
[Merge several files into one output](recipes/merge-files.md).

## Templates on the command line

An inline template starts with `?`. Quote it so the shell passes it as one argument:

| Shell | Example |
|---|---|
| bash, zsh | `bafi -i data.json -t '?{{toXML .}}'` |
| PowerShell | `.\bafi.exe -i data.json -t "?{{toXML .}}"` |
| cmd.exe | `bafi.exe -i data.json -t "?{{toXML .}}"` |

Inside single quotes, template strings use double quotes: `-t '?{{index . "my-key"}}'`. A template
file avoids quoting issues altogether: `-t report.tmpl`. The [template guide](templates.md) explains the
syntax.

## Output

The result goes to standard output unless `-o` names a file. bafi renders the whole result first: when
a template fails, standard output stays empty and an existing `-o` file is left as it was, so a pipe
never receives half a document and a scheduled job keeps its last good report.

Append to a file with the shell:

```sh
bafi -i testdata.xml -t template.tmpl >> output.txt
```

## Errors, warnings and exit codes

Errors and warnings go to standard error, never into the output. An error names the file and line,
shows the problem and suggests a fix; see [Troubleshooting](troubleshooting.md).

| Exit code | Meaning |
|---|---|
| `0` | Success (warnings may have been printed). |
| `1` | The input, the template, the output file or the OpenAI request failed. |
| `2` | The command line is wrong: missing template or input, unknown flag or format. |

Warnings do not change the output or the exit code:

- the output contains `<no value>` because the template used a key the input does not have;
- a CSV header ended up as a single column, which usually means the wrong `-d`.

### Strict mode

`-strict` turns a missing key into an error with the key name and the template line, instead of
printing `<no value>`:

<!-- test:cmd -->
```sh
bafi -i testdata.xml -strict -t '?{{.TOP_LEVEL.DATA_LINES}}'
```

<!-- test:stderr -->
```
bafi: template error in the inline template (line 1, column 3)

  1 | {{.TOP_LEVEL.DATA_LINES}}
    |   ^
  map has no entry for key "DATA_LINES" (at <.TOP_LEVEL.DATA_LINES>)

hint: the input has no key "DATA_LINES" here (-strict is on); check spelling and case - XML attributes start with "-" (index . "-id"); for keys that may be absent use {{with index . "DATA_LINES"}}...{{end}}
docs: https://mmalcek.github.io/bafi/troubleshooting/#missing-keys-and-no-value
```

Use it while writing a template, and in automation where a missing field must not go unnoticed.
For keys that may legitimately be absent, use `index`: `{{with index . "nickname"}}{{.}}{{end}}`.
