package pix

import (
	"strings"
	"testing"

	"golang.org/x/image/font/basicfont"

	"github.com/vitruves/ghostty-config/internal/kimg"
)

func TestRunsDrawsHanFromTheEmbeddedFace(t *testing.T) {
	f := kimg.Face(false, false, 16)
	if !kimg.CJKCovers('主') {
		t.Fatal("embedded face has no 主")
	}
	got := runs(f, "主題")
	if len(got) != 1 || strings.Contains(got[0].s, "?") || got[0].f == f {
		t.Fatalf("runs = %#v", got)
	}
	// A face this package did not hand out is a specimen font. It is left
	// alone, even when it has no Han.
	if kimg.CJKFor(basicfont.Face7x13) != nil {
		t.Fatal("CJK fallback ran for a face that is not one of ours")
	}
}

func TestWrapSplitsHanAndKeepsSpacedEnglish(t *testing.T) {
	c := New(400, 80, 1)
	f := c.Face(false, false, 16)
	limit := c.Width(f, "主題牆")
	lines := c.Wrap(f, "主題牆會依家族分組", limit)
	if len(lines) < 2 {
		t.Fatalf("no break: %#v", lines)
	}
	eng := c.Wrap(f, "one two three", c.Width(f, "one two")+0.5)
	if strings.Join(eng, "|") != "one two|three" {
		t.Fatalf("english wrap: %#v", eng)
	}
	if strings.Join(c.Wrap(f, "a\n\nb", 400), "|") != "a||b" {
		t.Fatal("empty paragraph was dropped")
	}
}
