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
void  macwin_setWindowAlpha(void *win, double alpha);
*/
import "C"

import (
	"errors"
	"time"
	"unsafe"
)

// Handle is a retained reference to the toast NSWindow. Release with Free.
type Handle struct {
	ptr unsafe.Pointer
}

// SetAccessoryPolicy switches the process to
// NSApplicationActivationPolicyAccessory: no Dock icon, no menu bar.
// Overrides Gio's hardcoded Regular policy. Call once at startup.
func SetAccessoryPolicy() {
	C.macwin_setAccessoryPolicy()
}

// ConfigureToast finds the Gio NSWindow by title, applies toast flags +
// positioning, and returns a Handle so the caller can drive the fade
// animation via SetAlpha. The window is shown with alpha=0; the caller
// must animate it up.
//
// Polls because Gio creates the NSWindow lazily after the first FrameEvent.
// Returns an error if no matching window appears within timeout.
func ConfigureToast(title string, width, height, offsetY int, timeout time.Duration) (*Handle, error) {
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
			return nil, errors.New("macwin: no window with matching title within timeout")
		}
		time.Sleep(time.Millisecond)
	}

	C.macwin_configureToast(ptr, C.int(width), C.int(height), C.int(offsetY))
	return &Handle{ptr: ptr}, nil
}

// SetAlpha sets the window's opacity (0.0..1.0). Safe to call from any
// goroutine; the AppKit call is dispatched to the main thread.
func (h *Handle) SetAlpha(alpha float64) {
	if h == nil || h.ptr == nil {
		return
	}
	C.macwin_setWindowAlpha(h.ptr, C.double(alpha))
}

// Free releases the retained NSWindow reference. Idempotent.
func (h *Handle) Free() {
	if h == nil || h.ptr == nil {
		return
	}
	C.macwin_releaseWindow(h.ptr)
	h.ptr = nil
}
