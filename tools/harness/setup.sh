#!/bin/sh
# Builds the capture wasm into tools/harness/web<N> and serves tools/harness
# on :8767. Usage: tools/harness/setup.sh [N]   (default 8)
set -e
N=${1:-8}
H=$(cd "$(dirname "$0")" && pwd)
mkdir -p "$H/web$N"
cp "$H/index.html" "$H/web$N/"
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$H/web$N/"
(cd "$H/../.." && GOOS=js GOARCH=wasm go build -tags capture -o "$H/web$N/game.wasm" .)
(cd "$H" && nohup python3 -m http.server 8767 >/dev/null 2>&1 &)
sleep 1
echo "built web$N; serving $H on :8767"
