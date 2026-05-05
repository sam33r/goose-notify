# goose-notify — Design

**Status:** Approved (brainstorm), pending implementation plan
**Date:** 2026-05-04
**Owner:** Sameer Ahuja

## Summary

`goose-notify` is a macOS CLI that displays a black, rounded-rectangle toast near
the top-center of the active screen with a fade-in/fade-out animation, then exits.
It's a one-shot transitional UI for Goose plugin notifications, modeled on the
high-level approach of `goose-launcher` (Go + Gio + a small cgo `NSWindow` shim).

## Goals

- Custom-styled toast that matches a specific visual (black rounded rect, white
  text, optional bold title, smooth fade) without macOS Notification Center's
  styling baggage.
- Click-through, non-focus-stealing overlay so it can appear over any work
  without interrupting it.
- Fast enough for plugin scripts to fire occasionally (one-shot per invocation
  is acceptable; ~50ms launch overhead is fine).
- Reuse patterns and mental model from `goose-launcher`.

## Non-goals (v1)

- Multiple simultaneous toasts, queueing, or stacking.
- Click-to-dismiss, ESC, hover-to-extend, or any user interaction.
- Sound, icons, images.
- Markup or rich text.
- Color or font configuration.
- Linux / Windows support.
- Daemon mode (revisit only if launch latency becomes a problem in practice).

## User-visible behavior

### Invocation

```
goose-notify [flags] [message]
```

Message source:

- Positional arg wins: `goose-notify "your message"`
- Otherwise stdin: `echo "your message" | goose-notify`
- Empty message (no arg, empty stdin) is an error.

### Flags

| Flag           | Type     | Default | Meaning                                          |
|----------------|----------|---------|--------------------------------------------------|
| `--title`      | string   | (none)  | Bold first line above body (optional)            |
| `--duration`   | duration | `2s`    | How long the toast stays fully visible           |
| `--fade-in`    | duration | `200ms` | Fade-in animation duration                       |
| `--fade-out`   | duration | `300ms` | Fade-out animation duration                      |
| `--max-width`  | int      | `600`   | Soft cap on toast width in pixels                |
| `--offset-y`   | int      | `80`    | Vertical offset from top of active screen (px)   |
| `--version`    | bool     | —       | Print version and exit                           |
| `--help`       | bool     | —       | Print usage and exit                             |

Durations parse via `time.ParseDuration` (`2s`, `500ms`, etc.).

### Exit codes

- `0` on normal completion (after fade-out finishes).
- `1` on flag error, empty message, multiple positional args, or window-creation
  timeout. Error message goes to stderr.

### Visual

- Rounded rectangle, ~16px corner radius.
- Background: `#000000` at 92% opacity.
- Padding: 20px vertical, 28px horizontal.
- Title (if present): JetBrains Mono Bold, 14pt, white, 6px gap below.
- Body: JetBrains Mono Regular, 13pt, white at 90% opacity.
- Wraps at `--max-width`. Box auto-sizes to content height. Min width ~200px.
- No border, no shadow in v1.

### Position

- Horizontally centered on the active screen's visible frame (excludes menu bar
  and Dock).
- `--offset-y` from the top of the visible frame (default 80px).
- Active screen = screen of the cursor at launch (`NSEvent.mouseLocation`).
  No tracking after launch.

### Animation

```
t=0ms      window mapped, opacity=0
t=0–200ms  fade-in (opacity 0 → 1, easeOutCubic)
t=200ms+   hold for --duration (default 2.0s)
t=2200ms   fade-out begins (opacity 1 → 0, easeInCubic, 300ms)
t=2500ms   window unmapped, process exits
```

Default total wall-clock = 2.5s plus ~50ms launch overhead. Opacity only — no
slide-in, to keep frames cheap and predictable.

### Window behavior (macOS)

The Gio window is created transparent and borderless, then `pkg/macwin` finds it
by title and applies these `NSWindow` settings:

- `setStyleMask:NSWindowStyleMaskBorderless`
- `setOpaque:NO` + `setBackgroundColor:[NSColor clearColor]`
- `setLevel:NSStatusWindowLevel` (above normal app windows, below true alerts)
- `setIgnoresMouseEvents:YES` (clicks pass through)
- `setCollectionBehavior:` `CanJoinAllSpaces | FullScreenAuxiliary | Stationary | IgnoresCycle`
- `setHidesOnDeactivate:NO`
- `orderFrontRegardless` (show without stealing key/focus)

