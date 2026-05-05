#!/usr/bin/env bash
# Integration smoke for goose-notify. Rebuilds, then fires a few canned
# toasts so the developer can eyeball the result. Mandatory before claiming
# any UI change works (Go tests don't render real windows).
set -euo pipefail

cd "$(dirname "$0")"
make build

echo "[1/5] short message via arg"
./goose-notify "hello world"
sleep 0.5

echo "[2/5] message via stdin"
echo "piped message" | ./goose-notify
sleep 0.5

echo "[3/5] with title"
./goose-notify --title "Status" "all systems normal"
sleep 0.5

echo "[4/5] long wrapping message"
./goose-notify "this is a longer message that should wrap across multiple lines once it exceeds the configured max-width of six hundred pixels"
sleep 0.5

echo "[5/5] custom duration"
./goose-notify --duration 1s --fade-in 100ms --fade-out 100ms "fast toast"

echo "Done. Verify each toast appeared at top-center, faded smoothly, and didn't steal focus."
