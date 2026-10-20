package ghostty

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveFollowsSymlinksAndKeepsMode(t *testing.T) {
	p, dir := fixture(t, map[string]string{"dotfiles/ghostty": "font-size = 10\n"})
	real := filepath.Join(dir, "dotfiles", "ghostty")
	if err := os.Chmod(real, 0o600); err != nil {
		t.Fatal(err)
	}
	link := p.Roots[0]
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../dotfiles/ghostty", link); err != nil {
		t.Fatal(err)
	}
	tree, _ := Load(p)
	tree.Set("font-size", "12")
	if _, err := tree.Save(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("the config link was replaced by a regular file")
	}
	data, _ := os.ReadFile(real)
	if string(data) != "font-size = 12\n" {
		t.Fatalf("the link target holds %q", data)
	}
	if info, _ := os.Stat(real); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, "*", ".*.tmp-*"))
	if len(leftovers) != 0 {
		t.Fatalf("temporary files left behind: %v", leftovers)
	}

	// A file created here gets the default mode.
	fresh := filepath.Join(dir, "new", "config")
	if err := writeFileAtomic(fresh, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(fresh); info.Mode().Perm() != 0o644 {
		t.Fatalf("new file mode = %v, want 0644", info.Mode().Perm())
	}
}

func TestBackupsOfSameNamedFilesDoNotCollide(t *testing.T) {
	p, dir := fixture(t, map[string]string{
		"xdg/config":     "font-size = 10\n",
		"support/config": "font-size = 11\n",
	})
	out, err := BackupNow(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 || out[0] == out[1] {
		t.Fatalf("BackupNow returned %v", out)
	}
	for i, want := range []string{"font-size = 10\n", "font-size = 11\n"} {
		if data, _ := os.ReadFile(out[i]); string(data) != want {
			t.Fatalf("%s holds %q, want %q", out[i], data, want)
		}
	}

	// A save touching both roots in the same second keeps both originals.
	tree, _ := Load(p)
	tree.Docs[0].SetLine(0, "font-size", "20")
	tree.Docs[1].SetLine(0, "font-size", "21")
	if _, err := tree.Save(); err != nil {
		t.Fatal(err)
	}
	backups, _ := os.ReadDir(filepath.Join(dir, "state", "backups"))
	if len(backups) != 4 {
		t.Fatalf("expected 4 backups, got %d", len(backups))
	}
}
