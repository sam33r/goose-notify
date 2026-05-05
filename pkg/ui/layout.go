package ui

import "strings"

// Metrics carries the per-pixel typography numbers BoxSize needs. They're
// approximate (we use char-cell widths rather than full text shaping). Good
// enough for sizing the toast box; JetBrains Mono is monospace so a single
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

	bodyLines := 0
	bodyMaxPx := 0
	for _, line := range strings.Split(body, "\n") {
		linePx := len(line) * m.BodyCharWidth
		if linePx > bodyMaxPx {
			bodyMaxPx = linePx
		}
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

	height = bodyLines*m.BodyLineHeight + 2*m.PaddingY
	if titleLines > 0 {
		height += titleLines*m.TitleLineHeight + m.TitleBodyGap
	}

	return width, height
}
