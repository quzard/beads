package setup

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/steveyegge/beads/internal/templates/agents"
)

// sectionRecipe 是写入同一份 AGENTS.md 托管段的配方。安装、检查、删除都走 installAgents。
type sectionRecipe struct {
	name         string
	setupCommand string
	readHint     string
	docsURL      string
	install      func(agentsEnv) error
	check        func(agentsEnv) error
	remove       func(agentsEnv) error
	installPub   func() error
	checkPub     func() error
	stub         func(t *testing.T, env agentsEnv)
}

func agentsSectionRecipes() []sectionRecipe {
	return []sectionRecipe{
		{
			name:         "Grok",
			setupCommand: "bd setup grok",
			readHint:     grokIntegration.readHint,
			docsURL:      grokProjectRulesURL,
			install:      installGrok,
			check:        checkGrok,
			remove:       removeGrok,
			installPub:   InstallGrok,
			checkPub:     CheckGrok,
			stub:         stubGrokEnv,
		},
		{
			name:         "Qoder CLI",
			setupCommand: "bd setup qoder",
			readHint:     qoderIntegration.readHint,
			docsURL:      qoderMemoryURL,
			install:      installQoder,
			check:        checkQoder,
			remove:       removeQoder,
			installPub:   InstallQoder,
			checkPub:     CheckQoder,
			stub:         stubQoderEnv,
		},
		{
			name:         "OMP",
			setupCommand: "bd setup omp",
			readHint:     ompIntegration.readHint,
			docsURL:      ompContextFilesURL,
			install:      installOMP,
			check:        checkOMP,
			remove:       removeOMP,
			installPub:   InstallOMP,
			checkPub:     CheckOMP,
			stub:         stubOMPEnv,
		},
		{
			name:         "Antigravity",
			setupCommand: "bd setup antigravity",
			readHint:     antigravityIntegration.readHint,
			docsURL:      antigravityRulesURL,
			install:      installAntigravity,
			check:        checkAntigravity,
			remove:       removeAntigravity,
			installPub:   InstallAntigravity,
			checkPub:     CheckAntigravity,
			stub:         stubAntigravityEnv,
		},
	}
}

func stubGrokEnv(t *testing.T, env agentsEnv) {
	t.Helper()
	orig := grokEnvProvider
	grokEnvProvider = func() agentsEnv { return env }
	t.Cleanup(func() { grokEnvProvider = orig })
}

func stubQoderEnv(t *testing.T, env agentsEnv) {
	t.Helper()
	orig := qoderEnvProvider
	qoderEnvProvider = func() agentsEnv { return env }
	t.Cleanup(func() { qoderEnvProvider = orig })
}

func stubOMPEnv(t *testing.T, env agentsEnv) {
	t.Helper()
	orig := ompEnvProvider
	ompEnvProvider = func() agentsEnv { return env }
	t.Cleanup(func() { ompEnvProvider = orig })
}

func stubAntigravityEnv(t *testing.T, env agentsEnv) {
	t.Helper()
	orig := antigravityEnvProvider
	antigravityEnvProvider = func() agentsEnv { return env }
	t.Cleanup(func() { antigravityEnvProvider = orig })
}

