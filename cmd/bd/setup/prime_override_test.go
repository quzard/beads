package setup

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/templates"
	"github.com/steveyegge/beads/internal/templates/agents"
)

// defaultRulesConflictingWithPrime 是默认模板里和 PRIME.md 覆盖冲突的规则。
var defaultRulesConflictingWithPrime = []string{
	"ALL task tracking",
	"Track ALL work",
	"bd remember",
	"TodoWrite",
	"markdown TODO",
	"Core CLI Workflow",
}

// withPrimeOverride 让测试期间的生成代码认为有（或没有）PRIME.md 覆盖。
func withPrimeOverride(t *testing.T, active bool) {
	t.Helper()
	prev := primeOverrideActiveImpl
	primeOverrideActiveImpl = func(bool) bool { return active }
	t.Cleanup(func() { primeOverrideActiveImpl = prev })
}

// isolateUserConfigDir points os.UserConfigDir at a fresh directory and returns it.
func isolateUserConfigDir(t *testing.T) string {
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

func writePrimeOverride(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "PRIME.md")
	if err := os.WriteFile(path, []byte("# Custom workflow\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	return path
}

func TestPrimeOverrideFromConfig(t *testing.T) {
	cfgDir := isolateUserConfigDir(t)
	t.Chdir(t.TempDir())

	if primeOverrideFromConfig(false) || primeOverrideFromConfig(true) {
		t.Fatal("no PRIME.md anywhere: want no override for project or user-level files")
	}

	// 项目里的覆盖只影响项目级文件：用户级文件对所有仓库生效。
	local := writePrimeOverride(t, ".beads")
	if !primeOverrideFromConfig(false) {
		t.Error(".beads/PRIME.md: want an override for project files")
	}
	if primeOverrideFromConfig(true) {
		t.Error(".beads/PRIME.md: want no override for user-level files")
	}

	if err := os.Remove(local); err != nil {
		t.Fatal(err)
	}
	writePrimeOverride(t, filepath.Join(cfgDir, "beads"))
	if !primeOverrideFromConfig(false) || !primeOverrideFromConfig(true) {
		t.Error("user config PRIME.md: want an override for project and user-level files")
	}
}

func TestGeneratedInstructionsPointToPrimeUnderOverride(t *testing.T) {
	withPrimeOverride(t, true)
	pointer := agents.PrimePointerText()

	generated := map[string]string{
		"codex project section": codexManagedSection(false),
		"codex global section":  codexManagedSection(true),
		"cursor rules":          cursorRulesContent(),
		"skill":                 agentSkillContent(false),
	}
	for name, content := range generated {
		if !strings.Contains(content, pointer) {
			t.Errorf("%s: missing the bd prime pointer:\n%s", name, content)
		}
		for _, rule := range defaultRulesConflictingWithPrime {
			if strings.Contains(content, rule) {
				t.Errorf("%s: still carries default rule %q:\n%s", name, rule, content)
			}
		}
	}

	rules := cursorRulesContent()
	if !strings.HasPrefix(rules, "---\nalwaysApply: true\n---\n") || !strings.Contains(rules, "<!-- END BEADS INTEGRATION -->") {
		t.Errorf("cursor rules lost the frontmatter or markers:\n%s", rules)
	}
	frontmatter, ok := skillFrontmatter(templates.BeadsAgentSkill())
	if !ok || !strings.HasPrefix(agentSkillContent(false), frontmatter) {
		t.Errorf("skill lost its frontmatter:\n%s", agentSkillContent(false))
	}
	if !detectRenderOpts().CustomPrime {
		t.Error("detectRenderOpts: want CustomPrime")
	}
	if opts := userLevelRenderOpts(); !opts.CustomPrime || !opts.UserLevel {
		t.Errorf("userLevelRenderOpts = %+v, want CustomPrime and UserLevel", opts)
	}
}

func TestGeneratedInstructionsUnchangedWithoutOverride(t *testing.T) {
	withPrimeOverride(t, false)
	if got := cursorRulesContent(); got != cursorRulesTemplate {
		t.Errorf("cursor rules changed without an override:\n%s", got)
	}
	if got := agentSkillContent(false); got != templates.BeadsAgentSkill() {
		t.Errorf("skill changed without an override:\n%s", got)
	}
	if got := codexManagedSection(false); !strings.Contains(got, agents.CodexSectionBody()) {
		t.Errorf("codex section changed without an override:\n%s", got)
	}
	if detectRenderOpts().CustomPrime || userLevelRenderOpts().CustomPrime {
		t.Error("render opts: want no CustomPrime without an override")
	}
}

// 已有完整段的 AGENTS.md：有覆盖后 --check 报过期，重装换成指针，之后 --check 通过。
func TestInstallAgentsReplacesSectionUnderOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	stdout := &bytes.Buffer{}
	env := agentsEnv{agentsPath: path, stdout: stdout, stderr: &bytes.Buffer{}}
	integration := agentsIntegration{name: "Test", setupCommand: "bd setup test", profile: agents.ProfileFull}

	withPrimeOverride(t, false)
	if err := installAgents(env, integration); err != nil {
		t.Fatalf("install without override: %v\n%s", err, stdout)
	}

	withPrimeOverride(t, true)
	if err := checkAgents(env, integration); !errors.Is(err, errBeadsSectionStale) {
		t.Fatalf("check before reinstall = %v, want errBeadsSectionStale\n%s", err, stdout)
	}
	if err := installAgents(env, integration); err != nil {
		t.Fatalf("install with override: %v\n%s", err, stdout)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, agents.PrimePointerBody()) {
		t.Errorf("AGENTS.md missing the pointer section:\n%s", content)
	}
	if strings.Contains(content, "## Issue Tracking with bd (beads)") {
		t.Errorf("AGENTS.md still has the full section:\n%s", content)
	}
	if err := checkAgents(env, integration); err != nil {
		t.Errorf("check after reinstall = %v, want current\n%s", err, stdout)
	}
}

func TestCodexInstallAndCheckUnderOverride(t *testing.T) {
	withPrimeOverride(t, true)
	env, stdout, _ := newCodexTestEnv(t)

	if err := installCodexInstructions(env, false); err != nil {
		t.Fatalf("install instructions: %v\n%s", err, stdout)
	}
	if err := checkCodexInstructions(env, false); err != nil {
		t.Errorf("check instructions = %v, want current\n%s", err, stdout)
	}
	skillEnv := codexAgentSkillEnv(env, false)
	if err := installAgentSkill(skillEnv); err != nil {
		t.Fatalf("install skill: %v\n%s", err, stdout)
	}
	if err := checkAgentSkill(skillEnv, "bd setup codex"); err != nil {
		t.Errorf("check skill = %v, want current\n%s", err, stdout)
	}
}

func TestAgentSkillTemplateHasFrontmatter(t *testing.T) {
	frontmatter, ok := skillFrontmatter(templates.BeadsAgentSkill())
	if !ok || !strings.HasPrefix(frontmatter, "---\nname: beads\n") || !strings.HasSuffix(frontmatter, "\n---\n") {
		t.Fatalf("skill template frontmatter = %q (ok=%v), want a leading name: beads block", frontmatter, ok)
	}
}
