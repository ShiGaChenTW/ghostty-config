package kimg

import (
	_ "embed"
	"os"
	"strings"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

// GhosttyConfigSansTC is a subset of Noto Sans TC, renamed because a subset
// is a derivative and the SIL OFL reserves the name Noto. One regular weight
// covers bold headings too. The face is not stored in faceInfo: Covers treats
// an unknown face as covering every rune, which is how a user's specimen font
// is left alone, and which would also hide a missing CJK glyph.

//go:embed fonts/GhosttyConfigSansTC-Regular.otf
var cjkRegular []byte

var (
	cjkOnce  sync.Once
	cjkFont  *opentype.Font
	cjkFaces = map[int]font.Face{}

	sysOnce  sync.Once
	sysFont  *opentype.Font
	sysFaces = map[int]font.Face{}
)

func loadCJK() {
	cjkOnce.Do(func() {
		f, err := opentype.Parse(cjkRegular)
		if err != nil {
			return
		}
		cjkFont = f
	})
}

func glyphCovers(f *opentype.Font, r rune) bool {
	if f == nil {
		return false
	}
	var b sfnt.Buffer
	i, err := f.GlyphIndex(&b, r)
	return err == nil && i != 0
}

// CJKCovers reports whether the embedded Traditional face has a glyph for r.
func CJKCovers(r rune) bool {
	loadCJK()
	return glyphCovers(cjkFont, r)
}

// CJKFor is the embedded Traditional face at the same size as f. Bold uses
// the same regular face. It is nil when f was not made by Face, so a specimen
// drawn in the user's own font is not rewritten.
func CJKFor(f font.Face) font.Face {
	faceMu.Lock()
	k, ok := faceInfo[f]
	faceMu.Unlock()
	if !ok {
		return nil
	}
	return sizedFace(&cjkFont, cjkFaces, float64(k.px)/10, loadCJK)
}

// SystemCJK is a Traditional face installed on this machine, for characters
// outside the embedded subset (a theme name, a search). Simplified faces are
// not used. Nil when this machine has no identifiable Traditional font, or
// when f was not made by Face.
func SystemCJK(f font.Face) font.Face {
	faceMu.Lock()
	k, ok := faceInfo[f]
	faceMu.Unlock()
	if !ok {
		return nil
	}
	return sizedFace(&sysFont, sysFaces, float64(k.px)/10, loadSystemCJK)
}

// SystemCJKCovers reports whether the system Traditional face has r.
func SystemCJKCovers(r rune) bool {
	loadSystemCJK()
	return glyphCovers(sysFont, r)
}

func sizedFace(src **opentype.Font, cache map[int]font.Face, px float64, load func()) font.Face {
	load()
	if *src == nil {
		return nil
	}
	key := int(px*10 + 0.5)
	faceMu.Lock()
	defer faceMu.Unlock()
	if face, ok := cache[key]; ok {
		return face
	}
	face, err := opentype.NewFace(*src, &opentype.FaceOptions{Size: float64(key) / 10, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil
	}
	cache[key] = face
	return face
}

// Traditional Chinese fonts this tool will use as a fallback. A collection
// may also contain a Simplified face; only a face whose name says Traditional
// is kept.
var systemCJKPaths = []string{
	"/System/Library/Fonts/PingFang.ttc",
	"/System/Library/Fonts/Supplemental/Songti.ttc",
	"/usr/share/fonts/opentype/noto/NotoSansCJKtc-Regular.otf",
	"/usr/share/fonts/opentype/noto/NotoSansTC-Regular.otf",
	"/usr/share/fonts/truetype/noto/NotoSansTC-Regular.ttf",
	"/usr/share/fonts/opentype/source-han-sans/SourceHanSansTC-Regular.otf",
	"/usr/share/fonts/opentype/source-han-sans/SourceHanSansHK-Regular.otf",
}

func loadSystemCJK() {
	sysOnce.Do(func() {
		for _, path := range systemCJKPaths {
			b, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			col, err := opentype.ParseCollection(b)
			if err != nil {
				continue
			}
			for i := 0; i < col.NumFonts(); i++ {
				f, err := col.Font(i)
				if err != nil || !isTraditional(f) {
					continue
				}
				sysFont = f
				return
			}
		}
	})
}

func isTraditional(f *opentype.Font) bool {
	var b sfnt.Buffer
	var bld strings.Builder
	for _, id := range []sfnt.NameID{sfnt.NameIDFamily, sfnt.NameIDFull, sfnt.NameIDTypographicFamily, sfnt.NameIDPostScript} {
		s, err := f.Name(&b, id)
		if err == nil {
			bld.WriteString(strings.ToLower(s))
			bld.WriteByte(' ')
		}
	}
	n := bld.String()
	traditional := strings.Contains(n, "hant") || strings.Contains(n, "traditional") ||
		strings.Contains(n, " tc") || strings.Contains(n, " hk") ||
		strings.Contains(n, "pingfangtc") || strings.Contains(n, "pingfanghk") ||
		strings.Contains(n, "notosanstc") || strings.Contains(n, "notosanscjktc") ||
		strings.Contains(n, "sourcehansanstc") || strings.Contains(n, "sourcehansanshk")
	simplified := strings.Contains(n, "hans") || strings.Contains(n, "simplified") ||
		strings.Contains(n, " sc") || strings.Contains(n, "pingfangsc") ||
		strings.Contains(n, "notosanssc") || strings.Contains(n, "sourcehansanssc")
	return traditional && !simplified
}