Process activation policy: `NSApplicationActivationPolicyAccessory` — no Dock
icon, no menu bar.

### Concurrency

Each invocation is a fully independent process. If two run at once, two windows
appear and may overlap visually. No coordination, no IPC, no daemon.

## Architecture

Single Go module: `github.com/sam33r/goose-notify` (Go 1.25).

```
stdin/arg → input.Read → string (title?, body)
                              ↓
                      ui.Toast.Run()   ← config (durations, position, sizing)
                              ↓
                          exit 0
```

`cmd/goose-notify/main.go` is intentionally thin: parse flags, read input,
launch the UI on a goroutine, hold the main thread with `app.Main()` (Gio macOS
requirement, identical pattern to `goose-launcher`).

```go
go func() { runUI(); os.Exit(0) }()
app.Main()
```

### Packages

| Package          | Responsibility                                                                 |
|------------------|--------------------------------------------------------------------------------|
| `pkg/input`      | Resolve message: positional arg wins, else stdin. Trim trailing newline.       |
| `pkg/config`     | Flag parsing and defaults.                                                     |
| `pkg/ui`         | Gio window, layout (rounded black rect + optional title + body), animation.   |
| `pkg/macwin`     | cgo shim: find Gio NSWindow by title and apply the flags listed above.        |
| `pkg/fontcache`  | On-disk font cache, copied from `goose-launcher` so font parsing isn't on the hot launch path. |

No daemon, no matcher, no input handling beyond reading the message. Roughly
half the surface of `goose-launcher`.

### macwin: shape and difference from goose-launcher

`goose-launcher`'s macwin keeps a daemon's window alive and toggles visibility.
`goose-notify`'s macwin doesn't manage a lifecycle — it applies a fixed set of
flags once on startup, then never touches the window again. Same cgo file
layout (`macwin.go` + `macwin.m`, `//go:build darwin`), different exported
surface:

```go
// macwin (notify)
func SetAccessoryPolicy()
func ConfigureToastWindow(title string, timeout time.Duration) error
```

`ConfigureToastWindow` finds the Gio NSWindow by title (polling like
`goose-launcher`'s `FindWindowByTitle`), then applies all the flags in one
shot. Returns error on timeout.

## Error handling

Small surface, no defensive scaffolding:

- Flag parse error → stderr (Go's `flag` package default) + exit 1.
- Empty message → `goose-notify: empty message` to stderr + exit 1.
- Multiple positional args → `goose-notify: expected at most one positional message argument` to stderr + exit 1.
- `macwin.ConfigureToastWindow` timeout (1s) → log to stderr + exit 1.
- Otherwise: standard Go panics propagate; no silent swallow.

Logging: stderr only. No log file. One-shot tool.

## Testing

| Layer            | Strategy                                                                 |
|------------------|--------------------------------------------------------------------------|
| `pkg/input`      | Table tests: arg/stdin precedence, empty, trim behavior.                 |
| `pkg/config`     | Table tests for flag parsing including durations.                        |
| `pkg/ui`         | Unit tests for layout math (text wrap, box sizing) and animation curves (endpoint + midpoint values for `easeOutCubic`/`easeInCubic`). Cannot render a real window in `go test`. |
| `pkg/macwin`     | None. Verified via integration.                                          |
| Integration      | `./test-notify.sh` rebuilds + fires the binary with canned messages for visual eyeball. Mandatory before claiming UI changes work. |

No CI for the visual side; nothing renders headless.

## Conventions (carry-overs from goose-launcher)

- Never commit built binaries. `.gitignore` covers `goose-notify` and any
  `test-*` artifacts.
- `BENCHMARK_MODE=1` env var pattern is **not** ported in v1; revisit if launch
  latency becomes a tracked concern.
- UI changes that affect layout or animation must be verified interactively via
  `./test-notify.sh` — Go tests don't render real windows.

## Future considerations (not v1)

Captured here so they're not forgotten if the tool gains usage:

- Daemon mode for sub-10ms launch (mirroring `goose-launcher`'s daemon path).
- Stacking / queueing multiple toasts.
- Click-to-dismiss, ESC.
- Color and font configuration flags.
- Pango-style markup (`pkg/markup` already exists in `goose-launcher` and could
  be lifted).
- Sound and icon support.
