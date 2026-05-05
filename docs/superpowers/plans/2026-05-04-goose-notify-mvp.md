# goose-notify MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship a one-shot macOS CLI that prints a black, fading toast at the top-center of the active screen and exits, modeled on goose-launcher.

**Architecture:** Single Go module with thin `main.go`, pure unit-tested `pkg/config` and `pkg/input`, pure-math `pkg/ui` helpers (easing + box layout) with tests, a `pkg/fontcache` ported from goose-launcher, a `pkg/macwin` cgo shim that configures the Gio NSWindow to be borderless, transparent, click-through, non-key, top-level, and positions it on the active screen, and `pkg/ui/toast.go` that drives the fade-in/hold/fade-out timeline via Gio paint opacity.

**Tech Stack:** Go 1.25, Gio v0.9.0, cgo + AppKit (macOS-only), JetBrains Mono fonts (carry over from goose-launcher).

**Reference:** Spec at `docs/superpowers/specs/2026-05-04-goose-notify-design.md`. Reference implementation at `~/gt/goose-launcher` — same Go + Gio + macwin pattern.

---

## File Structure

```
goose-notify/
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── .gitignore
├── test-notify.sh
├── cmd/goose-notify/
│   └── main.go
└── pkg/
    ├── config/
    │   ├── config.go
    │   └── config_test.go
    ├── input/
    │   ├── input.go
    │   └── input_test.go
    ├── fontcache/
    │   └── fontcache.go        # ported from goose-launcher
    ├── macwin/
    │   ├── macwin.go           # cgo Go wrapper (darwin-only)
    │   └── macwin.m            # Obj-C: NSWindow flags, positioning
    └── ui/
        ├── easing.go
        ├── easing_test.go
        ├── layout.go           # box sizing math (pure)
        ├── layout_test.go
        ├── toast.go            # Gio Window driver + animation
        └── fonts/
            ├── JetBrainsMono-Regular.ttf
            └── JetBrainsMono-Bold.ttf
```

Boundaries:
- `pkg/config` and `pkg/input` know nothing about UI.
- `pkg/ui/easing.go` and `pkg/ui/layout.go` are pure math, no Gio types — fully unit-testable.
- `pkg/ui/toast.go` owns Gio + windowing.
- `pkg/macwin` is the only cgo, only darwin.

---

## Task 1: Project bootstrap

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `Makefile`
- Create: `README.md`
- Create: `pkg/ui/fonts/JetBrainsMono-Regular.ttf` (copy)
- Create: `pkg/ui/fonts/JetBrainsMono-Bold.ttf` (copy)

- [ ] **Step 1: Initialize go.mod**

```bash
cd /Users/sameer/src/goose-notify
go mod init github.com/sam33r/goose-notify
```

- [ ] **Step 2: Add Gio dependency to go.mod**

Run:
```bash
go get gioui.org@v0.9.0
```

Expected: `go.sum` populated, `go.mod` lists `gioui.org v0.9.0`.

- [ ] **Step 3: Create .gitignore**

```gitignore
# Build artifacts
goose-notify
goose-notify-*
test-*

# OS junk
.DS_Store

# Editor
.vscode/
.idea/
*.swp
```

- [ ] **Step 4: Copy fonts from goose-launcher**

```bash
mkdir -p pkg/ui/fonts
cp ~/gt/goose-launcher/pkg/ui/fonts/JetBrainsMono-Regular.ttf pkg/ui/fonts/
cp ~/gt/goose-launcher/pkg/ui/fonts/JetBrainsMono-Bold.ttf pkg/ui/fonts/
```

- [ ] **Step 5: Create Makefile**

```makefile
.PHONY: build build-macos install test clean

build:
	go build -o goose-notify ./cmd/goose-notify

build-macos:
	GOOS=darwin GOARCH=arm64 go build -o goose-notify-arm64 ./cmd/goose-notify
	GOOS=darwin GOARCH=amd64 go build -o goose-notify-amd64 ./cmd/goose-notify
	lipo -create -output goose-notify goose-notify-arm64 goose-notify-amd64
	rm goose-notify-arm64 goose-notify-amd64

install: build-macos
	cp goose-notify /usr/local/bin/

test:
	go test -v ./...

clean:
	rm -f goose-notify goose-notify-arm64 goose-notify-amd64
```

- [ ] **Step 6: Create minimal README.md**

```markdown
# goose-notify

macOS CLI that displays a black, fading notification toast near the top-center of the active screen, then exits.

## Usage

```
goose-notify "your message"
echo "your message" | goose-notify
goose-notify --title "Status" --duration 3s "Build succeeded"
```

## Build

```bash
make build         # build for host arch
make install       # universal binary to /usr/local/bin
make test
```

See [docs/superpowers/specs/2026-05-04-goose-notify-design.md](docs/superpowers/specs/2026-05-04-goose-notify-design.md) for the full design.
```

- [ ] **Step 7: Verify it builds (empty)**

```bash
go build ./... 2>&1 || echo "no Go files yet — expected"
```

Expected: "no Go files in ..." or similar; no error from go.mod itself.

- [ ] **Step 8: Commit**

```bash
git add go.mod go.sum .gitignore Makefile README.md pkg/ui/fonts/
git commit -m "chore: project bootstrap (go.mod, Makefile, fonts)"
```

---

## Task 2: pkg/config — flag parsing

**Files:**
- Create: `pkg/config/config.go`
- Create: `pkg/config/config_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/config/config_test.go`:

