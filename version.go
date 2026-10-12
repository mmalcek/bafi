package main

import (
	_ "embed"
	"strings"
)

//go:embed version.txt
var versionFile string

// version is the release version, trimmed from version.txt (single source of truth, also read by Taskfile.yml).
var version = strings.TrimSpace(versionFile)
