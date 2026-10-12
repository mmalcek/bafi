# Changelog

Release notes for bafi. Each release is a level-2 heading (`## X.Y.Z`) matching
`version.txt`. `task release` extracts the matching section as the GitHub release
body and refuses to tag if the section is missing. Newest release first.

## 1.5.0

### Added

- **Helpful error messages.** Errors name the input or template and, where the
  parser reports it, the line and column; they show the offending line with a
  caret, explain the likely cause and link to the matching section of the new
  troubleshooting page. Covers JSON, YAML, XML and CSV input (with location),
  BSON and MT940 input, template syntax and execution errors, files and flags.
  Unknown functions and input formats get a "did you mean" suggestion.
- `-strict` flag: a key missing from the input stops the run with an error that
  names the key and the template line, instead of printing `<no value>`.
- Warnings on standard error (output unchanged): when the output contains
  `<no value>`, and when a CSV header ends up as a single column because of a
  wrong `-d`.
- Task-oriented documentation: 16 recipes (XML to JSON, JSON to CSV, REST to
  SOAP, HTML reports, SQL from CSV, MT940, ...), install, command line, template
  guide, function reference, troubleshooting and security pages. The sample
  commands, the function tables and the error examples in the documentation are
  executed by the test suite.
- The online demo is built from this repository on every release and loads
  only when used. It shows the same error messages as the CLI.

### Changed

- Errors are printed without a timestamp and without internal function names.
- Usage mistakes (no template, no input, unknown flag or `-f` value, `-gk`
  without `-gq`, a stray argument such as an unquoted inline template) print to
  standard error and exit with code **2**. Before, some printed to standard
  output and exited with 0. Runtime errors still exit with 1. A single trailing
  `-` (read standard input) is still accepted.
- With `-o`, bafi renders the whole result before writing the file: a failing
  template leaves an existing output file untouched. Before, an error during
  execution left it half-written.
- `bafi -h` shows usage examples and a link to the documentation.
- Standard input redirected from a file (`bafi -f json ... < data.json`) is
  accepted; before, only pipes were.
- The docs build runs in strict mode, so broken internal links fail it.

### Fixed

- `toXML` wrote `&`, `<`, `>` and quotes unescaped, producing invalid XML; for
  lists it failed with an `err:` text instead.
- Files listed in `-i '?files.yaml'` now have a UTF-8 byte order mark removed,
  like a single `-i` file.
- Some malformed BSON input crashed bafi with a Go stack trace; it is now
  reported as an input error.
- Several documentation examples were wrong or depended on APIs that no longer
  exist.

## 1.4.0

### Security

- **Lua sandbox.** The Lua state no longer exposes `os.execute`, `os.exit`,
  `os.remove`, `os.rename`, `os.tmpname` and `io.popen`. Everything else in the
  Lua standard library (`os.date`, `os.time`, `io.open`, `require`, ...) works as
  before. If your `functions.lua` shelled out through one of the removed
  functions, it will now fail with `luaError: ... attempt to call a non-function object`.
- **Lazy Lua loading.** `./lua/functions.lua` is loaded the first time a template
  calls `{{lua ...}}`, not at startup. `bafi -v`, `bafi -h` and templates that do
  not use Lua never execute code from the current directory.
- **OpenAI key from the environment.** The API key is read from `OPENAI_API_KEY`
  when `-gk` is not given, so it no longer has to appear in process listings and
  shell history. `-gk` still works and overrides the variable.
- **CI hardening.** All GitHub Actions are pinned by commit SHA, workflows declare
  minimal `permissions`, deprecated action versions are upgraded (checkout v7,
  setup-go v7, setup-python v7, CodeQL v4) and a `govulncheck` job runs on every
  push. Dependabot keeps the pins current.
- **Release pipeline rebuilt.** Releases are built on the GitHub runner with the
  Go version from `go.mod` (`task dist`) instead of the `go-release.action`
  Docker image. gofmt/vet, tests and govulncheck gate the release, every asset is
  listed in a `SHA256SUMS.txt`, and the release notes are the matching
  `CHANGELOG.md` section.
- `go.mongodb.org/mongo-driver` bumped to v1.17.10 (GO-2026-5327; the affected
  GSSAPI code was never reachable from bafi).
- Minimum Go raised to 1.27.2 (`go.mod`), the release that fixes the stdlib
  advisories GO-2026-6603 … GO-2026-6617 reported by `govulncheck`. Release
  binaries are built with the same version.
- Release archives: new `darwin_arm64` (Apple silicon) and `linux_arm64` builds
  next to the existing `windows_amd64`, `linux_amd64`, `linux_386` and
  `darwin_amd64`. The per-asset MD5 `_checksum.txt` files are replaced by one
  `SHA256SUMS.txt`.
- `task release` only tags and pushes; the GitHub release is created by CI.
- New `## Security notes` section in the documentation (trusted templates, no
  output escaping, CSV formula injection, OpenAI data sharing, `randInt`).

### Added

- `csvSafe` template function: prefixes values starting with `=`, `+`, `-`, `@`,
  TAB or CR with `'` to defuse spreadsheet formula injection. Plain numbers such
  as `-5` are left untouched.

### Fixed

- Lua functions returning a non-string leaked one value on the Lua stack per call.
- A missing or broken `./lua/functions.lua`, or a non-string function name, now
  yields an inline `luaError:` in the output instead of aborting the whole run.
- Unicode CSV delimiters (`-d`) are handled correctly (PR #13).
- A malformed `-i ?files.yaml` record (wrong type or missing `file`/`label`) is
  reported as `fileList[<index>]: ...` instead of crashing with a stack trace.
- An invalid hex CSV delimiter (`-d 0xZZ`) is reported as `mapCSV: invalid
  delimiter` through the normal error path instead of `log.Fatalf`.
- `toXML` renders `[]interface{}` slices (e.g. from `mustArray` or nested JSON
  arrays) as `<record>` elements instead of failing on a type assertion.
- `randInt` cannot overflow on extreme bounds such as `{{randInt 0 9223372036854775807}}`.
- An invalid pattern in `replaceAllRegex` aborts the template with a readable
  `error calling replaceAllRegex: error parsing regexp: ...` instead of a
  recovered panic.
- An OpenAI response without choices is reported as an error instead of a panic.

### Changed

- `-gq` alone now selects OpenAI mode (key from `-gk` or `OPENAI_API_KEY`).
  Without any key it fails with `OpenAI API key must be defined` instead of being
  silently ignored.
- `format` in the `-i ?files.yaml` file list is optional; it is detected from the
  file extension when omitted, exactly like for a single `-i` file.
- `randInt` uses `math/rand/v2`; the explicit `rand.Seed` at startup is gone
  (the generator is auto-seeded since Go 1.20).
- Version is read from `version.txt` (embedded at build time) instead of a
  constant in `main.go`.
- New `Taskfile.yml` with `task build|test|vet|check|vuln|docs|release|clean`.
  `task release` tags, pushes and publishes the GitHub release that triggers the
  CI build.
- The changelog is part of the documentation site.
- CI release build runs only on published releases (PR #12).

## 1.3.0

Reconstructed from git history; this release predates the changelog.

### Added

- TOON (Token Oriented Object Notation) output via the `toTOON` template function.

### Fixed

- `randInt` handles an invalid range and includes the upper bound.
