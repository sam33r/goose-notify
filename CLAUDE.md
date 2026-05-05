# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

`goose-notify` is a macOS-only, one-shot CLI that displays a black, rounded-rectangle toast near the top-center of the active screen, then exits. It's the notification surface for Goose plugins, designed to pair with `goosey` and `goose-launcher` (the latter is also the architectural template — same Go + Gio + cgo `NSWindow` shim pattern, roughly half the surface).

Authoritative design: `docs/superpowers/specs/2026-05-04-goose-notify-design.md`. Read it before changing user-visible behavior or the package layout.

## Build & test

```bash
make build           # host arch → ./goose-notify
make build-macos     # universal binary (arm64 + amd64 via lipo)
make install         # universal binary to /usr/local/bin
make test            # go test -v ./...

# Single test
go test -v ./pkg/config -run TestParseDefaults

# Mandatory visual smoke after UI / animation / window-config changes.
# Go tests cannot render a real window — this is the only way to verify.
./test-notify.sh
```

Note: the Makefile builds `./cmd/goose-notify`, but `cmd/` does not yet exist in the tree. Wiring `cmd/goose-notify/main.go` is the final outstanding task in `docs/superpowers/plans/2026-05-04-goose-notify-mvp.md` (Task 9). Until that lands, `make build` will fail; package-level tests (`go test ./pkg/...`) still work.

## Architecture: the two non-obvious things

**1. Two-stage window construction (Gio + macwin handoff).**
Gio creates a borderless NSWindow with a stable title (`ui.WindowTitle = "goose-notify-toast"`). `pkg/macwin` then *polls* `[NSApp windows]` by that title — Gio creates the NSWindow lazily, only after the first `FrameEvent`, so a synchronous lookup at startup would miss it. Once found, `macwin_configureToast` applies the AppKit-only flags in one shot: `NSStatusWindowLevel`, `setIgnoresMouseEvents:YES`, transparent background, `CanJoinAllSpaces | FullScreenAuxiliary | Stationary | IgnoresCycle`, and frames the window top-center on the screen under the cursor (`[NSEvent mouseLocation]`). `macwin.SetAccessoryPolicy()` overrides Gio's hardcoded `NSApplicationActivationPolicyRegular` so no Dock icon appears.

**2. Fade is driven by `NSWindow.alphaValue`, not by Gio paint colors.**
The naive approach (modulating per-frame color alpha) hits Gio's opaque framebuffer — transparent pixels rendered as white. Instead, Gio paints the rounded rect fully opaque every frame, and a Go ticker calls `handle.SetAlpha(...)` (which dispatches `[NSWindow setAlphaValue:]` on the main thread) to drive the fade-in/hold/fade-out timeline. AppKit composites with the alpha. If you change the animation, change it in `pkg/ui/toast.go`'s ticker goroutine, not in `paintToast`. The opacity curve itself lives in pure math at `pkg/ui/easing.go` (`Opacity(elapsed, fadeIn, hold, fadeOut)`).

## Package boundaries (enforced)

| Package         | Constraint                                                                    |
|-----------------|-------------------------------------------------------------------------------|
| `pkg/config`    | Pure stdlib (`flag`, `time`). No UI, no Gio.                                  |
| `pkg/input`     | Pure stdlib. Resolve message: positional arg wins, else stdin (trim trailing `\n`). |
| `pkg/ui/easing.go`, `pkg/ui/layout.go` | Pure math, no Gio types. Fully unit-tested.            |
| `pkg/ui/toast.go` | Owns Gio event loop + per-frame paint. No tests (visual only).             |
| `pkg/macwin`    | Only cgo in the project. `//go:build darwin`. Surface is just `SetAccessoryPolicy`, `ConfigureToast`, `Handle.SetAlpha`, `Handle.Free`. |
| `pkg/fontcache` | Lifted verbatim from goose-launcher; signature keeps italic/emoji params (nil-tolerated) so future ports don't need a new shape. |

`pkg/ui/layout.go` sizes the NSWindow *before* Gio creates it, so it uses a monospace char-cell approximation (JetBrains Mono is monospace — accurate enough). The window is created with fixed `MinSize == MaxSize == BoxSize(...)` so the AppKit frame matches the painted content. If you change padding or font sizes in `paintToast`, mirror the change in the `Metrics` struct passed to `BoxSize`.

## Conventions

- **Never commit binaries.** `.gitignore` covers `goose-notify`, `goose-notify-*`, and `test-*` (with `!test-*.sh` exception for the smoke script).
- **UI/animation/window-config changes require `./test-notify.sh`.** Go tests don't render. Eyeball each toast: top-center placement, smooth fade, no focus stealing, no Dock icon.
- **macOS only.** No Linux/Windows path. `pkg/macwin` is `//go:build darwin`.
- **One-shot, no daemon.** Each invocation = independent process. Two concurrent runs = two windows; no IPC, no coordination. Daemon mode is explicit non-goal in v1.
- **No defensive scaffolding.** Errors go to stderr + `os.Exit(1)`; no logging file, no recovery for spec'd error paths (empty message, multi-arg, macwin timeout). Match the spec's error surface — don't expand it.
