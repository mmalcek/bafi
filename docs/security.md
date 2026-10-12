# Security notes

bafi is a local formatter: whoever controls the template controls the output. Keep these points in
mind when you use it in automation.

## Templates and Lua are trusted code

A template can call every built-in function and every Lua function, so never run templates or a `lua/`
directory from an untrusted source. Only the input is treated as data: values in the input can never
inject template actions.

`./lua/functions.lua` is loaded from the current working directory the first time a template calls
`lua`, and runs with the permissions of the bafi process. Functions that start programs, end bafi or
delete and rename files are removed (`os.execute`, `os.exit`, `os.remove`, `os.rename`, `os.tmpname`,
`io.popen`); the rest of the Lua standard library (`os.date`, `os.time`, `io.open`, `require`, ...)
works normally. `bafi -v`, `bafi -h` and templates that do not call `lua` never execute Lua code.

## No output escaping

bafi uses Go's `text/template`, which writes values verbatim. If the output is HTML, XML, SQL, a shell
script or JSON, a value containing quotes, `<` or `;` goes straight through. Use `toJSON`, `toXML`,
`toYAML` or `toTOON` to produce structured output from data instead of building it from strings, use
`html` for HTML and XML text, and escape values yourself for other formats
(see [Template guide](templates.md#escaping-for-the-output-format)).

## CSV formula injection

Spreadsheet software evaluates cells that start with `=`, `+`, `-`, `@`, TAB or CR. When CSV output will
be opened in Excel or LibreOffice and the input is not under your control, wrap values with `csvSafe`
(see [JSON to CSV](recipes/json-to-csv.md)).

## OpenAI mode sends your data away

With `-gq`, the question **and the entire parsed input** are sent to OpenAI. Pass the key through the
`OPENAI_API_KEY` environment variable rather than `-gk`, so it does not appear in process listings and
shell history.

## randInt is not a secure random source

Use `uuid` (backed by `crypto/rand`) for identifiers; never derive tokens or passwords from `randInt`.

## Verbose error messages

Errors include file paths and an excerpt of the input or template to make problems easy to fix. Keep
that in mind if standard error ends up in shared logs.

## Verifying downloads

Every release lists the SHA-256 checksum of each archive in `SHA256SUMS.txt`; see
[Install](install.md#verify-the-download).
