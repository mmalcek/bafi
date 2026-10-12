# Changelog

Release notes for bafi. Each release is a level-2 heading (`## X.Y.Z`) matching
`version.txt`. `task release` extracts the matching section as the GitHub release
body and refuses to tag if the section is missing. Newest release first.

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

### Fixed

- Lua functions returning a non-string leaked one value on the Lua stack per call.
- A missing or broken `./lua/functions.lua`, or a non-string function name, now
  yields an inline `luaError:` in the output instead of aborting the whole run.
- Unicode CSV delimiters (`-d`) are handled correctly (PR #13).

### Changed

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
