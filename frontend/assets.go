package frontend

import "embed"

// SourceFS contains the files needed to build the Web UI when AgentMeter is
// installed with `go install` and no source checkout is available.
//
//go:embed package.json package-lock.json index.html tsconfig.json tsconfig.node.json vite.config.ts public src
var SourceFS embed.FS
