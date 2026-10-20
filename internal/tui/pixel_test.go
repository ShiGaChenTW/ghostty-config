package tui

import (
	"bytes"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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
