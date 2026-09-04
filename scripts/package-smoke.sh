#!/usr/bin/env bash
set -euo pipefail

PACKAGE_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SMOKE_ROOT="$(mktemp -d)"
trap 'rm -rf "$SMOKE_ROOT"' EXIT

cat > "$SMOKE_ROOT/go.mod" <<EOF
module waffo-go-smoke

go 1.20

require github.com/waffo-com/waffo-sdk/packages/waffo-go v0.0.0

replace github.com/waffo-com/waffo-sdk/packages/waffo-go => $PACKAGE_ROOT
EOF

cat > "$SMOKE_ROOT/main.go" <<'GO'
package main

import (
	"fmt"

	"github.com/waffo-com/waffo-sdk/packages/waffo-go/config"
)

func main() {
	if config.SDKVersion == "" {
		panic("SDKVersion is empty")
	}
	fmt.Println("Imported waffo-go", config.SDKVersion)
}
GO

cd "$SMOKE_ROOT"
go run .
echo "✓ Local Go module is importable by a clean consumer"
