#!/bin/bash
# Build + install goose-notify from source.
#
# Installs the binary via `go install` (lands in $GOBIN, defaulting to
# $GOPATH/bin). One-shot tool — no daemon to restart.
#
# Usage: ./install.sh

set -e

cd "$(dirname "$0")"

if [[ "$(uname -s)" != "Darwin" ]]; then
    echo "Error: goose-notify is macOS-only (uses AppKit/cgo)." >&2
    exit 1
fi

if ! command -v go &>/dev/null; then
    echo "Error: Go is not installed. Get it from https://go.dev/dl/" >&2
    exit 1
fi

GOBIN="$(go env GOBIN)"
if [[ -z "$GOBIN" ]]; then
    GOBIN="$(go env GOPATH)/bin"
fi

echo "=== Goose Notify Install ==="
echo "Go:     $(go version | awk '{print $3}')"
echo "Target: $GOBIN"
echo

echo "Building + installing..."
go install ./cmd/goose-notify
echo "  -> $GOBIN/goose-notify"
echo

if ! command -v goose-notify &>/dev/null; then
    echo "Warning: $GOBIN is not on your PATH; add it to use 'goose-notify' bare." >&2
fi

echo "Done."
echo
echo "Try it:  goose-notify 'hello from goose-notify'"
echo "         echo 'piped' | goose-notify"
echo "         goose-notify --title 'Status' 'all systems normal'"
