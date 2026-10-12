# bafi - CLAUDE.md

## Purpose

CLI that renders structured input (JSON, BSON, YAML, CSV, XML, MT940) through Go `text/template`
templates, with optional user-defined Lua helper functions and an experimental OpenAI query mode.
Typical use: translate data between services (REST to SOAP, XML to CSV, ...) in a pipe.
Module `github.com/mmalcek/bafi`. Single `main` package, no internal packages.

## Layout

- `main.go` - flags, input parsing per format, template execution, output.
- `functions.go` - the template `FuncMap` (math, dates, encoding, regex, `toJSON`/`toXML`/`toTOON`, `lua`).
- `lua.go` - lazy, restricted Lua state. `luaRestrictedFunctions` is the list of removed stdlib functions.
- `lua/` - shipped example `functions.lua` and `json.lua`; also what release artifacts bundle.
- `docs/` - mkdocs guide (`README.md`, `examples.md`, `about.md`, `changelog.md` -> symlink to `../CHANGELOG.md`).
  Deployed to GitHub Pages by the release workflow.
- `version.txt` - the one version. Embedded into the binary (`bafi -v`) and read by `Taskfile.yml`.
- `CHANGELOG.md` - release notes, one `## X.Y.Z` section per release.
- `Taskfile.yml` - build, test, docs and release tasks (go-task).

## Security model

Templates and `./lua/functions.lua` are **trusted code**: whoever controls them controls the output and
can run Lua. The Lua sandbox in `lua.go` only removes process-spawning and file-destruction primitives
(`os.execute`, `os.exit`, `os.remove`, `os.rename`, `os.tmpname`, `io.popen`) and defers loading until a
template actually calls `lua`. Do not re-add those functions and do not move loading back to startup.
No other template function performs I/O; keep it that way.

## Commands

```bash
task build        # go build -> ./bafi
task test         # go test ./...
task vet          # gofmt check + go vet
task check        # vet + test + docs build — run before every commit
task vuln         # govulncheck
task docs         # mkdocs build -> ./site
task docs:serve   # live-reload docs
task dist         # cross-compile release archives into ./build (what CI publishes)
task release      # tag + push; CI builds archives, publishes the release, deploys docs
task clean
```

## Keep docs and changelog in sync (MUST)

Every user-visible change — new flag, template function, format, behavior change, bug fix — updates in
the **same change**:

1. `CHANGELOG.md` under the upcoming `## X.Y.Z` section (`Added` / `Changed` / `Fixed` / `Security`).
   If that section does not exist yet, bump `version.txt` and create it.
2. The affected page under `docs/` (functions list, examples, Lua notes).

`task release` refuses to run without a `## <version.txt>` section. Development rules (gofmt,
`go test ./...`, `mkdocs build` after docs changes) live in `AGENTS.md` and apply as well.

## Release

1. Bump `version.txt`, make sure `CHANGELOG.md` has the matching `## X.Y.Z` section, commit.
2. `task release` on a clean `main`: runs `task check`, tags `vX.Y.Z` and pushes.
3. The pushed tag triggers `.github/workflows/release.yml`: verifies the tag against `version.txt`, runs
   vet/tests/govulncheck, builds the archives with `task dist`, writes `SHA256SUMS.txt`, creates the
   GitHub release with the changelog section as body, then deploys the docs. Do not create releases by
   hand in the GitHub UI. `workflow_dispatch` runs the build without publishing.
