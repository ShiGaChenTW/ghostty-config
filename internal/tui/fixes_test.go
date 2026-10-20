package tui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vitruves/ghostty-config/internal/extensions"
	"github.com/vitruves/ghostty-config/internal/ghostty"
)

// activeLines lists the values of a key's lines that are not commented out.
func activeLines(config, key string) []string {
	var out []string
	for _, l := range strings.Split(config, "\n") {
		if v, ok := strings.CutPrefix(strings.TrimSpace(l), key+" = "); ok {
			out = append(out, v)
		}
	}
	return out
}

// stubShaders makes every shader download succeed without the network.
func stubShaders(t *testing.T) {
	t.Helper()
	real := extensions.Get
	extensions.Get = func(string) ([]byte, error) { return []byte("void mainImage(out vec4 colour, in vec2 at) {}\n"), nil }
	t.Cleanup(func() { extensions.Get = real })
}

// stubReload makes Ghostty reload by itself, counting the keystrokes.
func stubReload(t *testing.T, h *harness) *int {
	t.Helper()
	real := reloadGhostty
	reloads := 0
	reloadGhostty = func(string) error { reloads++; return nil }
	t.Cleanup(func() { reloadGhostty = real })
	h.m.opts.NoReload = false
	h.m.caps.inGhostty = true
	return &reloads
}

// While the editor steps aside no dialog is drawn: an update offered then
// waits, and the key that brings the editor back answers nothing.
func TestAnUpdateNeverInstallsWhileTheEditorStepsAside(t *testing.T) {
	h := newHarness(t, "theme = Dracula\n")
	h.m.opts.Version = "1.0.0"
	asked := ""
	old := installRelease
	installRelease = func(_ context.Context, v string) (string, error) { asked = v; return "/tmp/x", nil }
	defer func() { installRelease = old }()

	h.key("f2")
	h.send(updateMsg{latest: "1.2.0"})
	if h.m.overlay != overlayNone || !h.m.peek {
		t.Fatalf("the offer should wait while peeking, overlay %v", h.m.overlay)
	}
	h.key("enter")
	if asked != "" || h.m.peek {
		t.Fatalf("the key that ends the peek installed %q", asked)
	}
	if h.m.overlay != overlayUpdate {
		t.Fatalf("the offer should show once the editor is back, overlay %v", h.m.overlay)
	}
	h.key("esc")

	// A dialog that opened some other way while peeking is not answered by
	// the key that ends the peek either.
	h.key("f2")
	h.m.overlay = overlayUpdate
	h.key("y")
	if asked != "" || h.m.overlay != overlayUpdate {
		t.Fatalf("a dialog nobody saw was answered, installed %q", asked)
	}
}

// An update offered behind another dialog is shown when that one closes,
// whichever dialog it was.
func TestAnUpdateHeldBehindADialogIsShownWhenItCloses(t *testing.T) {
	h := newHarness(t, "theme = Dracula\n")
	h.m.opts.Version = "1.0.0"
	h.key("f1")
	h.send(updateMsg{latest: "1.2.0"})
	if h.m.overlay != overlayHelp {
		t.Fatalf("the help should stay up, overlay %v", h.m.overlay)
	}
	h.key("x")
	if h.m.overlay != overlayUpdate {
		t.Fatalf("closing the help should show the update, overlay %v", h.m.overlay)
	}
	h.key("esc")

	h.m.openConfirm("Really?", nil)
	h.send(updateMsg{latest: "1.3.0"})
	h.key("n")
	if h.m.overlay != overlayUpdate || h.m.updateLatest != "1.3.0" {
		t.Fatalf("answering a question should show the update, overlay %v", h.m.overlay)
	}
}

