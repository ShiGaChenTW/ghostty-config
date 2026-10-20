package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestNormalizeLang(t *testing.T) {
	for _, s := range []string{"zh-Hant", "zh_TW", "zh-tw", "zh_HK", "zh-MO"} {
		if got := NormalizeLang(s); got != langZh {
			t.Errorf("NormalizeLang(%q) = %q", s, got)
		}
	}
	for _, s := range []string{"", "zh_CN", "zh-Hans", "zh-hans", "fr", "zh"} {
		if got := NormalizeLang(s); got != "" {
			t.Errorf("NormalizeLang(%q) = %q, want empty", s, got)
		}
	}
	if got := NormalizeLang("en"); got != langEN {
		t.Errorf("en: %q", got)
	}
}

func TestEnglishIsTheUntranslatedSource(t *testing.T) {
	en := &Model{}
	if en.t("Themes") != "Themes" || en.tf("Search %d themes", 3) != "Search 3 themes" {
		t.Fatalf("english moved: %q %q", en.t("Themes"), en.tf("Search %d themes", 3))
	}
	if en.sentence("Pick a family") != "Pick a family." {
		t.Fatalf("sentence: %q", en.sentence("Pick a family"))
	}
	zh := &Model{langOverride: langZh}
	if zh.t("Themes") != "主題" {
		t.Fatalf("zh Themes: %q", zh.t("Themes"))
	}
	if zh.t("not a catalog key") != "not a catalog key" {
		t.Fatal("missing key was rewritten")
	}
	got := zh.tf("Applied %s as the dark half of your light/dark pair", "Dracula")
	if got != "已套用 Dracula，作為明暗成對裡的深色那一半" {
		t.Fatalf("format: %q", got)
	}
	if zh.sentence("Background opacity") == "background opacity." {
		t.Fatal("chinese sentence was lowercased")
	}
	if got := zh.tf("Downloading %s from %s…", "JetBrains Mono", "ryanoasis/nerd-fonts"); got != "正在從 ryanoasis/nerd-fonts 下載 JetBrains Mono…" {
		t.Fatalf("download order: %q", got)
	}
}

func TestMenuSearchKeepsEnglishAndAddsChinese(t *testing.T) {
	cmd := &command{name: "opacity", desc: "How see-through the background is", set: &setting{label: "Background opacity"}}
	en := &Model{}
	if !en.menuMatches(cmd, "") {
		t.Fatal("empty filter must still match the command name")
	}
	if en.menuMatches(cmd, "不透明") {
		t.Fatal("english search matched a translation")
	}
	zh := &Model{langOverride: langZh}
	if !zh.menuMatches(cmd, "不透明") {
		t.Fatal("chinese search missed the setting label")
	}
	if !zh.menuMatches(cmd, "opacity") {
		t.Fatal("chinese search dropped the english command name")
	}
}

func TestWrapPlainBreaksSpacelessChineseOnly(t *testing.T) {
	s := "主題牆會依家族分組由深到淺排列"
	lines := wrapPlain(s, 16)
	if len(lines) < 2 {
		t.Fatalf("no break: %#v", lines)
	}
	for _, l := range lines {
		if lipgloss.Width(l) > 16 {
			t.Fatalf("line wider than 16: %q (%d)", l, lipgloss.Width(l))
		}
	}
	eng := wrapPlain("the quick brown fox", 10)
	if strings.Join(eng, "|") != "the quick|brown fox" {
		t.Fatalf("english wrap changed: %#v", eng)
	}
	if strings.Join(wrapPlain("a\n\nb", 20), "|") != "a||b" {
		t.Fatal("empty paragraph was dropped")
	}
}