```go
package config

import (
	"testing"
	"time"
)

func TestParseDefaults(t *testing.T) {
	cfg, msg, err := Parse([]string{"hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "hello" {
		t.Errorf("msg = %q; want %q", msg, "hello")
	}
	if cfg.Title != "" {
		t.Errorf("Title = %q; want empty", cfg.Title)
	}
	if cfg.Duration != 2*time.Second {
		t.Errorf("Duration = %v; want 2s", cfg.Duration)
	}
	if cfg.FadeIn != 200*time.Millisecond {
		t.Errorf("FadeIn = %v; want 200ms", cfg.FadeIn)
	}
	if cfg.FadeOut != 300*time.Millisecond {
		t.Errorf("FadeOut = %v; want 300ms", cfg.FadeOut)
	}
	if cfg.MaxWidth != 600 {
		t.Errorf("MaxWidth = %d; want 600", cfg.MaxWidth)
	}
	if cfg.OffsetY != 80 {
		t.Errorf("OffsetY = %d; want 80", cfg.OffsetY)
	}
}

func TestParseFlags(t *testing.T) {
	cfg, msg, err := Parse([]string{
		"--title", "Status",
		"--duration", "3s",
		"--fade-in", "100ms",
		"--fade-out", "150ms",
		"--max-width", "800",
		"--offset-y", "50",
		"hello world",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "hello world" {
		t.Errorf("msg = %q; want %q", msg, "hello world")
	}
	if cfg.Title != "Status" {
		t.Errorf("Title = %q; want %q", cfg.Title, "Status")
	}
	if cfg.Duration != 3*time.Second {
		t.Errorf("Duration = %v; want 3s", cfg.Duration)
	}
	if cfg.FadeIn != 100*time.Millisecond {
		t.Errorf("FadeIn = %v; want 100ms", cfg.FadeIn)
	}
	if cfg.FadeOut != 150*time.Millisecond {
		t.Errorf("FadeOut = %v; want 150ms", cfg.FadeOut)
	}
	if cfg.MaxWidth != 800 {
		t.Errorf("MaxWidth = %d; want 800", cfg.MaxWidth)
	}
	if cfg.OffsetY != 50 {
		t.Errorf("OffsetY = %d; want 50", cfg.OffsetY)
	}
}

func TestParseNoMessage(t *testing.T) {
	_, msg, err := Parse([]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg != "" {
		t.Errorf("msg = %q; want empty (so caller can fall back to stdin)", msg)
	}
}

func TestParseRejectsExtraArgs(t *testing.T) {
	_, _, err := Parse([]string{"hello", "world"})
	if err == nil {
		t.Fatal("expected error for multiple positional args, got nil")
	}
}

func TestParseBadDuration(t *testing.T) {
	_, _, err := Parse([]string{"--duration", "two seconds", "hello"})
	if err == nil {
		t.Fatal("expected error for bad duration, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/config/`
Expected: FAIL with `package config has no Go files` or `undefined: Parse`.

- [ ] **Step 3: Implement Parse**

Create `pkg/config/config.go`:

