#!/usr/bin/env bash
set -euo pipefail

PACKAGE_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SMOKE_ROOT="$(mktemp -d)"
trap 'rm -rf "$SMOKE_ROOT"' EXIT
MODULE_PATH="$(cd "$PACKAGE_ROOT" && go list -m -f '{{.Path}}')"
MODULE_VERSION="v0.0.0"
if [[ "$MODULE_PATH" =~ /v([0-9]+)$ ]] && [ "${BASH_REMATCH[1]}" -ge 2 ]; then
  MODULE_VERSION="v${BASH_REMATCH[1]}.0.0"
fi

cat > "$SMOKE_ROOT/go.mod" <<EOF
module waffo-go-smoke

go 1.20

require $MODULE_PATH $MODULE_VERSION

replace $MODULE_PATH => $PACKAGE_ROOT
EOF

cat > "$SMOKE_ROOT/main.go" <<GO
package main

import (
	"fmt"

	"$MODULE_PATH/config"
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