// The timer of "keep this shader?" answers that question and no other, and
// leaving while it is open puts the shader back first.
func TestTheShaderTimerAnswersOnlyItsOwnQuestion(t *testing.T) {
	stubShaders(t)
	h := newHarness(t, "theme = Dracula\n")
	stubReload(t, h)
	h.typeText("edit backg")
	h.key("enter", "right", "esc", "esc")
	if !h.m.dirty {
		t.Fatal("the theme should have unsaved edits")
	}
	h.typeText("shader bloom")
	h.m.Update(keyPress("enter"))
	h.m.Update(shaderDoneMsg{name: "bloom", path: filepath.Join(h.paths.ConfigDir, "shaders", "bloom.glsl")})
	if h.m.overlay != overlayConfirm || !strings.Contains(h.m.confirmText, "Keep it?") {
		t.Fatalf("the shader should be asked about, overlay %v", h.m.overlay)
	}
	seq := h.m.confirmSeq
	h.m.Update(keyPress("ctrl+d"))
	if len(activeLines(h.config(), "custom-shader")) != 0 {
		t.Fatalf("leaving should put the unkept shader back first:\n%s", h.config())
	}
	if h.m.overlay != overlayConfirm || !strings.Contains(h.m.confirmText, "unsaved edits") || h.m.quitting {
		t.Fatalf("the edits should still be asked about, overlay %v %q", h.m.overlay, h.m.confirmText)
	}
	h.m.Update(shaderExpireMsg{seq})
	if h.m.quitting || h.m.overlay != overlayConfirm {
		t.Fatal("the shader's timer answered the question about the edits, and left")
	}

	// Ctrl+C with the question open puts the shader back and says so.
	h = newHarness(t, "theme = Dracula\n")
	stubReload(t, h)
	h.typeText("shader bloom")
	h.m.Update(keyPress("enter"))
	h.m.Update(shaderDoneMsg{name: "bloom", path: filepath.Join(h.paths.ConfigDir, "shaders", "bloom.glsl")})
	if len(activeLines(h.config(), "custom-shader")) != 1 {
		t.Fatalf("the shader should be written:\n%s", h.config())
	}
	h.m.Update(keyPress("ctrl+c"))
	if !h.m.quitting || len(activeLines(h.config(), "custom-shader")) != 0 {
		t.Fatalf("Ctrl+C should put the shader back:\n%s", h.config())
	}
	if note := h.m.ExitNote(); !strings.Contains(note, "put back") {
		t.Fatalf("the note should say the shader was put back, says %q", note)
	}
}

// custom-shader stacks: choosing one switches every other line off, and an
// unkept choice puts back exactly the lines that were there.
func TestChoosingAShaderReplacesEveryShaderLine(t *testing.T) {
	stubShaders(t)
	config := "theme = Dracula\ncustom-shader = /a.glsl\ncustom-shader = /b.glsl\n"
	h := newHarness(t, config)
	h.typeText("shader bloom")
	h.key("enter")
	bloom := filepath.Join(h.paths.ConfigDir, "shaders", "bloom.glsl")
	if got := activeLines(h.config(), "custom-shader"); len(got) != 1 || got[0] != bloom {
		t.Fatalf("only the chosen shader should be on, are %v:\n%s", got, h.config())
	}

	h = newHarness(t, config)
	stubReload(t, h)
	h.typeText("shader bloom")
	h.m.Update(keyPress("enter"))
	h.m.Update(shaderDoneMsg{name: "bloom", path: bloom})
	if got := activeLines(h.config(), "custom-shader"); len(got) != 1 {
		t.Fatalf("only the chosen shader should be on while asking, are %v", got)
	}
	h.key("n")
	if h.config() != "theme = Dracula\ncustom-shader = /a.glsl\ncustom-shader = /b.glsl\n" {
		t.Fatalf("no should put back both lines as they were:\n%s", h.config())
	}
}

// selectSetting puts the highlight on a setting of the menu.
func (h *harness) selectSetting(group, name string) {
	h.t.Helper()
	h.key("esc")
	h.m.jumpToGroup(group)
	for i := 0; i < 40 && (h.m.selected() == nil || h.m.selected().value != name); i++ {
		h.key("down")
	}
	if o := h.m.selected(); o == nil || o.value != name {
		h.t.Fatalf("no %s in %s", name, group)
	}
}