```go
// Package config parses goose-notify's CLI flags.
package config

import (
	"flag"
	"fmt"
	"io"
	"time"
)

type Config struct {
	Title    string
	Duration time.Duration
	FadeIn   time.Duration
	FadeOut  time.Duration
	MaxWidth int
	OffsetY  int
}

// Parse parses argv (without the program name) and returns the config plus
// the optional positional message. If the message is missing, msg == "" and
// callers should fall back to stdin.
//
// Errors include flag parse errors and "more than one positional arg".
func Parse(args []string) (Config, string, error) {
	fs := flag.NewFlagSet("goose-notify", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // Errors are returned, not printed by the FlagSet.

	cfg := Config{
		Duration: 2 * time.Second,
		FadeIn:   200 * time.Millisecond,
		FadeOut:  300 * time.Millisecond,
		MaxWidth: 600,
		OffsetY:  80,
	}
	fs.StringVar(&cfg.Title, "title", "", "bold first line above body")
	fs.DurationVar(&cfg.Duration, "duration", cfg.Duration, "how long the toast stays fully visible")
	fs.DurationVar(&cfg.FadeIn, "fade-in", cfg.FadeIn, "fade-in animation duration")
	fs.DurationVar(&cfg.FadeOut, "fade-out", cfg.FadeOut, "fade-out animation duration")
	fs.IntVar(&cfg.MaxWidth, "max-width", cfg.MaxWidth, "soft cap on toast width in pixels")
	fs.IntVar(&cfg.OffsetY, "offset-y", cfg.OffsetY, "vertical offset from top of active screen (px)")

	if err := fs.Parse(args); err != nil {
		return Config{}, "", err
	}

	rest := fs.Args()
	switch len(rest) {
	case 0:
		return cfg, "", nil
	case 1:
		return cfg, rest[0], nil
	default:
		return Config{}, "", fmt.Errorf("expected at most one positional message argument, got %d", len(rest))
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/config/`
Expected: PASS for all five tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/config/
git commit -m "feat(config): flag parsing with defaults"
```

---

## Task 3: pkg/input — message resolution

**Files:**
- Create: `pkg/input/input.go`
- Create: `pkg/input/input_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/input/input_test.go`:

```go
package input

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveArgWins(t *testing.T) {
	got, err := Resolve("from-arg", strings.NewReader("from-stdin\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "from-arg" {
		t.Errorf("got %q; want %q", got, "from-arg")
	}
}

func TestResolveStdinFallback(t *testing.T) {
	got, err := Resolve("", strings.NewReader("hello stdin\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "hello stdin" {
		t.Errorf("got %q; want %q (trailing newline trimmed)", got, "hello stdin")
	}
}

func TestResolveStdinPreservesInternalNewlines(t *testing.T) {
	got, err := Resolve("", strings.NewReader("line1\nline2\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "line1\nline2" {
		t.Errorf("got %q; want %q (only trailing newline trimmed)", got, "line1\nline2")
	}
}

func TestResolveEmptyArgEmptyStdin(t *testing.T) {
	_, err := Resolve("", strings.NewReader(""))
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("got err %v; want ErrEmpty", err)
	}
}

func TestResolveEmptyArgWhitespaceStdin(t *testing.T) {
	_, err := Resolve("", strings.NewReader("   \n\n"))
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("got err %v; want ErrEmpty (whitespace-only is empty)", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/input/`
Expected: FAIL with `undefined: Resolve` / `undefined: ErrEmpty`.

- [ ] **Step 3: Implement Resolve**

Create `pkg/input/input.go`:

```go
// Package input resolves the toast message from either a positional CLI
// argument (preferred) or stdin (fallback).
package input

import (
	"errors"
	"io"
	"strings"
)

// ErrEmpty is returned when no message was provided via arg or stdin.
var ErrEmpty = errors.New("empty message")

// Resolve returns the toast message. If arg is non-empty it wins; otherwise
// the entire stdin is read and its trailing newline is trimmed. Returns
// ErrEmpty if both are empty (after trim).
func Resolve(arg string, stdin io.Reader) (string, error) {
	if arg != "" {
		return arg, nil
	}
	b, err := io.ReadAll(stdin)
	if err != nil {
		return "", err
	}
	s := strings.TrimRight(string(b), "\n")
	if strings.TrimSpace(s) == "" {
		return "", ErrEmpty
	}
	return s, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/input/`
Expected: PASS for all five tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/input/
git commit -m "feat(input): resolve message from arg (preferred) or stdin"
```

---

## Task 4: pkg/ui — easing functions

**Files:**
- Create: `pkg/ui/easing.go`
- Create: `pkg/ui/easing_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/ui/easing_test.go`:

```go
package ui

import (
	"math"
	"testing"
)

func nearlyEqual(a, b, tol float64) bool {
	return math.Abs(a-b) < tol
}

func TestEaseOutCubicEndpoints(t *testing.T) {
	if got := EaseOutCubic(0); !nearlyEqual(got, 0, 1e-9) {
		t.Errorf("EaseOutCubic(0) = %v; want 0", got)
	}
	if got := EaseOutCubic(1); !nearlyEqual(got, 1, 1e-9) {
		t.Errorf("EaseOutCubic(1) = %v; want 1", got)
	}
}

func TestEaseOutCubicMidpoint(t *testing.T) {
	// 1 - (1-0.5)^3 = 1 - 0.125 = 0.875
	got := EaseOutCubic(0.5)
	if !nearlyEqual(got, 0.875, 1e-9) {
		t.Errorf("EaseOutCubic(0.5) = %v; want 0.875", got)
	}
}

func TestEaseInCubicEndpoints(t *testing.T) {
	if got := EaseInCubic(0); !nearlyEqual(got, 0, 1e-9) {
		t.Errorf("EaseInCubic(0) = %v; want 0", got)
	}
	if got := EaseInCubic(1); !nearlyEqual(got, 1, 1e-9) {
		t.Errorf("EaseInCubic(1) = %v; want 1", got)
	}
}

func TestEaseInCubicMidpoint(t *testing.T) {
	// 0.5^3 = 0.125
	got := EaseInCubic(0.5)
	if !nearlyEqual(got, 0.125, 1e-9) {
		t.Errorf("EaseInCubic(0.5) = %v; want 0.125", got)
	}
}

func TestEaseClampsOutOfRange(t *testing.T) {
	if got := EaseOutCubic(-0.5); !nearlyEqual(got, 0, 1e-9) {
		t.Errorf("EaseOutCubic(-0.5) = %v; want 0 (clamped)", got)
	}
	if got := EaseInCubic(1.5); !nearlyEqual(got, 1, 1e-9) {
		t.Errorf("EaseInCubic(1.5) = %v; want 1 (clamped)", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/ui/`
Expected: FAIL with `undefined: EaseOutCubic` / `undefined: EaseInCubic`.

- [ ] **Step 3: Implement easing**

Create `pkg/ui/easing.go`:

```go
package ui

// EaseOutCubic maps t in [0,1] to a curve that starts fast and decelerates
// toward 1. Inputs outside [0,1] clamp to the nearest endpoint.
func EaseOutCubic(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	u := 1 - t
	return 1 - u*u*u
}

// EaseInCubic maps t in [0,1] to a curve that starts slow and accelerates
// toward 1. Inputs outside [0,1] clamp to the nearest endpoint.
func EaseInCubic(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * t
}

// Opacity returns the toast's current alpha (0..1) given the elapsed time
// since fade-in started and the three timeline durations. The second return
// is true once the timeline is fully complete (fade-out is done).
func Opacity(elapsedNs, fadeInNs, holdNs, fadeOutNs int64) (alpha float64, done bool) {
	if elapsedNs < fadeInNs {
		return EaseOutCubic(float64(elapsedNs) / float64(fadeInNs)), false
	}
	holdEnd := fadeInNs + holdNs
	if elapsedNs < holdEnd {
		return 1, false
	}
	fadeOutEnd := holdEnd + fadeOutNs
	if elapsedNs < fadeOutEnd {
		t := float64(elapsedNs-holdEnd) / float64(fadeOutNs)
		return 1 - EaseInCubic(t), false
	}
	return 0, true
}
```

- [ ] **Step 4: Add Opacity tests to easing_test.go**

Append to `pkg/ui/easing_test.go`:

```go
func TestOpacityFadeInPhase(t *testing.T) {
	a, done := Opacity(100_000_000, 200_000_000, 1_000_000_000, 300_000_000) // 100ms in, 200ms fadeIn
	// at t=0.5 of fadeIn, EaseOutCubic(0.5) = 0.875
	if !nearlyEqual(a, 0.875, 1e-9) || done {
		t.Errorf("Opacity fade-in midpoint = (%v, %v); want (0.875, false)", a, done)
	}
}

func TestOpacityHoldPhase(t *testing.T) {
	// 500ms elapsed, fade-in 200ms, hold 1s — inside hold
	a, done := Opacity(500_000_000, 200_000_000, 1_000_000_000, 300_000_000)
	if !nearlyEqual(a, 1, 1e-9) || done {
		t.Errorf("Opacity hold = (%v, %v); want (1, false)", a, done)
	}
}

func TestOpacityFadeOutPhase(t *testing.T) {
	// 1.35s elapsed = fadeIn(0.2) + hold(1.0) + 150ms = halfway through 300ms fadeOut
	a, done := Opacity(1_350_000_000, 200_000_000, 1_000_000_000, 300_000_000)
	// 1 - EaseInCubic(0.5) = 1 - 0.125 = 0.875
	if !nearlyEqual(a, 0.875, 1e-9) || done {
		t.Errorf("Opacity fade-out midpoint = (%v, %v); want (0.875, false)", a, done)
	}
}

func TestOpacityDone(t *testing.T) {
	// 2s elapsed > fadeIn+hold+fadeOut = 1.5s
	a, done := Opacity(2_000_000_000, 200_000_000, 1_000_000_000, 300_000_000)
	if !nearlyEqual(a, 0, 1e-9) || !done {
		t.Errorf("Opacity done = (%v, %v); want (0, true)", a, done)
	}
}
```

- [ ] **Step 5: Run all easing tests**

Run: `go test -v ./pkg/ui/ -run "TestEase|TestOpacity"`
Expected: PASS for all 9 tests.

- [ ] **Step 6: Commit**

```bash
git add pkg/ui/easing.go pkg/ui/easing_test.go
git commit -m "feat(ui): cubic easing and opacity timeline"
```

---

## Task 5: pkg/ui — layout math

**Files:**
- Create: `pkg/ui/layout.go`
- Create: `pkg/ui/layout_test.go`

This computes the toast's outer dimensions from text content, max width, and padding. Pure math — no Gio.

- [ ] **Step 1: Write the failing test**

Create `pkg/ui/layout_test.go`:

```go
package ui

import "testing"

func TestBoxSizeShortMessageNoTitle(t *testing.T) {
	m := Metrics{
		BodyCharWidth:    8,   // px per char
		BodyLineHeight:   18,  // px
		TitleCharWidth:   9,
		TitleLineHeight:  20,
		PaddingX:         28,
		PaddingY:         20,
		TitleBodyGap:     6,
		MaxWidth:         600,
		MinWidth:         200,
	}
	w, h := BoxSize("hello", "", m)
	// "hello" width = 40px content + 56 padding = 96, but MinWidth=200 wins
	if w != 200 {
		t.Errorf("width = %d; want 200 (min width)", w)
	}
	// 1 body line: 18 + 40 padding = 58
	if h != 58 {
		t.Errorf("height = %d; want 58", h)
	}
}

func TestBoxSizeLongMessageWraps(t *testing.T) {
	m := Metrics{
		BodyCharWidth:    8,
		BodyLineHeight:   18,
		TitleCharWidth:   9,
		TitleLineHeight:  20,
		PaddingX:         28,
		PaddingY:         20,
		TitleBodyGap:     6,
		MaxWidth:         200,
		MinWidth:         100,
	}
	// "abcdefghijklmnopqrst" = 20 chars = 160px content; max content width = 200-56 = 144 → wraps to 2 lines
	w, h := BoxSize("abcdefghijklmnopqrst", "", m)
	if w != 200 {
		t.Errorf("width = %d; want 200 (clamped to max)", w)
	}
	// 2 body lines = 36 + 40 padding = 76
	if h != 76 {
		t.Errorf("height = %d; want 76", h)
	}
}

func TestBoxSizeWithTitle(t *testing.T) {
	m := Metrics{
		BodyCharWidth:    8,
		BodyLineHeight:   18,
		TitleCharWidth:   9,
		TitleLineHeight:  20,
		PaddingX:         28,
		PaddingY:         20,
		TitleBodyGap:     6,
		MaxWidth:         600,
		MinWidth:         200,
	}
	w, h := BoxSize("body line", "Title", m)
	// width: title 5*9=45, body 9*8=72, max=72 + 56 padding = 128 < 200 min → 200
	if w != 200 {
		t.Errorf("width = %d; want 200", w)
	}
	// title 20 + gap 6 + body 18 = 44 + 40 padding = 84
	if h != 84 {
		t.Errorf("height = %d; want 84", h)
	}
}

func TestBoxSizeMultilineBody(t *testing.T) {
	m := Metrics{
		BodyCharWidth:    8,
		BodyLineHeight:   18,
		TitleCharWidth:   9,
		TitleLineHeight:  20,
		PaddingX:         28,
		PaddingY:         20,
		TitleBodyGap:     6,
		MaxWidth:         600,
		MinWidth:         200,
	}
	_, h := BoxSize("line1\nline2\nline3", "", m)
	// 3 body lines = 54 + 40 padding = 94
	if h != 94 {
		t.Errorf("height = %d; want 94", h)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/ui/ -run TestBoxSize`
Expected: FAIL with `undefined: Metrics` / `undefined: BoxSize`.

- [ ] **Step 3: Implement BoxSize**

Create `pkg/ui/layout.go`:

```go
package ui

import "strings"

// Metrics carries the per-pixel typography numbers BoxSize needs. They're
// approximate (we use char-cell widths rather than full text shaping) — good
// enough for sizing the toast box. JetBrains Mono is monospace so a single
// CharWidth per face is accurate.
type Metrics struct {
	BodyCharWidth   int
	BodyLineHeight  int
	TitleCharWidth  int
	TitleLineHeight int
	PaddingX        int
	PaddingY        int
	TitleBodyGap    int
	MaxWidth        int
	MinWidth        int
}

// BoxSize returns the outer (width, height) of the toast given body and
// optional title text. Width is clamped to [MinWidth, MaxWidth]. Height is
// computed from wrapped line counts.
func BoxSize(body, title string, m Metrics) (width, height int) {
	contentMax := m.MaxWidth - 2*m.PaddingX
	if contentMax < 1 {
		contentMax = 1
	}

	// Body: split on \n, then wrap each line.
	bodyLines := 0
	bodyMaxPx := 0
	for _, line := range strings.Split(body, "\n") {
		linePx := len(line) * m.BodyCharWidth
		if linePx > bodyMaxPx {
			bodyMaxPx = linePx
		}
		// Wrap count: how many contentMax-wide rows does linePx need?
		wrapped := (linePx + contentMax - 1) / contentMax
		if wrapped < 1 {
			wrapped = 1
		}
		bodyLines += wrapped
	}

	titleLines := 0
	titleMaxPx := 0
	if title != "" {
		titlePx := len(title) * m.TitleCharWidth
		titleMaxPx = titlePx
		wrapped := (titlePx + contentMax - 1) / contentMax
		if wrapped < 1 {
			wrapped = 1
		}
		titleLines = wrapped
	}

	// Width = max(bodyContentPx, titleContentPx) clamped + padding.
	contentPx := bodyMaxPx
	if titleMaxPx > contentPx {
		contentPx = titleMaxPx
	}
	if contentPx > contentMax {
		contentPx = contentMax
	}
	width = contentPx + 2*m.PaddingX
	if width < m.MinWidth {
		width = m.MinWidth
	}
	if width > m.MaxWidth {
		width = m.MaxWidth
	}

	// Height = title block + gap (if title) + body block + padding.
	height = bodyLines*m.BodyLineHeight + 2*m.PaddingY
	if titleLines > 0 {
		height += titleLines*m.TitleLineHeight + m.TitleBodyGap
	}

	return width, height
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./pkg/ui/ -run TestBoxSize`
Expected: PASS for all 4 tests.

- [ ] **Step 5: Commit**

```bash
git add pkg/ui/layout.go pkg/ui/layout_test.go
git commit -m "feat(ui): box sizing math for toast layout"
```

---

## Task 6: pkg/fontcache — port from goose-launcher

**Files:**
- Create: `pkg/fontcache/fontcache.go`

This is a near-verbatim copy from `~/gt/goose-launcher/pkg/fontcache/fontcache.go`. Same package, same API. We use only Regular + Bold; the italic/emoji parameters remain (with nil) so the signature stays compatible if we ever lift more code over.

- [ ] **Step 1: Copy the file**

```bash
cp ~/gt/goose-launcher/pkg/fontcache/fontcache.go pkg/fontcache/fontcache.go
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./pkg/fontcache/`
Expected: success, no output.

- [ ] **Step 3: Commit**

```bash
git add pkg/fontcache/
git commit -m "feat(fontcache): port from goose-launcher (Regular+Bold only used)"
```

---

## Task 7: pkg/macwin — cgo NSWindow shim

**Files:**
- Create: `pkg/macwin/macwin.go`
- Create: `pkg/macwin/macwin.m`

This shim does three jobs:
1. Set the process to `NSApplicationActivationPolicyAccessory` so no Dock icon appears.
2. Find the Gio NSWindow by title (it's created lazily after Gio's first `FrameEvent`).
3. Apply toast flags (transparent, click-through, top-level, all-spaces) and position it top-center on the active screen.

Different surface from goose-launcher's macwin (no daemon lifecycle), same cgo pattern.

- [ ] **Step 1: Create the Obj-C side**

Create `pkg/macwin/macwin.m`:

```objc
// macwin: AppKit shim that configures the Gio-managed NSWindow to behave as
// a non-interactive toast — borderless, transparent, click-through, never
// key, top-level, all-spaces — and positions it on the active screen.
//
// All AppKit calls dispatch onto the main thread because NSWindow methods
// are not thread-safe. Gio's app.Main() owns the main thread, so
// dispatch_sync from a Go goroutine is safe.

#import <Cocoa/Cocoa.h>

// macwin_findWindowByTitle returns a retained pointer to the first NSWindow
// in [NSApp windows] whose title matches titleC. Returns NULL if none.
// Caller must macwin_releaseWindow when done.
void* macwin_findWindowByTitle(const char *titleC) {
    __block void *result = NULL;
    NSString *want = [NSString stringWithUTF8String:titleC];
    dispatch_sync(dispatch_get_main_queue(), ^{
        for (NSWindow *w in [NSApp windows]) {
            if (w == nil) continue;
            if ([[w title] isEqualToString:want]) {
                result = (void *)CFBridgingRetain(w);
                break;
            }
        }
    });
    return result;
}

void macwin_releaseWindow(void *win) {
    if (win != NULL) {
        CFRelease(win);
    }
}

// macwin_setAccessoryPolicy switches the running app to
// NSApplicationActivationPolicyAccessory — no Dock icon, no menu bar by
// default. Call early, before any window is shown.
void macwin_setAccessoryPolicy(void) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}

// Find the screen that contains the cursor. Falls back to mainScreen.
static NSScreen* screenAtCursor(void) {
    NSPoint p = [NSEvent mouseLocation];
    for (NSScreen *s in [NSScreen screens]) {
        if (NSPointInRect(p, [s frame])) {
            return s;
        }
    }
    return [NSScreen mainScreen];
}

// macwin_configureToast applies the toast NSWindow flags AND positions the
// window at top-center of the active screen with vertical offset offsetY
// from the visible-frame top. width/height are in pixels.
//
// Sets:
//   - opaque NO + clearColor (transparent surface)
//   - level NSStatusWindowLevel (above app windows)
//   - ignoresMouseEvents YES (clicks pass through)
//   - hidesOnDeactivate NO
//   - collectionBehavior CanJoinAllSpaces | FullScreenAuxiliary | Stationary | IgnoresCycle
//
// Then orderFrontRegardless to show without stealing key/focus.
void macwin_configureToast(void *win, int width, int height, int offsetY) {
    if (win == NULL) return;
    NSWindow *w = (__bridge NSWindow *)win;
    dispatch_sync(dispatch_get_main_queue(), ^{
        [w setOpaque:NO];
        [w setBackgroundColor:[NSColor clearColor]];
        [w setHasShadow:NO];
        [w setLevel:NSStatusWindowLevel];
        [w setIgnoresMouseEvents:YES];
        [w setHidesOnDeactivate:NO];
        [w setCollectionBehavior:
            NSWindowCollectionBehaviorCanJoinAllSpaces |
            NSWindowCollectionBehaviorFullScreenAuxiliary |
            NSWindowCollectionBehaviorStationary |
            NSWindowCollectionBehaviorIgnoresCycle];

        NSScreen *s = screenAtCursor();
        NSRect vf = [s visibleFrame];
        NSRect target;
        target.size = NSMakeSize((CGFloat)width, (CGFloat)height);
        target.origin.x = vf.origin.x + (vf.size.width - (CGFloat)width) / 2.0;
        // NSWindow coords are bottom-left origin; we want offsetY from the top.
        target.origin.y = vf.origin.y + vf.size.height - (CGFloat)offsetY - (CGFloat)height;
        [w setFrame:target display:YES];

        [w orderFrontRegardless];
    });
}
```

- [ ] **Step 2: Create the Go side**

Create `pkg/macwin/macwin.go`:

```go
// Package macwin is the AppKit shim for goose-notify. It configures the
// Gio-managed NSWindow to behave as a non-interactive overlay toast and
// positions it on the active screen.
//
// macOS-only by design.
//
//go:build darwin

package macwin

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>

void* macwin_findWindowByTitle(const char *titleC);
void  macwin_releaseWindow(void *win);
void  macwin_setAccessoryPolicy(void);
void  macwin_configureToast(void *win, int width, int height, int offsetY);
*/
import "C"

import (
	"errors"
	"time"
	"unsafe"
)

// SetAccessoryPolicy switches the process to
// NSApplicationActivationPolicyAccessory: no Dock icon, no menu bar.
// Overrides Gio's hardcoded Regular policy. Call once at startup.
func SetAccessoryPolicy() {
	C.macwin_setAccessoryPolicy()
}

// ConfigureToast finds the Gio NSWindow by title and applies toast flags +
// positioning. width/height are in points (the NSWindow frame size, not
// backing pixels). offsetY is the gap from the active screen's visible-top.
//
// Polls because Gio creates the NSWindow lazily after the first FrameEvent.
// Returns an error if no window with the given title appears within timeout.
func ConfigureToast(title string, width, height, offsetY int, timeout time.Duration) error {
	cTitle := C.CString(title)
	defer C.free(unsafe.Pointer(cTitle))

	deadline := time.Now().Add(timeout)
	var ptr unsafe.Pointer
	for {
		ptr = C.macwin_findWindowByTitle(cTitle)
		if ptr != nil {
			break
		}
		if time.Now().After(deadline) {
			return errors.New("macwin: no window with matching title within timeout")
		}
		time.Sleep(5 * time.Millisecond)
	}
	defer C.macwin_releaseWindow(ptr)

	C.macwin_configureToast(ptr, C.int(width), C.int(height), C.int(offsetY))
	return nil
}
```

- [ ] **Step 3: Verify it compiles**

Run: `go build ./pkg/macwin/`
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add pkg/macwin/
git commit -m "feat(macwin): cgo shim — configure NSWindow as toast overlay"
```

---

## Task 8: pkg/ui/toast.go — Gio window driver and animation

**Files:**
- Create: `pkg/ui/toast.go`

This file owns the Gio event loop and the per-frame paint of the rounded black rect with title + body, multiplied by current opacity. No unit test — visual behavior is verified in Task 10's smoke script.

- [ ] **Step 1: Create the toast driver**

Create `pkg/ui/toast.go`:

```go
package ui

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget/material"

	"github.com/sam33r/goose-notify/pkg/fontcache"
	"github.com/sam33r/goose-notify/pkg/macwin"
)

// WindowTitle is the NSWindow title used to find the window from macwin.
// Must be unique enough that other apps' windows won't collide.
const WindowTitle = "goose-notify-toast"

//go:embed fonts/JetBrainsMono-Regular.ttf
var fontRegular []byte

//go:embed fonts/JetBrainsMono-Bold.ttf
var fontBold []byte

// Toast configures the timeline and content for one toast invocation.
type Toast struct {
	Title    string
	Body     string
	Duration time.Duration
	FadeIn   time.Duration
	FadeOut  time.Duration
	MaxWidth int
	OffsetY  int
}

// Run displays the toast and blocks until the animation completes, then
// returns. Caller invokes app.Main() on the OS main thread.
func Run(t Toast) error {
	// Approximate metrics for box sizing. Real text measurement happens at
	// paint time via Gio's shaper; these numbers just need to be close enough
	// that the NSWindow frame fits the painted content. They're conservative
	// (slightly over-large) so wrapping doesn't get cut off.
	metrics := Metrics{
		BodyCharWidth:   8,
		BodyLineHeight:  18,
		TitleCharWidth:  9,
		TitleLineHeight: 20,
		PaddingX:        28,
		PaddingY:        20,
		TitleBodyGap:    6,
		MaxWidth:        t.MaxWidth,
		MinWidth:        200,
	}
	width, height := BoxSize(t.Body, t.Title, metrics)

	w := new(app.Window)
	w.Option(
		app.Title(WindowTitle),
		app.Decorated(false),
		app.Size(unit.Dp(width), unit.Dp(height)),
		app.MinSize(unit.Dp(width), unit.Dp(height)),
		app.MaxSize(unit.Dp(width), unit.Dp(height)),
	)

	macwin.SetAccessoryPolicy()

	// Configure the NSWindow once it appears. Runs in a goroutine because
	// FindWindowByTitle polls and we want the Gio event loop to start
	// creating frames in parallel.
	go func() {
		if err := macwin.ConfigureToast(WindowTitle, width, height, t.OffsetY, time.Second); err != nil {
			fmt.Fprintf(os.Stderr, "goose-notify: %v\n", err)
			os.Exit(1)
		}
	}()

	theme, err := buildTheme()
	if err != nil {
		return err
	}

	startTime := time.Now()
	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			elapsedNs := time.Since(startTime).Nanoseconds()
			alpha, done := Opacity(elapsedNs, t.FadeIn.Nanoseconds(), t.Duration.Nanoseconds(), t.FadeOut.Nanoseconds())

			paintToast(gtx, theme, t.Title, t.Body, alpha)

			if done {
				e.Frame(gtx.Ops)
				// Animation complete — exit. Defer to let this final frame flush.
				go func() {
					time.Sleep(20 * time.Millisecond)
					os.Exit(0)
				}()
			} else {
				gtx.Execute(op.InvalidateCmd{At: time.Now().Add(8 * time.Millisecond)})
				e.Frame(gtx.Ops)
			}
		}
	}
}

func buildTheme() (*material.Theme, error) {
	regular, bold, _, _, err := fontcache.GetFonts(fontRegular, fontBold, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("load fonts: %w", err)
	}
	th := material.NewTheme()
	th.Shaper = text.NewShaper(text.WithCollection([]font.FontFace{
		{Font: font.Font{Typeface: "JetBrains Mono"}, Face: regular},
		{Font: font.Font{Typeface: "JetBrains Mono", Weight: font.Bold}, Face: bold},
	}))
	return th, nil
}

// paintToast draws the rounded black rect, title (if any), and body, all
// multiplied by alpha (0..1).
func paintToast(gtx layout.Context, th *material.Theme, title, body string, alpha float64) {
	// Background: black at 0.92 base opacity, multiplied by current alpha.
	bgA := uint8(255.0 * 0.92 * alpha)
	bg := color.NRGBA{R: 0, G: 0, B: 0, A: bgA}

	// Whole-window rounded rect.
	bounds := image.Rectangle{Max: gtx.Constraints.Max}
	rrect := clip.RRect{Rect: bounds, NE: 16, NW: 16, SE: 16, SW: 16}
	paint.FillShape(gtx.Ops, bg, rrect.Op(gtx.Ops))

	// Text colors at current alpha.
	titleA := uint8(255.0 * alpha)
	bodyA := uint8(255.0 * 0.90 * alpha)
	titleColor := color.NRGBA{R: 255, G: 255, B: 255, A: titleA}
	bodyColor := color.NRGBA{R: 255, G: 255, B: 255, A: bodyA}

	inset := layout.UniformInset(unit.Dp(20))
	inset.Left = unit.Dp(28)
	inset.Right = unit.Dp(28)

	inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceEnd}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if title == "" {
					return layout.Dimensions{}
				}
				lbl := material.Label(th, unit.Sp(14), title)
				lbl.Color = titleColor
				lbl.Font.Typeface = "JetBrains Mono"
				lbl.Font.Weight = font.Bold
				dims := lbl.Layout(gtx)
				dims.Size.Y += gtx.Dp(unit.Dp(6)) // gap below title
				return dims
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				lbl := material.Label(th, unit.Sp(13), body)
				lbl.Color = bodyColor
				lbl.Font.Typeface = "JetBrains Mono"
				return lbl.Layout(gtx)
			}),
		)
	})
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./pkg/ui/`
Expected: success. If Gio API names differ in v0.9.0 (e.g. `op.InvalidateCmd` may be `op.InvalidateOp` — check goose-launcher's `pkg/ui/window.go` for the exact API in use), adjust to match. Reference: `~/gt/goose-launcher/pkg/ui/window.go`.

- [ ] **Step 3: Commit**

```bash
git add pkg/ui/toast.go
git commit -m "feat(ui): Gio toast window with fade-in/hold/fade-out timeline"
```

---

## Task 9: cmd/goose-notify/main.go — wire it together

**Files:**
- Create: `cmd/goose-notify/main.go`

- [ ] **Step 1: Create main.go**

Create `cmd/goose-notify/main.go`:

```go
// Command goose-notify displays a black, fading notification toast at the
// top-center of the active macOS screen, then exits.
package main

