package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

// The systray library keeps its NSStatusItem private, so find our status bar button among the
// app's windows (each status item lives in its own window) and give it a bigger font.
// Setting the button's font sticks across later title changes.
static NSStatusBarButton* findButton(NSView* v) {
	if ([v isKindOfClass:[NSStatusBarButton class]]) return (NSStatusBarButton*)v;
	for (NSView* s in v.subviews) {
		NSStatusBarButton* b = findButton(s);
		if (b) return b;
	}
	return nil;
}

static void setMenuBarFontSize(double size) {
	dispatch_async(dispatch_get_main_queue(), ^{
		CGFloat pt = size > 0 ? size : [NSFont systemFontSize];
		// monospaced digits: the item doesn't jiggle in width as the price changes
		NSFont* f = [NSFont monospacedDigitSystemFontOfSize:pt weight:NSFontWeightRegular];
		for (NSWindow* w in [NSApp windows]) {
			NSStatusBarButton* b = findButton(w.contentView);
			if (b) b.font = f;
		}
	});
}
*/
import "C"

func setMenuBarFontSize(pt float64) { C.setMenuBarFontSize(C.double(pt)) }