func TestAgentsSectionRecipesInstallCheckRemove(t *testing.T) {
	stubDetectRenderOpts(t)

	for _, recipe := range agentsSectionRecipes() {
		t.Run(recipe.name, func(t *testing.T) {
			env, stdout, _ := newFactoryTestEnv(t)
			if err := recipe.install(env); err != nil {
				t.Fatalf("install returned error: %v", err)
			}
			data, err := os.ReadFile(env.agentsPath)
			if err != nil {
				t.Fatalf("read AGENTS.md: %v", err)
			}
			content := string(data)
			if strings.Count(content, "<!-- BEGIN BEADS INTEGRATION") != 1 || !strings.Contains(content, agentsEndMarker) {
				t.Fatalf("expected one beads section, got:\n%s", content)
			}
			out := stdout.String()
			if !strings.Contains(out, recipe.name+" integration installed") {
				t.Errorf("stdout missing install success:\n%s", out)
			}
			if !strings.Contains(out, recipe.readHint) {
				t.Errorf("stdout missing read hint %q:\n%s", recipe.readHint, out)
			}
			if !strings.Contains(out, "Review guide: "+recipe.docsURL) {
				t.Errorf("stdout missing review guide:\n%s", out)
			}

			// 内容已是当前版本时，再装一次不改文件，只报告已是最新。
			stdout.Reset()
			before := content
			if err := recipe.install(env); err != nil {
				t.Fatalf("second install returned error: %v", err)
			}
			data, err = os.ReadFile(env.agentsPath)
			if err != nil {
				t.Fatalf("read AGENTS.md after second install: %v", err)
			}
			if string(data) != before {
				t.Fatal("second install rewrote a current AGENTS.md")
			}
			if !strings.Contains(stdout.String(), "already has the current beads section") {
				t.Errorf("second install stdout:\n%s", stdout.String())
			}

			stdout.Reset()
			if err := recipe.check(env); err != nil {
				t.Fatalf("check returned error: %v", err)
			}
			if !strings.Contains(stdout.String(), "integration installed") || !strings.Contains(stdout.String(), "(current)") {
				t.Errorf("check stdout:\n%s", stdout.String())
			}

			// 旧段落会被替换，段落外面的内容保留。
			stale := "# Header\n\n<!-- BEGIN BEADS INTEGRATION -->\nOld content\n<!-- END BEADS INTEGRATION -->\n\n# Footer\n"
			if err := os.WriteFile(env.agentsPath, []byte(stale), 0o644); err != nil {
				t.Fatalf("seed stale file: %v", err)
			}
			if err := recipe.install(env); err != nil {
				t.Fatalf("update install returned error: %v", err)
			}
			data, err = os.ReadFile(env.agentsPath)
			if err != nil {
				t.Fatalf("read updated AGENTS.md: %v", err)
			}
			updated := string(data)
			if strings.Contains(updated, "Old content") {
				t.Fatal("stale beads section was kept")
			}
			if !strings.Contains(updated, "# Header") || !strings.Contains(updated, "# Footer") {
				t.Fatalf("surrounding content dropped:\n%s", updated)
			}
			if strings.Count(updated, "<!-- BEGIN BEADS INTEGRATION") != 1 {
				t.Fatalf("expected one beads section after update:\n%s", updated)
			}

			stdout.Reset()
			if err := recipe.remove(env); err != nil {
				t.Fatalf("remove returned error: %v", err)
			}
			data, err = os.ReadFile(env.agentsPath)
			if err != nil {
				t.Fatalf("read AGENTS.md after remove: %v", err)
			}
			if strings.Contains(string(data), "BEGIN BEADS INTEGRATION") {
				t.Fatalf("beads section still present:\n%s", data)
			}
			if !strings.Contains(string(data), "# Header") || !strings.Contains(string(data), "# Footer") {
				t.Fatalf("remove changed surrounding content:\n%s", data)
			}
			if !strings.Contains(stdout.String(), "Removed beads section") {
				t.Errorf("remove stdout:\n%s", stdout.String())
			}
		})
	}
}

func TestAgentsSectionRecipesMissingFile(t *testing.T) {
	for _, recipe := range agentsSectionRecipes() {
		t.Run(recipe.name, func(t *testing.T) {
			env, stdout, _ := newFactoryTestEnv(t)
			err := recipe.check(env)
			if !errors.Is(err, errAgentsFileMissing) {
				t.Fatalf("expected errAgentsFileMissing, got %v", err)
			}
			if !strings.Contains(stdout.String(), "Run: "+recipe.setupCommand) {
				t.Errorf("stdout missing setup guidance:\n%s", stdout.String())
			}

			stdout.Reset()
			if err := recipe.remove(env); err != nil {
				t.Fatalf("remove missing file: %v", err)
			}
			if !strings.Contains(stdout.String(), "No AGENTS.md file found") {
				t.Errorf("remove stdout:\n%s", stdout.String())
			}
		})
	}
}

func TestAgentsSectionRecipesShareOneSection(t *testing.T) {
	stubDetectRenderOpts(t)
	env, stdout, _ := newFactoryTestEnv(t)
	if err := installGrok(env); err != nil {
		t.Fatalf("install grok: %v", err)
	}
	stdout.Reset()
	if err := installQoder(env); err != nil {
		t.Fatalf("install qoder: %v", err)
	}
	if err := installOMP(env); err != nil {
		t.Fatalf("install omp: %v", err)
	}
	if err := installAntigravity(env); err != nil {
		t.Fatalf("install antigravity: %v", err)
	}
	data, err := os.ReadFile(env.agentsPath)
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if strings.Count(string(data), "<!-- BEGIN BEADS INTEGRATION") != 1 {
		t.Fatalf("recipes wrote more than one beads section:\n%s", data)
	}
	if !strings.Contains(stdout.String(), "already has the current beads section") {
		t.Errorf("later recipes should see the grok section as current:\n%s", stdout.String())
	}
}