import (
	"errors"
	"fmt"
	"os"

	"gioui.org/app"

	"github.com/sam33r/goose-notify/pkg/config"
	"github.com/sam33r/goose-notify/pkg/input"
	"github.com/sam33r/goose-notify/pkg/ui"
)

func main() {
	cfg, argMsg, err := config.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "goose-notify: %v\n", err)
		os.Exit(1)
	}

	msg, err := input.Resolve(argMsg, os.Stdin)
	if err != nil {
		if errors.Is(err, input.ErrEmpty) {
			fmt.Fprintln(os.Stderr, "goose-notify: empty message")
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "goose-notify: read input: %v\n", err)
		os.Exit(1)
	}

	t := ui.Toast{
		Title:    cfg.Title,
		Body:     msg,
		Duration: cfg.Duration,
		FadeIn:   cfg.FadeIn,
		FadeOut:  cfg.FadeOut,
		MaxWidth: cfg.MaxWidth,
		OffsetY:  cfg.OffsetY,
	}

	// Same threading pattern as goose-launcher: UI on a goroutine, app.Main()
	// holds the main OS thread (required by Gio on macOS).
	go func() {
		if err := ui.Run(t); err != nil {
			fmt.Fprintf(os.Stderr, "goose-notify: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
}
```

- [ ] **Step 2: Build the full binary**

Run: `make build`
Expected: produces `./goose-notify` with no errors.

- [ ] **Step 3: Smoke test with arg**

Run: `./goose-notify "hello from goose-notify"`
Expected: A black rounded-rect toast appears top-center of the active screen, fades in over ~200ms, holds ~2s, fades out ~300ms, process exits 0. Other windows remain interactive while it's visible. No Dock icon appears.

- [ ] **Step 4: Smoke test with stdin**

Run: `echo "from stdin" | ./goose-notify`
Expected: Same as above, message reads "from stdin".

- [ ] **Step 5: Smoke test with title**

Run: `./goose-notify --title "Build" "succeeded in 4.3s"`
Expected: Bold "Build" line above "succeeded in 4.3s".

- [ ] **Step 6: Smoke test error paths**

Run: `./goose-notify`
Expected: stderr "goose-notify: empty message", exit code 1.

Run: `./goose-notify a b`
Expected: stderr error about multiple positional args, exit code 1.

- [ ] **Step 7: Commit**

```bash
git add cmd/
git commit -m "feat(cmd): wire config + input + ui into goose-notify binary"
```

---

## Task 10: test-notify.sh — integration smoke

**Files:**
- Create: `test-notify.sh`

- [ ] **Step 1: Create the script**

Create `test-notify.sh`:

```bash
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
```

- [ ] **Step 2: Make it executable**

```bash
chmod +x test-notify.sh
```

- [ ] **Step 3: Run it**

Run: `./test-notify.sh`
Expected: Five toasts appear in sequence at top-center of the active screen. Confirm visually that:
- Each fades in and out smoothly
- Position is top-center, ~80px below the menu bar
- You can keep clicking/typing in other windows while the toast is visible
- No Dock icon ever appears
- The toast appears above fullscreen apps if you trigger it from one

- [ ] **Step 4: Commit**

```bash
git add test-notify.sh
git commit -m "chore: add test-notify.sh integration smoke"
```

---

## Self-review

(This section is for the plan author, not the executor.)

**Spec coverage** — checked each section of the spec against the plan:
- CLI surface (Section 2 of spec) → Task 2 implements all flags + defaults; Task 9 wires the empty-message and multi-arg errors.
- Visual (Section 3) → Task 5 sizes the box; Task 8 paints rounded rect, title, body with the spec's colors and padding.
- Animation (Section 3) → Task 4 implements easing + opacity timeline math; Task 8 drives it per frame.
- Position (Section 3) → Task 7's `macwin_configureToast` reads `[NSEvent mouseLocation]` and frames the window top-anchored with `--offset-y` from `visibleFrame` top.
- Window/macOS behavior (Section 4) → Task 7 applies every NSWindow flag listed in the spec; Task 8 sets `app.Decorated(false)` for borderless; Task 7's `SetAccessoryPolicy` removes the Dock icon.
- Threading (Section 4) → Task 9 puts UI on a goroutine and calls `app.Main()` on the main thread.
- Concurrency (Section 4) → no shared state, each invocation is one process. No special handling needed.
- Errors (Section 5) → Task 9 covers empty message, multi-arg; Task 8 covers macwin timeout.
- Testing (Section 5) → Tasks 2, 3, 4, 5 are TDD; Task 10 is the integration smoke.
- Out-of-scope items → none implemented (correct).

**Placeholder scan:** searched for "TBD", "later", "appropriate" — none in the plan.

**Type consistency:** `ui.Toast` struct in Task 8 matches what main.go constructs in Task 9. `config.Config` field names match what main.go reads. `macwin.ConfigureToast` signature in Task 7 matches the call site in Task 8.

One known soft spot: Gio v0.9 API names like `op.InvalidateCmd` and `gtx.Execute` may differ slightly from earlier/later Gio versions. Task 8 Step 2 explicitly directs the implementer to cross-check `~/gt/goose-launcher/pkg/ui/window.go` for the actual API in use and adjust. This is a known constraint, not a placeholder.
