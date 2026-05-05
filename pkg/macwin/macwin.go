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
// positioning. Polls because Gio creates the NSWindow lazily after the first
// FrameEvent. Returns an error if no matching window appears within timeout.
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
