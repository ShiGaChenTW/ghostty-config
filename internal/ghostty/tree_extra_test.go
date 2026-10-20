package ghostty

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A repeatable key set to one value keeps a single line in effect, and a
// snapshot taken before puts every line back exactly.
func TestSetOnlyAndRestoreKey(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "config")
	original := "custom-shader = /a.glsl\nfont-size = 13\ncustom-shader=/b.glsl\n"
	os.WriteFile(root, []byte(original), 0o644)
	tree, err := Load(Paths{Roots: []string{root}, StateDir: filepath.Join(dir, "state")})
	if err != nil {
		t.Fatal(err)
	}
	was := tree.SnapshotKey("custom-shader")
	tree.SetOnly("custom-shader", "/c.glsl", "disabled by ghostty-config: one shader at a time")
	if all := tree.All("custom-shader"); len(all) != 1 || all[0].Value != "/c.glsl" {
		t.Fatalf("one line should be in effect, are %+v", all)
	}
	if _, err := tree.Save(); err != nil {
		t.Fatal(err)
	}
	tree.RestoreKey(was, "disabled by ghostty-config: no shader")
	if got := tree.Primary.Text(); got != original {
		t.Fatalf("the lines should be back as they were:\n%s", got)
	}

	// With nothing there before, the appended line is switched off again.
	tree, _ = Load(Paths{Roots: []string{root}, StateDir: filepath.Join(dir, "state")})
	tree.Unset("custom-shader", "disabled by ghostty-config: no shader")
	was = tree.SnapshotKey("custom-shader")
	tree.SetOnly("custom-shader", "/c.glsl", "disabled by ghostty-config: one shader at a time")
	tree.RestoreKey(was, "disabled by ghostty-config: no shader")
	if all := tree.All("custom-shader"); len(all) != 0 {
		t.Fatalf("no shader should be in effect, are %+v", all)
	}
}

// Shader lines picked and put back across a save that found the file edited
// elsewhere: the replay and the restore both work from values, since the
// lines have moved.
func TestShaderLinesSurviveAnOutsideEdit(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "config")
	os.WriteFile(root, []byte("custom-shader = /a.glsl\nfont-size = 13\ncustom-shader = /b.glsl\n"), 0o644)
	paths := Paths{Roots: []string{root}, StateDir: filepath.Join(dir, "state")}
	tree, err := Load(paths)
	if err != nil {
		t.Fatal(err)
	}
	was := tree.SnapshotKey("custom-shader")
	tree.SetOnly("custom-shader", "/c.glsl", "disabled by ghostty-config: one shader at a time")

	// Someone adds lines above while the question is open.
	edited := "# mine\nwindow-padding-x = 20\ncustom-shader = /a.glsl\nfont-size = 13\ncustom-shader = /b.glsl\n"
	os.WriteFile(root, []byte(edited), 0o644)
	if _, err := tree.Save(); err != nil {
		t.Fatal(err)
	}
	if all := tree.All("custom-shader"); len(all) != 1 || all[0].Value != "/c.glsl" {
		t.Fatalf("the pick should have been replayed onto the edited file, in effect: %+v", all)
	}
	tree.RestoreKey(was, "disabled by ghostty-config: no shader")
	if _, err := tree.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(root)
	got := string(data)
	if all := tree.All("custom-shader"); len(all) != 2 || all[0].Value != "/a.glsl" || all[1].Value != "/b.glsl" {
		t.Fatalf("both shaders should be back in order, in effect: %+v\n%s", all, got)
	}
	if !strings.Contains(got, "window-padding-x = 20") || !strings.Contains(got, "# mine") {
		t.Fatalf("the outside edit was lost:\n%s", got)
	}
}
