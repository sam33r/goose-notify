// macwin: AppKit shim that configures the Gio-managed NSWindow to behave as
// a non-interactive toast — borderless, transparent, click-through, never
// key, top-level, all-spaces — and positions it on the active screen.
//
// All AppKit calls dispatch onto the main thread because NSWindow methods
// are not thread-safe. Gio's app.Main() owns the main thread, so
// dispatch_sync from a Go goroutine is safe.

#import <Cocoa/Cocoa.h>
#import <objc/runtime.h>

void* macwin_findWindowByTitle(const char *titleC) {
    __block void *result = NULL;
    NSString *want = [NSString stringWithUTF8String:titleC];
    dispatch_sync(dispatch_get_main_queue(), ^{
        for (NSWindow *w in [NSApp windows]) {
            if (w == nil) continue;
            if ([[w title] isEqualToString:want]) {
                // Do every operation that affects yabai's view of the window
                // RIGHT HERE, before returning to Go. configureToast runs in
                // a separate dispatch tick and yabai can read the window's
                // initial AXStandardWindow subrole in that gap, firing
                // mouse_follows_focus.
                [w setAnimationBehavior:NSWindowAnimationBehaviorNone];
                [w setAlphaValue:0.0];
                object_setClass(w, [NSPanel class]);
                [w setAccessibilityElement:NO];
                [w setAccessibilitySubrole:NSAccessibilityFloatingWindowSubrole];
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

// Globally replace [NSWindow makeKeyAndOrderFront:] with orderFrontRegardless
// and no-op [NSApplication activateIgnoringOtherApps:]. Gio calls both during
// window creation, which makes the window the key window and activates our
// app — yabai's mouse_follows_focus then warps the cursor to the toast
// before we get a chance to morph the window into a panel and resign key.
//
// Our process only has the one toast window and never wants focus, so the
// blanket swizzle is safe. Must run before Gio creates anything.
static void installSwizzles(void) {
    static dispatch_once_t once;
    dispatch_once(&once, ^{
        Method m1 = class_getInstanceMethod([NSWindow class], @selector(makeKeyAndOrderFront:));
        method_setImplementation(m1, imp_implementationWithBlock(^(NSWindow *self, id sender) {
            [self orderFrontRegardless];
        }));

        Method m2 = class_getInstanceMethod([NSApplication class], @selector(activateIgnoringOtherApps:));
        method_setImplementation(m2, imp_implementationWithBlock(^(NSApplication *self, BOOL flag) {
            // intentional no-op
        }));

        // macOS 14+ [NSApp activate] — Gio likely uses this on newer SDKs
        // and the older activateIgnoringOtherApps: swizzle wouldn't catch it.
        SEL activateSel = @selector(activate);
        Method m2b = class_getInstanceMethod([NSApplication class], activateSel);
        if (m2b) {
            method_setImplementation(m2b, imp_implementationWithBlock(^(NSApplication *self) {
                // intentional no-op
            }));
        }

        // Also no-op makeKeyWindow / becomeKeyWindow defensively. Gio's
        // window creation may call these directly bypassing our
        // makeKeyAndOrderFront swizzle.
        Method m4 = class_getInstanceMethod([NSWindow class], @selector(makeKeyWindow));
        method_setImplementation(m4, imp_implementationWithBlock(^(NSWindow *self) {
            // intentional no-op
        }));

        // Hard refusal to ever become key/main. Our window-level fixes
        // (NSPanel + becomesKeyOnlyIfNeeded) only soften the request;
        // overriding canBecomeKeyWindow / canBecomeMainWindow to return
        // NO at the AppKit level means the OS itself never marks our
        // window focused, so yabai's has-focus stays false and
        // mouse_follows_focus never fires.
        Method m5 = class_getInstanceMethod([NSWindow class], @selector(canBecomeKeyWindow));
        method_setImplementation(m5, imp_implementationWithBlock(^BOOL(NSWindow *self) {
            return NO;
        }));
        Method m6 = class_getInstanceMethod([NSWindow class], @selector(canBecomeMainWindow));
        method_setImplementation(m6, imp_implementationWithBlock(^BOOL(NSWindow *self) {
            return NO;
        }));

        // (Swizzling becomeKeyWindow / becomeMainWindow to no-op was tried
        // and made things strictly worse — AppKit relies on those for
        // bookkeeping and breaking them generates extra focus traffic.)

        // Force every NSWindow's AX subrole to AXFloatingWindow. yabai's
        // window_is_standard() checks this; reporting Floating from the
        // first AX query (before configureToast runs) prevents the race
        // where yabai reads AXStandardWindow on creation, fires
        // mouse_follows_focus, and only then sees our override.
        Method m3 = class_getInstanceMethod([NSWindow class], @selector(accessibilitySubrole));
        method_setImplementation(m3, imp_implementationWithBlock(^NSString *(NSWindow *self) {
            return NSAccessibilityFloatingWindowSubrole;
        }));
    });
}

void macwin_setAccessoryPolicy(void) {
    dispatch_sync(dispatch_get_main_queue(), ^{
        installSwizzles();
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
        // Morph the Gio-created NSWindow into an NSPanel before any other
        // setup. This changes the AX subrole reported to the accessibility
        // tree from AXStandardWindow → AXSystemDialog/AXFloatingWindow, so
        // tiling window managers (yabai, Amethyst) skip it instead of
        // tiling/reflowing around it. Safe because NSPanel inherits from
        // NSWindow — all selectors we send below remain valid.
        object_setClass(w, [NSPanel class]);

        // (We don't touch styleMask — calling setStyleMask: after the window
        // has been shown crashes inside AppKit's titlebar-view KVO teardown.
        // NSPanel's default becomesKeyOnlyIfNeeded:YES + the resignKeyWindow
        // below are enough to drop key status.)

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
            NSWindowCollectionBehaviorIgnoresCycle |
            NSWindowCollectionBehaviorTransient];

        // Belt-and-braces: also tell the AX tree this window isn't a
        // focusable element and exclude it from the Window menu.
        [w setAccessibilityElement:NO];
        [w setExcludedFromWindowsMenu:YES];

        // Force the AX subrole to AXFloatingWindow. The subrole is normally
        // derived from styleMask, and changing the mask post-show crashes
        // AppKit; setAccessibilitySubrole: lets us override the reported
        // value without touching the mask. yabai's window_is_standard()
        // checks this and treats non-Standard subroles as floating, which
        // means mouse_follows_focus skips the toast.
        [w setAccessibilitySubrole:NSAccessibilityFloatingWindowSubrole];

        // Disable AppKit's built-in show/hide alpha animation. Otherwise our
        // setAlphaValue:0 here gets overridden by AppKit's default fade-in
        // when orderFrontRegardless runs, and our ticker-driven fade never
        // visibly takes effect.
        [w setAnimationBehavior:NSWindowAnimationBehaviorNone];

        // Start the window fully transparent — our ticker will fade it in.
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

        // No resignKeyWindow / deactivate here: the swizzles in
        // installSwizzles() prevent our app from activating and our window
        // from becoming key, so explicitly resigning/deactivating just
        // generates extra focus events that yabai reacts to (e.g. by
        // pulling keyboard focus away from the user's foreground window).
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
