// macwin: AppKit shim that configures the Gio-managed NSWindow to behave as
// a non-interactive toast — borderless, transparent, click-through, never
// key, top-level, all-spaces — and positions it on the active screen.
//
// All AppKit calls dispatch onto the main thread because NSWindow methods
// are not thread-safe. Gio's app.Main() owns the main thread, so
// dispatch_sync from a Go goroutine is safe.

#import <Cocoa/Cocoa.h>

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

void macwin_setAccessoryPolicy(void) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
    });
}

static NSScreen* screenAtCursor(void) {
    NSPoint p = [NSEvent mouseLocation];
    for (NSScreen *s in [NSScreen screens]) {
        if (NSPointInRect(p, [s frame])) {
            return s;
        }
    }
    return [NSScreen mainScreen];
}

// macwin_configureToast applies toast NSWindow flags and positions the
// window at top-center of the active screen with vertical offset offsetY
// from the visible-frame top. width/height are in points.
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

        // Tell window-management tools (yabai, etc.) and the accessibility
        // tree to ignore this window — it's a toast, not a focusable window.
        [w setAccessibilityElement:NO];
        [w setExcludedFromWindowsMenu:YES];

        // Start the window fully transparent — animation will fade it in.
        [w setAlphaValue:0.0];

        NSScreen *s = screenAtCursor();
        NSRect vf = [s visibleFrame];
        NSRect target;
        target.size = NSMakeSize((CGFloat)width, (CGFloat)height);
        target.origin.x = vf.origin.x + (vf.size.width - (CGFloat)width) / 2.0;
        // NSWindow coords are bottom-left origin; offsetY is from the top.
        target.origin.y = vf.origin.y + vf.size.height - (CGFloat)offsetY - (CGFloat)height;
        [w setFrame:target display:YES];

        [w orderFrontRegardless];

        // Gio's window creation calls activateIgnoringOtherApps +
        // makeKeyAndOrderFront, which steals focus from whatever the user
        // was working on. Hand focus back: deactivate our app so the
        // previously-active app regains key.
        [NSApp deactivate];
    });
}

// macwin_setWindowAlpha animates the entire NSWindow's alpha. This is the
// cleanest way to fade a Gio window because it bypasses Gio's opaque
// framebuffer — the entire layer (no matter what Gio painted) fades in/out.
void macwin_setWindowAlpha(void *win, double alpha) {
    if (win == NULL) return;
    NSWindow *w = (__bridge NSWindow *)win;
    dispatch_sync(dispatch_get_main_queue(), ^{
        [w setAlphaValue:(CGFloat)alpha];
    });
}
