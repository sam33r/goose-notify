package ui

import "testing"

func TestBoxSizeShortMessageNoTitle(t *testing.T) {
	m := Metrics{
		BodyCharWidth:   8,
		BodyLineHeight:  18,
		TitleCharWidth:  9,
		TitleLineHeight: 20,
		PaddingX:        28,
		PaddingY:        20,
		TitleBodyGap:    6,
		MaxWidth:        600,
		MinWidth:        200,
	}
	w, h := BoxSize("hello", "", m)
	if w != 200 {
		t.Errorf("width = %d; want 200 (min width)", w)
	}
	if h != 58 {
		t.Errorf("height = %d; want 58", h)
	}
}

func TestBoxSizeLongMessageWraps(t *testing.T) {
	m := Metrics{
		BodyCharWidth:   8,
		BodyLineHeight:  18,
		TitleCharWidth:  9,
		TitleLineHeight: 20,
		PaddingX:        28,
		PaddingY:        20,
		TitleBodyGap:    6,
		MaxWidth:        200,
		MinWidth:        100,
	}
	w, h := BoxSize("abcdefghijklmnopqrst", "", m)
	if w != 200 {
		t.Errorf("width = %d; want 200 (clamped to max)", w)
	}
	if h != 76 {
		t.Errorf("height = %d; want 76", h)
	}
}

func TestBoxSizeWithTitle(t *testing.T) {
	m := Metrics{
		BodyCharWidth:   8,
		BodyLineHeight:  18,
		TitleCharWidth:  9,
		TitleLineHeight: 20,
		PaddingX:        28,
		PaddingY:        20,
		TitleBodyGap:    6,
		MaxWidth:        600,
		MinWidth:        200,
	}
	w, h := BoxSize("body line", "Title", m)
	if w != 200 {
		t.Errorf("width = %d; want 200", w)
	}
	if h != 84 {
		t.Errorf("height = %d; want 84", h)
	}
}

func TestBoxSizeMultilineBody(t *testing.T) {
	m := Metrics{
		BodyCharWidth:   8,
		BodyLineHeight:  18,
		TitleCharWidth:  9,
		TitleLineHeight: 20,
		PaddingX:        28,
		PaddingY:        20,
		TitleBodyGap:    6,
		MaxWidth:        600,
		MinWidth:        200,
	}
	_, h := BoxSize("line1\nline2\nline3", "", m)
	if h != 94 {
		t.Errorf("height = %d; want 94", h)
	}
}