// A value written between two steps, or the same step written another way,
// is stepped from where it is, never from the bottom of the scale.
func TestArrowsStepAnOffStepValueFromWhereItIs(t *testing.T) {
	h := newHarness(t, "theme = Dracula\nbackground-opacity = 0.9\nwindow-padding-x = 5\n")
	h.selectSetting("Window", "opacity")
	h.key("left")
	if got := activeLines(h.config(), "background-opacity"); len(got) != 1 || got[0] != "0.85" {
		t.Fatalf("0.9 is 0.90: left should step to 0.85, wrote %v", got)
	}
	h.selectSetting("Window", "paddingx")
	h.key("left")
	if got := activeLines(h.config(), "window-padding-x"); len(got) != 1 || got[0] != "4" {
		t.Fatalf("5 is between 4 and 6: left should go to 4, wrote %v", got)
	}

	h = newHarness(t, "theme = Dracula\nwindow-padding-x = 5\n")
	h.selectSetting("Window", "paddingx")
	h.key("right")
	if got := activeLines(h.config(), "window-padding-x"); len(got) != 1 || got[0] != "6" {
		t.Fatalf("right from 5 should go to 6, wrote %v", got)
	}
	h.typeText("paddingx ")
	if o := h.m.selected(); o == nil || o.value != "6" || !o.current {
		t.Fatalf("the list should open on the value in effect, on %+v", o)
	}
}

// Typing filters a list; only moving the highlight by hand writes what it
// rests on.
func TestTypingASearchWritesNothing(t *testing.T) {
	config := "theme = Dracula\nfont-family = Menlo\nfont-size = 14\n"
	h := newHarness(t, config)
	h.typeText("size 2")
	time.Sleep(5 * time.Millisecond)
	h.key("esc")
	h.typeText("style b")
	h.key("esc")
	h.typeText("font h")
	h.key("esc")
	if h.config() != config {
		t.Fatalf("typing a search wrote the config:\n%s", h.config())
	}
	h.typeText("size ")
	h.key("down")
	if !strings.Contains(h.config(), "font-size = 14.5\n") {
		t.Fatalf("moving the highlight should still write the size:\n%s", h.config())
	}
}

// Walking the interfaces shows each one; leaving the list without Enter puts
// back the one in effect, so a look is never saved.
func TestAnInterfaceLookedAtIsNotSaved(t *testing.T) {
	h := newHarness(t, "theme = Dracula\n")
	h.typeText("interface ")
	h.key("up")
	if h.m.state.Interface == "clear" {
		t.Fatal("moving the highlight should show another interface")
	}
	h.key("esc")
	if h.m.state.Interface != "clear" {
		t.Fatalf("Esc should put the interface back, is %q", h.m.state.Interface)
	}
	h.typeText("interface ")
	h.key("up", "esc", "esc")
	if !h.m.quitting {
		t.Fatal("the editor should have been left")
	}
	if got := ghostty.LoadState(h.paths).Interface; got != "clear" {
		t.Fatalf("a look was saved as the interface: %q", got)
	}

	h = newHarness(t, "theme = Dracula\n")
	h.typeText("interface ")
	h.key("up")
	chosen := h.m.state.Interface
	h.key("enter", "esc", "esc")
	if got := ghostty.LoadState(h.paths).Interface; got != chosen {
		t.Fatalf("Enter should keep the interface %q, saved %q", chosen, got)
	}
}

// A theme file already in the themes directory is the user's: saving over it
// asks first, and a proposed name never lands on one.
func TestSavingOverAThemeFileAsks(t *testing.T) {
	h := newHarness(t, "theme = Dracula\n")
	mine := filepath.Join(h.paths.ThemesDir, "mine")
	original := "background = black\nforeground = white\n"
	os.WriteFile(mine, []byte(original), 0o644)
	h.typeText("random")
	h.key("enter")
	h.typeText("save mine")
	h.key("enter")
	if h.m.overlay != overlayConfirm || !strings.Contains(h.m.confirmText, "mine") {
		t.Fatalf("saving over a file should ask, overlay %v", h.m.overlay)
	}
	h.key("n")
	if data, _ := os.ReadFile(mine); string(data) != original {
		t.Fatalf("no should leave the file alone:\n%s", data)
	}
	h.key("enter", "y")
	if data, _ := os.ReadFile(mine); string(data) == original || !strings.Contains(h.config(), "theme = mine\n") {
		t.Fatalf("yes should save over it and apply it:\n%s\n%s", data, h.config())
	}

	h = newHarness(t, "theme = Dracula\n")
	for _, name := range []string{"Dracula-custom", "Dracula-custom-2"} {
		os.WriteFile(filepath.Join(h.paths.ThemesDir, name), []byte(original), 0o644)
	}
	if h.m.cur.Name != "Dracula" {
		t.Fatalf("on %s, want Dracula", h.m.cur.Name)
	}
	if got := h.m.suggestName(); got != "Dracula-custom-3" {
		t.Fatalf("the proposed name should be free on disk, is %s", got)
	}
}

