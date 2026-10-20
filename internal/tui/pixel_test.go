package tui

import (
	"bytes"
	"context"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/vitruves/ghostty-config/internal/pix"
)

const pixConfig = "theme = sumi-ink\nbackground-opacity = 0.85\nbackground-blur = macos-glass-regular\nmacos-titlebar-style = tabs\nwindow-padding-x = 14\nwindow-padding-color = extend\nwindow-save-state = always\n"

func newPixHarness(t *testing.T, iface string) *harness {
	t.Helper()
	h := newHarnessWith(t, pixConfig, Options{NoReload: true, Images: true, ImgOut: io.Discard, Pixel: true, PixCell: [2]int{17, 38}})
	h.m.state.Interface = iface
	h.send(tea.WindowSizeMsg{Width: 120, Height: 34})
	return h
}

// shot draws the screen as the terminal would show it and, when
// GHOSTTY_CONFIG_PXSHOTS names a directory, writes it there as a PNG.
func (h *harness) shot(name string) {
	h.t.Helper()
	m := h.m
	if !m.pixelOn() {
		h.t.Fatal("the picture interface is off")
	}
	// The real path, rows of placeholders included.
	if rows := strings.Split(m.render(), "\n"); len(rows) != m.height {
		h.t.Fatalf("%s: %d rows for a %d-row terminal", name, len(rows), m.height)
	}
	dir := os.Getenv("GHOSTTY_CONFIG_PXSHOTS")
	if dir == "" {
		return
	}
	m.regions = m.regions[:0]
	c := pix.New(m.width*m.px.cw, m.height*m.px.ch, float32(m.px.ch)/pixUnit)
	m.drawScreen(c)
	var b bytes.Buffer
	_ = png.Encode(&b, c.Img)
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, name+".png"), b.Bytes(), 0o644)
}

func (h *harness) highlightTheme(name string) {
	for i, o := range h.m.results {
		if o.theme != nil && o.theme.Name == name {
			h.m.sel.cursor = i
			h.m.galAlign = true
			h.m.afterMove()
			return
		}
	}
	h.t.Fatalf("no theme %q on the wall", name)
}

func (h *harness) highlightCommand(name string) {
	for i, o := range h.m.results {
		if o.cmd != nil && o.cmd.name == name {
			h.m.sel.cursor = i
			return
		}
	}
	h.t.Fatalf("no command %q in the menu", name)
}

func TestPixelScreens(t *testing.T) {
	h := newPixHarness(t, "midnight")

	h.m.setPromptText("Theme ")
	h.highlightTheme("matcha-whisk")
	h.shot("01-themes")

	h.m.jumpToGroup("Window")
	h.highlightCommand("paddingcolor")
	h.shot("02-window")

	h.key("right")
	if !strings.Contains(h.config(), "window-padding-color = extend-always") {
		t.Fatalf("→ on the padding colour did not step it:\n%s", h.config())
	}
	h.key("delete")
	if strings.Contains(h.config(), "\nwindow-padding-color") {
		t.Fatalf("Delete did not put the padding colour back to its default:\n%s", h.config())
	}

	h.m.jumpToGroup("Fonts")
	h.shot("03-fonts")
	h.m.jumpToGroup("Input")
	h.shot("04-input")
	h.m.jumpToGroup("Session")
	h.shot("05-session")
	h.m.jumpToGroup("Extensions")
	h.shot("06-extensions")
	h.m.jumpToGroup("Tool")
	h.shot("07-tool")

	h.m.setPromptText("Font ")
	h.shot("08-font-list")
	h.m.setPromptText("Opacity ")
	h.shot("09-opacity-values")

	h.m.setPromptText("")
	h.m.jumpToGroup("Window")
	h.m.overlay = overlayHelp
	h.shot("10-help")
	h.m.overlay = overlayNone
	h.m.openConfirm("Delete the theme file matcha-whisk from your themes directory? A backup is not kept.", nil)
	h.shot("11-confirm")
	h.m.overlay = overlayWelcome
	h.shot("12-welcome")
	h.m.overlay = overlayNone

	h.m.setPromptText("Edit ")
	h.key("enter")
	h.shot("13-editor")

	light := newPixHarness(t, "paper")
	light.m.setPromptText("Theme ")
	light.highlightTheme("sakura-fade")
	light.shot("14-themes-light")
	light.m.jumpToGroup("Window")
	light.highlightCommand("opacity")
	light.shot("15-window-light")
}

