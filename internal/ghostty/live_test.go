package ghostty

import (
	"strings"
	"testing"
)

// System Events types into whichever app is in front, whatever process the
// script names: the keystroke must only go out while that app is Ghostty.
func TestReloadKeystrokeOnlyGoesToGhosttyInFront(t *testing.T) {
	script := reloadScript(",", "command down, shift down")
	guard := strings.Index(script, "if frontmost of ghostty is false then return \""+notFrontmost+"\"")
	send := strings.Index(script, "tell ghostty to keystroke \",\" using {command down, shift down}")
	if guard < 0 || send < 0 || guard > send {
		t.Fatalf("the script should check that Ghostty is in front before typing:\n%s", script)
	}
}