// TestInitFullSectionSatisfiesAgentsRecipes 确认 bd init 默认写入的 full 段
// 能通过这些配方的 --check。无 remote 的 RenderOpts 与没有 Dolt remote 的 init 一致。
func TestInitFullSectionSatisfiesAgentsRecipes(t *testing.T) {
	opts := agents.RenderOpts{}
	orig := detectRenderOptsImpl
	detectRenderOptsImpl = func() agents.RenderOpts { return opts }
	t.Cleanup(func() { detectRenderOptsImpl = orig })

	content := agents.EmbeddedDefaultWithOpts(opts)
	replaced, _, err := agents.ReplaceSectionWithOpts(content, agents.ProfileFull, opts)
	if err != nil {
		t.Fatalf("replace section: %v", err)
	}
	env, stdout, _ := newFactoryTestEnv(t)
	if err := os.WriteFile(env.agentsPath, []byte(replaced), 0o644); err != nil {
		t.Fatal(err)
	}
	// 只有 AGENTS.md 段时，钩子配方的 --check 应该失败；装上钩子之后段仍是一份，检查通过。
	// OpenCode 只看段，不装钩子。
	for _, recipe := range agentsSectionRecipes() {
		t.Run(recipe.name+" hooks missing", func(t *testing.T) {
			stdout.Reset()
			if err := recipe.check(env); !errors.Is(err, errSessionPrimeIncomplete) {
				t.Fatalf("check = %v, want incomplete hooks\n%s", err, stdout.String())
			}
		})
	}
	stdout.Reset()
	if err := checkOpenCode(env); err != nil {
		t.Fatalf("opencode check: %v\n%s", err, stdout.String())
	}
	before := replaced
	for _, recipe := range append(agentsSectionRecipes(), sectionRecipe{
		name:    "Factory.ai (Droid)",
		check:   checkFactory,
		install: installFactory,
	}) {
		t.Run(recipe.name+" after hooks", func(t *testing.T) {
			stdout.Reset()
			if err := recipe.install(env); err != nil {
				t.Fatalf("install: %v", err)
			}
			data, err := os.ReadFile(env.agentsPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(data) != before {
				t.Fatal("hook install rewrote a current AGENTS.md section")
			}
			if strings.Count(string(data), "<!-- BEGIN BEADS INTEGRATION") != 1 {
				t.Fatalf("section count changed:\n%s", data)
			}
			stdout.Reset()
			if err := recipe.check(env); err != nil {
				t.Fatalf("check: %v\n%s", err, stdout.String())
			}
			if !strings.Contains(stdout.String(), "(current)") {
				t.Fatalf("stdout:\n%s", stdout.String())
			}
		})
	}

	minimal, _, err := agents.ReplaceSectionWithOpts(content, agents.ProfileMinimal, opts)
	if err != nil {
		t.Fatalf("replace minimal: %v", err)
	}
	if err := os.WriteFile(env.agentsPath, []byte(minimal), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := checkGrok(env); !errors.Is(err, errBeadsSectionStale) {
		t.Fatalf("minimal section check = %v, want stale\n%s", err, stdout.String())
	}
}

func TestAgentsSectionRecipeWrappers(t *testing.T) {
	for _, recipe := range agentsSectionRecipes() {
		t.Run(recipe.name, func(t *testing.T) {
			env, _, stderr := newFactoryTestEnv(t)
			if err := os.Mkdir(env.agentsPath, 0o755); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			recipe.stub(t, env)
			if err := recipe.installPub(); err == nil {
				t.Fatal("install should fail when AGENTS.md path is a directory")
			}
			if !strings.Contains(stderr.String(), "failed to read") {
				t.Errorf("stderr:\n%s", stderr.String())
			}

			missing, _, _ := newFactoryTestEnv(t)
			recipe.stub(t, missing)
			if err := recipe.checkPub(); err == nil {
				t.Fatal("check should fail when AGENTS.md is missing")
			}
		})
	}
}
