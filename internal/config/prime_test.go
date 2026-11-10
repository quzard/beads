package config

import (
	"os"
	"path/filepath"
	"testing"
)

// isolatePrimeUserConfigDir points os.UserConfigDir at a fresh directory and returns it.
func isolatePrimeUserConfigDir(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg-config"))
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("UserConfigDir: %v", err)
	}
	return dir
}

func writePrimeFile(t *testing.T, dir, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "PRIME.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

// 覆盖按 bd prime 的顺序生效：工作区的 .beads、解析出的 beads 目录、用户配置目录。
func TestReadPrimeOverrideOrder(t *testing.T) {
	cfgDir := isolatePrimeUserConfigDir(t)
	workspace := t.TempDir()
	redirected := t.TempDir()

	steps := []struct {
		want string
		path string
	}{
		{"local", writePrimeFile(t, filepath.Join(workspace, ".beads"), "local")},
		{"redirected", writePrimeFile(t, redirected, "redirected")},
		{"global", writePrimeFile(t, filepath.Join(cfgDir, "beads"), "global")},
	}
	for _, step := range steps {
		got, ok := ReadPrimeOverride(workspace, redirected)
		if !ok || got != step.want {
			t.Fatalf("ReadPrimeOverride = (%q, %v), want (%q, true)", got, ok, step.want)
		}
		if err := os.Remove(step.path); err != nil {
			t.Fatal(err)
		}
	}
	if got, ok := ReadPrimeOverride(workspace, redirected); ok {
		t.Fatalf("no PRIME.md left: ReadPrimeOverride = (%q, true), want no override", got)
	}
}

// beadsDir 为空时，当前目录下的 PRIME.md（不在 .beads 里）不算覆盖。
func TestReadPrimeOverrideEmptyBeadsDir(t *testing.T) {
	isolatePrimeUserConfigDir(t)
	dir := t.TempDir()
	t.Chdir(dir)
	writePrimeFile(t, dir, "stray")

	if got, ok := ReadPrimeOverride("", ""); ok {
		t.Fatalf("ReadPrimeOverride(\"\", \"\") = (%q, true), want no override", got)
	}
	writePrimeFile(t, filepath.Join(dir, ".beads"), "local")
	if got, ok := ReadPrimeOverride("", ""); !ok || got != "local" {
		t.Fatalf("ReadPrimeOverride(\"\", \"\") = (%q, %v), want (\"local\", true)", got, ok)
	}
}