// Leaving with nothing written sends no keystroke; a keystroke that found
// another app in front says so without giving up on reloading.
func TestReloadOnlyWhenWrittenAndOnlyIntoGhostty(t *testing.T) {
	h := newHarness(t, "theme = Dracula\n")
	reloads := stubReload(t, h)
	before := h.m.reloadSeq
	h.key("esc")
	if !h.m.quitting || h.m.reloadSeq != before || *reloads != 0 {
		t.Fatalf("leaving with nothing changed reloaded (%d)", *reloads)
	}

	h = newHarness(t, "theme = Dracula\n")
	stubReload(t, h)
	h.m.reloadSeq = 3
	h.send(reloadDoneMsg{seq: 3, err: ghostty.ErrNotFrontmost})
	if h.m.reloadFailed || !h.m.autoReloadWorks() {
		t.Fatal("Ghostty not being in front should not stop the next reload")
	}
	if !strings.Contains(h.m.status, "not in front") {
		t.Fatalf("the status should say why, says %q", h.m.status)
	}
	h.send(reloadDoneMsg{seq: 3, err: errors.New("blocked")})
	if !h.m.reloadFailed {
		t.Fatal("any other failure still stops the reloads")
	}
}

// paintLog stands in for the terminal: what the editor paints lands in a file.
func paintLog(t *testing.T) func() string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tty")
	real := openLive
	openLive = func() *ghostty.Live {
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		return ghostty.LiveOn(f)
	}
	t.Cleanup(func() { openLive = real })
	return func() string {
		data, _ := os.ReadFile(path)
		return string(data)
	}
}

const resetColours = "\x1b]104\x1b\\\x1b]110\x1b\\\x1b]111\x1b\\\x1b]112\x1b\\\x1b]117\x1b\\\x1b]119\x1b\\"

// Colours painted into the terminal never outlive the editor: not after a
// try followed by Enter on another theme, not when the program is ended from
// outside, and where Ghostty does not reload the applied theme is shown.
func TestPaintedColoursAreAlwaysPutBack(t *testing.T) {
	painted := paintLog(t)
	h := newHarness(t, "theme = Dracula\n")
	h.m.caps.inGhostty = false
	h.typeText("theme ")
	h.key("ctrl+t")
	if h.m.tried == "" {
		t.Fatal("Ctrl+T should try the theme")
	}
	h.key("right", "enter", "esc", "esc")
	if !h.m.quitting || !strings.HasSuffix(painted(), resetColours) {
		t.Fatalf("leaving should put the terminal's colours back: %q", painted())
	}

	// Interrupted: main asks the model, once is enough.
	painted = paintLog(t)
	h = newHarness(t, "theme = Dracula\n")
	h.m.caps.inGhostty = false
	h.typeText("theme ")
	h.key("ctrl+t")
	h.m.RestoreColours()
	after := painted()
	h.m.RestoreColours()
	if !strings.HasSuffix(after, resetColours) || painted() != after {
		t.Fatalf("RestoreColours should reset once: %q", painted())
	}

	// In Ghostty without a reload, the theme Enter applied is shown again
	// after the reset, since the window still holds its old configuration.
	painted = paintLog(t)
	h = newHarness(t, "theme = Dracula\n")
	h.m.caps.inGhostty = true
	h.typeText("theme Catppuccin Mocha")
	h.key("enter")
	applied := h.m.applied
	h.key("esc")
	h.typeText("theme ")
	h.key("ctrl+t", "right", "ctrl+t", "left", "esc", "esc")
	t2, _ := h.m.lib.Get(applied)
	log := painted()
	tail := log[strings.LastIndex(log, resetColours):]
	if applied == "Dracula" || !strings.Contains(tail, "\x1b]11;"+t2.Colors["background"]) {
		t.Fatalf("after the reset %s should be shown again: %q", applied, tail)
	}
}