// TestReadmeShots draws the pictures of the README when
// GHOSTTY_CONFIG_README names a directory: `make readme-shots`.
func TestReadmeShots(t *testing.T) {
	dir := os.Getenv("GHOSTTY_CONFIG_README")
	if dir == "" {
		t.Skip("GHOSTTY_CONFIG_README is not set")
	}
	t.Setenv("GHOSTTY_CONFIG_PXSHOTS", dir)
	harnessXDG = filepath.Join(".config", "ghostty")
	defer func() { harnessXDG = "xdg" }()
	shoot := func(iface string, name string, set func(h *harness)) {
		h := newPixHarness(t, iface)
		t.Setenv("HOME", h.dir)
		set(h)
		h.shot(name)
	}
	shoot("midnight", "themes", func(h *harness) {
		h.m.setPromptText("Theme ")
		h.highlightTheme("matcha-whisk")
	})
	shoot("paper", "window", func(h *harness) {
		h.m.setPromptText("")
		h.m.jumpToGroup("Window")
		h.highlightCommand("opacity")
	})
	shoot("midnight", "input", func(h *harness) {
		h.m.setPromptText("")
		h.m.jumpToGroup("Input")
		h.highlightCommand("cursor")
	})
	shoot("paper", "themes-light", func(h *harness) {
		h.m.setPromptText("Theme ")
		h.highlightTheme("sakura-fade")
	})
}

// The renderer clears the whole screen after a resize, and the terminal
// drops every picture with it: after a resize, every row must be sent again,
// even one whose pixels did not change.
func TestPicturesAreSentAgainAfterAResize(t *testing.T) {
	var out bytes.Buffer
	h := newHarnessWith(t, pixConfig, Options{NoReload: true, Images: true, ImgOut: &out, Pixel: true, PixCell: [2]int{17, 38}})
	h.m.View()
	first := strings.Count(out.String(), "a=T")
	if first == 0 {
		t.Fatal("the first frame sent no picture")
	}
	out.Reset()
	h.m.View()
	if strings.Contains(out.String(), "a=T") {
		t.Fatal("an unchanged frame sent its pictures again")
	}
	h.send(tea.WindowSizeMsg{Width: 130, Height: 38})
	h.m.View()
	if n := strings.Count(out.String(), "a=T"); n != first {
		t.Fatalf("after a resize %d pictures were sent, want all %d of the frame", n, first)
	}
}

// The controls of a setting answer to the mouse: a pill picks its value, a
// switch flips, a point along a slider sets the value under it.
func TestSettingControlsAreClickable(t *testing.T) {
	h := newPixHarness(t, "midnight")
	h.m.jumpToGroup("Window")

	pill, ok := h.find(hitValue, -1, "dark")
	if !ok {
		t.Fatal("no pill for a dark window theme")
	}
	h.click(pill.x+pill.w/2, pill.y)
	if !strings.Contains(h.config(), "window-theme = dark") {
		t.Fatalf("clicking the pill did not set it:\n%s", h.config())
	}

	toggle, ok := h.find(hitValue, -1, "on")
	if !ok {
		t.Fatal("no switch to turn on")
	}
	h.click(toggle.x+toggle.w/2, toggle.y)
	if o := h.m.results[toggle.index]; o.cmd.set.get(h.m) != "on" {
		t.Fatalf("clicking the switch of %s left it %s", o.cmd.set.key, o.cmd.set.get(h.m))
	}

	for _, r := range h.m.regions {
		if r.kind == hitValue && r.name == "slider" && h.m.results[r.index].cmd.name == "paddingx" {
			h.click(r.x, r.y)
			if !strings.Contains(h.config(), "window-padding-x = 0") {
				t.Fatalf("clicking the start of the slider did not set 0:\n%s", h.config())
			}
			h.click(r.x+r.w-1, r.y)
			if !strings.Contains(h.config(), "window-padding-x = 60") {
				t.Fatalf("clicking the end of the slider did not set 60:\n%s", h.config())
			}
			return
		}
	}
	t.Fatal("no slider for the left and right padding")
}

