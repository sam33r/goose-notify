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

const WindowTitle = "goose-notify-toast"

//go:embed fonts/JetBrainsMono-Regular.ttf
var fontRegular []byte

//go:embed fonts/JetBrainsMono-Bold.ttf
var fontBold []byte

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

	// Find + configure the window once Gio creates it, then drive the fade
	// via NSWindow.alphaValue from a ticker. This bypasses Gio's opaque
	// framebuffer — the window's alpha controls compositing directly.
	startTime := time.Now()
	go func() {
		handle, err := macwin.ConfigureToast(WindowTitle, width, height, t.OffsetY, time.Second)
		if err != nil {
			fmt.Fprintf(os.Stderr, "goose-notify: %v\n", err)
			os.Exit(1)
		}
		defer handle.Free()

		ticker := time.NewTicker(8 * time.Millisecond)
		defer ticker.Stop()
		for {
			elapsedNs := time.Since(startTime).Nanoseconds()
			alpha, done := Opacity(elapsedNs, t.FadeIn.Nanoseconds(), t.Duration.Nanoseconds(), t.FadeOut.Nanoseconds())
			handle.SetAlpha(alpha)
			if done {
				time.Sleep(20 * time.Millisecond) // let the last frame flush
				os.Exit(0)
			}
			<-ticker.C
		}
	}()

	theme, err := buildTheme()
	if err != nil {
		return err
	}

	var ops op.Ops
	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			paintToast(gtx, theme, t.Title, t.Body)
			e.Frame(&ops)
		}
	}
}

func buildTheme() (*material.Theme, error) {
	regular, bold, _, _, err := fontcache.GetFonts(fontRegular, fontBold, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("load fonts: %w", err)
	}
	th := material.NewTheme()
	th.Bg = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	th.Fg = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	th.Shaper = text.NewShaper(text.WithCollection([]font.FontFace{
		{Font: font.Font{Typeface: "JetBrains Mono"}, Face: regular},
		{Font: font.Font{Typeface: "JetBrains Mono", Weight: font.Bold}, Face: bold},
	}))
	return th, nil
}

// paintToast draws the rounded black rect with title + body. Always opaque —
// the per-frame fade animation runs at the NSWindow alpha level, not in
// pixel colors.
func paintToast(gtx layout.Context, th *material.Theme, title, body string) {
	bg := color.NRGBA{R: 0, G: 0, B: 0, A: 255}
	bounds := image.Rectangle{Max: gtx.Constraints.Max}
	rrect := clip.RRect{Rect: bounds, NE: 16, NW: 16, SE: 16, SW: 16}
	paint.FillShape(gtx.Ops, bg, rrect.Op(gtx.Ops))

	titleColor := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	bodyColor := color.NRGBA{R: 230, G: 230, B: 230, A: 255}

	inset := layout.Inset{
		Top:    unit.Dp(20),
		Bottom: unit.Dp(20),
		Left:   unit.Dp(28),
		Right:  unit.Dp(28),
	}

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
				dims.Size.Y += gtx.Dp(unit.Dp(6))
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
