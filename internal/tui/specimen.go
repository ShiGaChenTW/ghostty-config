package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/vitruves/ghostty-config/internal/ghostty"
	"github.com/vitruves/ghostty-config/internal/pix"
)

// The picture interface sets a family's specimen in the family itself, read
// from its files, so looking through the fonts never touches Ghostty: a
// reload with each family changes the size of every cell, and with it the
// whole screen. Enter is what writes the family, as it is for a theme.

// specimenCap is how many font files stay open; a few of the system's
// families are collections of tens of megabytes.
const specimenCap = 24

// specimenFont is one file of the cache: nil with no error while it loads.
type specimenFont struct {
	font *pix.Font
	err  error
}

type specimenMsg struct{ fonts map[string]specimenFont }

func fileKey(f ghostty.FontFile) string {
	return f.Path + "\x00" + f.PostScript + "\x00" + strconv.Itoa(f.Index)
}

// browsesFonts is whether the open command lists families or their styles,
// where moving the highlight only looks in the picture interface.
func (m *Model) browsesFonts() bool {
	return m.pixelOn() && m.cmd != nil && (m.cmd.name == "font" || m.cmd.name == "style")
}

// specimenOf is the family and style the font panel shows: the highlighted
// family, or the configured one with the highlighted style.
func (m *Model) specimenOf(opt *option) (ghostty.Family, string, bool) {
	if m.cmd != nil && m.cmd.name == "font" {
		if opt == nil || opt.family == nil {
			return ghostty.Family{}, "", false
		}
		return *opt.family, m.currentStyle(), true
	}
	f, ok := m.findFamily(m.fontFamily)
	if !ok {
		return ghostty.Family{}, "", false
	}
	style := m.currentStyle()
	if m.cmd != nil && m.cmd.name == "style" && opt != nil && opt.value != "" {
		style = opt.value
	}
	return f, style, true
}

// specimenFaces are the files of the three lines that differ in face: the
// style shown, bold, italic.
func specimenFaces(f ghostty.Family, style string) [3]ghostty.FontFile {
	var out [3]ghostty.FontFile
	out[0], _ = f.File(style)
	out[1], _ = f.File("Bold")
	out[2] = out[0]
	for _, s := range []string{"Italic", "Oblique"} {
		if file, ok := f.Files[s]; ok {
			out[2] = file
			break
		}
	}
	return out
}

// specimenFont returns a loaded file, nil while it loads or when it cannot
// be read.
func (m *Model) specimenFont(f ghostty.FontFile) *pix.Font {
	if f.Path == "" {
		return nil
	}
	return m.specimens[fileKey(f)].font
}

// wantSpecimen starts reading the files the font panel is about to show.
func (m *Model) wantSpecimen() tea.Cmd {
	if !m.pixelOn() || m.cmd == nil || m.cmd.preview == nil || !m.fontsLoaded {
		return nil
	}
	fam, style, ok := m.specimenOf(m.selected())
	if !ok {
		return nil
	}
	var need []ghostty.FontFile
	for _, f := range specimenFaces(fam, style) {
		if f.Path == "" {
			continue
		}
		if _, seen := m.specimens[fileKey(f)]; !seen {
			need = append(need, f)
		}
	}
	if len(need) == 0 {
		return nil
	}
	if m.specimens == nil || len(m.specimens)+len(need) > specimenCap {
		m.specimens = map[string]specimenFont{}
	}
	for _, f := range need {
		m.specimens[fileKey(f)] = specimenFont{}
	}
	return func() tea.Msg {
		out := map[string]specimenFont{}
		for _, f := range need {
			font, err := pix.OpenFont(f.Path, f.PostScript, f.Index)
			out[fileKey(f)] = specimenFont{font, err}
		}
		return specimenMsg{out}
	}
}

// pxFontSide is the font panel of the picture interface: the family set in
// itself.
func (m *Model) pxFontSide(c *pix.Canvas, p pxPal, f pxFonts, opt *option, x, y, w, h float32) bool {
	fam, style, ok := m.specimenOf(opt)
	if !ok {
		return false
	}
	files := specimenFaces(fam, style)
	if files[0].Path == "" {
		return false
	}
	if sf, ok := m.specimens[fileKey(files[0])]; ok && sf.err != nil {
		return false
	}
	c.Clip(x, y, w, h)
	defer c.Unclip()

	c.Text(f.h3, x, y+18, c.Fit(f.h3, fam.Name, w), p.fg)
	kind := "monospace"
	if !fam.Mono {
		kind = "proportional: columns will not line up"
	}
	meta := fmt.Sprintf("%s · %s · %s pt", kind, style, ghostty.FormatSize(m.fontSize))
	c.Text(f.small, x, y+44, c.Fit(f.small, meta, w), p.muted)
	yy := y + 66
	if len(fam.Styles) > 1 {
		for _, line := range c.Wrap(f.small, strings.Join(fam.Styles, ", "), w-12) {
			c.Text(f.small, x, yy+12, line, p.text2)
			yy += 19
		}
	}
	yy += 18

	regular, bold, italic := m.specimenFont(files[0]), m.specimenFont(files[1]), m.specimenFont(files[2])
	if regular != nil {
		big := c.FaceIn(regular, 30)
		c.Text(big, x, yy+30, c.Fit(big, "Aa Gg 0O 1lI", w), p.fg)
		yy += 50
		body := c.FaceIn(regular, 14)
		for _, s := range []string{"ABCDEFGHIJKLMNOPQRSTUVWXYZ", "abcdefghijklmnopqrstuvwxyz", "0123456789  {}[]()<>  !@#$%&*"} {
			c.Text(body, x, yy+16, c.Fit(body, s, w), p.fg)
			yy += 24
		}
		yy += 6
		if bold == nil {
			bold = regular
		}
		if italic == nil {
			italic = regular
		}
		c.Text(c.FaceIn(bold, 14), x, yy+16, c.Fit(c.FaceIn(bold, 14), "The quick brown fox, in bold", w), p.fg)
		yy += 24
		c.Text(c.FaceIn(italic, 14), x, yy+16, c.Fit(c.FaceIn(italic, 14), "The quick brown fox, in italic", w), p.fg)
		yy += 34
		lw := float32(0)
		for _, l := range []string{"look-alikes", "symbols"} {
			lw = maxF(lw, c.Width(f.small, l))
		}
		for _, row := range [][2]string{{"look-alikes", "0O  1lI|  5S  8B  `'\""}, {"symbols", "-> => != <= >= === ::"}} {
			c.Text(f.small, x, yy+16, row[0], p.muted)
			c.Text(body, x+lw+16, yy+16, c.Fit(body, row[1], w-lw-16), p.fg)
			yy += 24
		}
	} else {
		yy += 80
	}

	yy += 18
	note := "Moving the highlight only looks; Enter writes it"
	if m.autoReloadWorks() {
		note += " and Ghostty reloads."
	} else {
		note += "; " + m.reloadHint() + "."
	}
	for _, line := range c.Wrap(f.body, note, w-12) {
		c.Text(f.body, x, yy+14, line, p.text2)
		yy += 21
	}
	m.pxRegion(c, x, y, w, h, hitPreview, 0, "")
	return true
}
