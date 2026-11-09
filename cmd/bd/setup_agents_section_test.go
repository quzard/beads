package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// resetAgentsSectionSetupFlags 把 bd setup 的包级开关拨回默认值，避免别的测试留下的
// --check / --remove 让这一轮装到错误的分支。
func resetAgentsSectionSetupFlags(t *testing.T) {
	t.Helper()
	project, global, check, remove := setupProject, setupGlobal, setupCheck, setupRemove
	stealth, print, list := setupStealth, setupPrint, setupList
	output, add := setupOutput, setupAdd
	setupProject, setupGlobal, setupCheck, setupRemove = false, false, false, false
	setupStealth, setupPrint, setupList = false, false, false
	setupOutput, setupAdd = "", ""
	t.Cleanup(func() {
		setupProject, setupGlobal, setupCheck, setupRemove = project, global, check, remove
		setupStealth, setupPrint, setupList = stealth, print, list
		setupOutput, setupAdd = output, add
	})
}

func TestSetupListIncludesAgentsSectionRecipes(t *testing.T) {
	resetAgentsSectionSetupFlags(t)
	out := captureStdout(t, listRecipes)
	for _, needle := range []string{
		"grok",
		"Grok CLI AGENTS.md section + SessionStart hook and rule",
		"qoder",
		"Qoder CLI AGENTS.md section + SessionStart hook",
		"omp",
		"OMP AGENTS.md section + session hook and always-on rule",
		"antigravity",
		"Antigravity AGENTS.md section + SessionStart hook and rule",
	} {
		if !strings.Contains(out, needle) {
			t.Errorf("bd setup --list missing %q\n%s", needle, out)
		}
	}
}

func TestSetupAgentsSectionRecipeRoundTrip(t *testing.T) {
	for _, name := range []string{"grok", "qoder", "omp", "antigravity"} {
		t.Run(name, func(t *testing.T) {
			resetAgentsSectionSetupFlags(t)
			dir := t.TempDir()
			t.Chdir(dir)

			if err := runRecipe(name); err != nil {
				t.Fatalf("bd setup %s: %v", name, err)
			}
			path := filepath.Join(dir, "AGENTS.md")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read AGENTS.md: %v", err)
			}
			if strings.Count(string(data), "<!-- BEGIN BEADS INTEGRATION") != 1 {
				t.Fatalf("AGENTS.md:\n%s", data)
			}

			again := captureStdout(t, func() error { return runRecipe(name) })
			if !strings.Contains(again, "already has the current beads section") {
				t.Fatalf("second install output:\n%s", again)
			}
			data2, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(data2) != string(data) {
				t.Fatal("second install changed AGENTS.md")
			}

			setupCheck = true
			if err := runRecipe(name); err != nil {
				t.Fatalf("bd setup %s --check: %v", name, err)
			}
			setupCheck = false
			setupRemove = true
			if err := runRecipe(name); err != nil {
				t.Fatalf("bd setup %s --remove: %v", name, err)
			}
			data, err = os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "BEGIN BEADS INTEGRATION") {
				t.Fatalf("section remained:\n%s", data)
			}
		})
	}
}