// A newer release opens a dialog: Next time closes it, Install replaces the
// binary through the installer and says so.
func TestUpdateDialog(t *testing.T) {
	h := newPixHarness(t, "midnight")
	h.m.opts.Version = "1.0.0"
	var asked string
	old := installRelease
	installRelease = func(_ context.Context, v string) (string, error) {
		asked = v
		return "/usr/local/bin/ghostty-config", nil
	}
	defer func() { installRelease = old }()

	h.send(updateMsg{latest: "1.0.0"})
	if h.m.overlay == overlayUpdate {
		t.Fatal("the same version offered itself as an update")
	}
	h.send(updateMsg{latest: "1.2.0"})
	if h.m.overlay != overlayUpdate {
		t.Fatal("a newer release opened no dialog")
	}
	h.shot("16-update")
	if _, ok := h.find(hitButton, -1, "update-install"); !ok {
		t.Fatal("the dialog has no Install button")
	}
	h.key("esc")
	if h.m.overlay != overlayNone || asked != "" {
		t.Fatalf("Next time left overlay %d, installed %q", h.m.overlay, asked)
	}

	h.send(updateMsg{latest: "1.2.0"})
	h.key("enter")
	if asked != "1.2.0" {
		t.Fatalf("Install asked for %q, want 1.2.0", asked)
	}
	if h.m.overlay != overlayMessage || !strings.Contains(h.m.messageText, "Updated to 1.2.0") {
		t.Fatalf("after Install: overlay %d, %q", h.m.overlay, h.m.messageText)
	}
	if !strings.Contains(h.m.ExitNote(), "1.2.0") {
		t.Fatalf("the exit note does not say the new version: %q", h.m.ExitNote())
	}

	// Held back while the welcome is open, shown when it closes.
	w := newPixHarness(t, "midnight")
	w.m.opts.Version = "1.0.0"
	w.m.overlay = overlayWelcome
	w.send(updateMsg{latest: "1.2.0"})
	if w.m.overlay != overlayWelcome {
		t.Fatal("the update dialog covered the welcome")
	}
	w.key("enter")
	if w.m.overlay != overlayUpdate {
		t.Fatal("closing the welcome did not show the update")
	}
}

// The cell size comes from the terminal, never from the window's pixel size:
// a resize asks for it, and only a new answer changes it.
func TestCellSizeComesFromTheTerminal(t *testing.T) {
	var out bytes.Buffer
	h := newHarnessWith(t, pixConfig, Options{NoReload: true, Images: true, ImgOut: &out, Pixel: true, PixCell: [2]int{17, 38}})
	_, cmd := h.m.Update(tea.WindowSizeMsg{Width: 118, Height: 30})
	if cmd == nil {
		t.Fatal("a resize asked nothing of the terminal")
	}
	if h.m.px.cw != 17 || h.m.px.ch != 38 {
		t.Fatalf("a resize changed the cell to %dx%d", h.m.px.cw, h.m.px.ch)
	}
	h.send(uv.CellSizeEvent{Width: 19, Height: 42})
	if h.m.px.cw != 19 || h.m.px.ch != 42 {
		t.Fatalf("the reported cell was not taken: %dx%d", h.m.px.cw, h.m.px.ch)
	}
	out.Reset()
	h.m.View()
	if !strings.Contains(out.String(), "s="+itoa(118*19)+",v=42") {
		t.Fatal("the rows were not drawn again at the new cell size")
	}
}
