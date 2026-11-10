package setup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSessionPrimeInstallCheckRemove(t *testing.T) {
	stubDetectRenderOpts(t)

	cases := []struct {
		name    string
		install func(agentsEnv) error
		check   func(agentsEnv) error
		remove  func(agentsEnv) error
		files   []string
		// hooksRel 非空时，断言 JSON 里有这条 SessionStart 命令，并且用户原有的钩子还在。
		hooksRel string
		command  string
		wrapped  bool
		note     string
	}{
		{
			name:     "grok",
			install:  installGrok,
			check:    checkGrok,
			remove:   removeGrok,
			files:    []string{grokHooksFile, grokRulesFile},
			hooksRel: grokHooksFile,
			command:  SessionPrimeCommand,
			wrapped:  true,
			note:     "does not inject context",
		},
		{
			name:     "qoder",
			install:  installQoder,
			check:    checkQoder,
			remove:   removeQoder,
			files:    []string{qoderSettingsFile},
			hooksRel: qoderSettingsFile,
			command:  SessionPrimeCommand,
			wrapped:  true,
			note:     "source=compact",
		},
		{
			name:    "omp",
			install: installOMP,
			check:   checkOMP,
			remove:  removeOMP,
			files:   []string{ompHookFile, ompRulesFile},
			note:    "before_agent_start",
		},
		{
			name:     "antigravity",
			install:  installAntigravity,
			check:    checkAntigravity,
			remove:   removeAntigravity,
			files:    []string{antigravityHooksFile, antigravityRulesFile},
			hooksRel: antigravityHooksFile,
			command:  AntigravityPrimeCommand,
			note:     "injectSteps",
		},
		{
			name:     "factory",
			install:  installFactory,
			check:    checkFactory,
			remove:   removeFactory,
			files:    []string{factoryHooksFile},
			hooksRel: factoryHooksFile,
			command:  SessionPrimeCommand,
			note:     "additionalContext",
		},
		{
			name:     "devin",
			install:  installDevin,
			check:    checkDevin,
			remove:   removeDevin,
			files:    []string{devinHooksFile},
			hooksRel: devinHooksFile,
			command:  SessionPrimeCommand,
			note:     "additionalContext",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			env, stdout, _ := newFactoryTestEnv(t)
			root := filepath.Dir(env.agentsPath)
			if tc.hooksRel != "" {
				seedUserHooks(t, filepath.Join(root, tc.hooksRel), tc.wrapped)
			}
			if err := tc.install(env); err != nil {
				t.Fatalf("install: %v", err)
			}
			if !strings.Contains(stdout.String(), tc.note) {
				t.Fatalf("stdout missing note %q:\n%s", tc.note, stdout.String())
			}
			data, err := os.ReadFile(env.agentsPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(string(data), "<!-- BEGIN BEADS INTEGRATION") != 1 {
				t.Fatalf("AGENTS.md markers:\n%s", data)
			}
			for _, rel := range tc.files {
				body, err := os.ReadFile(filepath.Join(root, rel))
				if err != nil {
					t.Fatalf("missing %s: %v", rel, err)
				}
				if strings.Contains(rel, ".json") {
					continue
				}
				if !strings.Contains(string(body), beadsSetupManagedMarker) {
					t.Fatalf("%s missing managed marker", rel)
				}
			}
			if tc.name == "omp" {
				body, err := os.ReadFile(filepath.Join(root, ompHookFile))
				if err != nil {
					t.Fatal(err)
				}
				text := string(body)
				for _, needle := range []string{"session_start", "before_agent_start", "session_compact", `execFileSync("bd", ["prime"]`} {
					if !strings.Contains(text, needle) {
						t.Fatalf("omp hook missing %q", needle)
					}
				}
			}
			if tc.hooksRel != "" {
				assertPrimeCommand(t, filepath.Join(root, tc.hooksRel), tc.wrapped, tc.command, true)
			}

			before := map[string]string{}
			mtimes := map[string]time.Time{}
			for _, rel := range tc.files {
				path := filepath.Join(root, rel)
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				before[rel] = string(body)
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				mtimes[rel] = info.ModTime()
			}
			// 文件系统时间精度不够时，隔一下再装，才能看出有没有重写。
			time.Sleep(20 * time.Millisecond)
			stdout.Reset()
			if err := tc.install(env); err != nil {
				t.Fatalf("second install: %v", err)
			}
			for _, rel := range tc.files {
				path := filepath.Join(root, rel)
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if string(body) != before[rel] {
					t.Fatalf("second install rewrote %s", rel)
				}
				info, err := os.Stat(path)
				if err != nil {
					t.Fatal(err)
				}
				if info.ModTime() != mtimes[rel] {
					t.Fatalf("second install touched mtime of %s", rel)
				}
			}

			stdout.Reset()
			if err := tc.check(env); err != nil {
				t.Fatalf("check: %v\n%s", err, stdout.String())
			}

			stdout.Reset()
			if err := tc.remove(env); err != nil {
				t.Fatalf("remove: %v", err)
			}
			agents, err := os.ReadFile(env.agentsPath)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(agents), "BEGIN BEADS INTEGRATION") {
				t.Fatalf("section remained:\n%s", agents)
			}
			for _, rel := range tc.files {
				path := filepath.Join(root, rel)
				if tc.hooksRel == rel {
					// 用户自己的 PreToolUse 还在，我们的 SessionStart 没了。
					assertPrimeCommand(t, path, tc.wrapped, tc.command, false)
					continue
				}
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("%s still present: %v", rel, err)
				}
			}
		})
	}
}

func TestSessionPrimeKeepsUnrelatedAntigravityFiles(t *testing.T) {
	stubDetectRenderOpts(t)
	env, _, _ := newFactoryTestEnv(t)
	root := filepath.Dir(env.agentsPath)
	skill := filepath.Join(root, ".agents", "skills", "beads", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skill), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skill, []byte("skill\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installAntigravity(env); err != nil {
		t.Fatal(err)
	}
	if err := removeAntigravity(env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(skill); err != nil {
		t.Fatalf("skill removed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, antigravityHooksFile)); !os.IsNotExist(err) {
		t.Fatalf("hooks.json status: %v", err)
	}
}

func TestSessionPrimeLeavesUnmanagedRule(t *testing.T) {
	stubDetectRenderOpts(t)
	env, stdout, _ := newFactoryTestEnv(t)
	root := filepath.Dir(env.agentsPath)
	rule := filepath.Join(root, grokRulesFile)
	if err := os.MkdirAll(filepath.Dir(rule), 0o755); err != nil {
		t.Fatal(err)
	}
	const userRule = "# mine\n"
	if err := os.WriteFile(rule, []byte(userRule), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installGrok(env); err != nil {
		t.Fatal(err)
	}
	// 安装会按托管内容覆盖同名规则；用户文件没有标记时，删配方不会再删一份已经不是我们的文件。
	if err := os.WriteFile(rule, []byte(userRule), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	if err := removeGrok(env); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(rule)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != userRule {
		t.Fatalf("unmanaged rule changed: %q", body)
	}
	if !strings.Contains(stdout.String(), "not beads-managed") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}

func TestSessionPrimeRejectsInvalidHooksJSON(t *testing.T) {
	stubDetectRenderOpts(t)
	env, _, stderr := newFactoryTestEnv(t)
	path := filepath.Join(filepath.Dir(env.agentsPath), qoderSettingsFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := installQoder(env); err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(stderr.String(), "parse") {
		t.Fatalf("stderr:\n%s", stderr.String())
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "{" {
		t.Fatalf("invalid settings rewritten: %q", body)
	}
}

func TestProjectPrimeInstallSkipsAgentsSection(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	out := ProjectInstallOutput{}
	var stdout, stderr strings.Builder
	out.Stdout = &stdout
	out.Stderr = &stderr
	if err := InstallGrokProjectTo(out); err != nil {
		t.Fatal(err)
	}
	if err := InstallQoderProjectTo(out); err != nil {
		t.Fatal(err)
	}
	if err := InstallOMPProjectTo(out); err != nil {
		t.Fatal(err)
	}
	if err := InstallAntigravityProjectTo(out); err != nil {
		t.Fatal(err)
	}
	if err := InstallFactoryProjectTo(out); err != nil {
		t.Fatal(err)
	}
	if err := InstallDevinProjectTo(out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("project install created AGENTS.md: %v", err)
	}
	for _, rel := range []string{grokHooksFile, grokRulesFile, qoderSettingsFile, ompHookFile, ompRulesFile, antigravityHooksFile, antigravityRulesFile, factoryHooksFile, devinHooksFile} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
}

func TestGrokSkipsHookWhenClaudePrimeExists(t *testing.T) {
	stubDetectRenderOpts(t)
	env, stdout, _ := newFactoryTestEnv(t)
	root := filepath.Dir(env.agentsPath)
	claude := filepath.Join(root, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(claude), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claude, []byte("{\n  \"hooks\": {\n    \"SessionStart\": [\n      {\n        \"hooks\": [\n          {\"type\": \"command\", \"command\": \"bd prime --hook-json\"}\n        ]\n      }\n    ]\n  }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	grokHook := filepath.Join(root, grokHooksFile)
	if err := os.MkdirAll(filepath.Dir(grokHook), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(grokHook, []byte("{\n  \"hooks\": {\n    \"SessionStart\": [\n      {\n        \"hooks\": [\n          {\"type\": \"command\", \"command\": \"bd prime --hook-json\"}\n        ]\n      }\n    ]\n  }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := installGrok(env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(grokHook); !os.IsNotExist(err) {
		t.Fatalf("grok hook still present: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, grokRulesFile)); err != nil {
		t.Fatalf("rule missing: %v", err)
	}
	if !strings.Contains(stdout.String(), "Skipped .grok/hooks/beads.json") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	stdout.Reset()
	if err := checkGrok(env); err != nil {
		t.Fatalf("check: %v\n%s", err, stdout.String())
	}
}

func TestOMPSkipsHooksWhenAncestorOMPExists(t *testing.T) {
	stubDetectRenderOpts(t)
	parent := t.TempDir()
	if err := os.MkdirAll(filepath.Join(parent, ".omp"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parent, ".omp", "RULES.md"), []byte("# parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(parent, "pkg")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr strings.Builder
	env := agentsEnv{
		agentsPath: filepath.Join(child, "AGENTS.md"),
		stdout:     &stdout,
		stderr:     &stderr,
	}
	if err := installOMP(env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(child, ".omp")); !os.IsNotExist(err) {
		t.Fatalf("child .omp created and would shadow %s: %v", parent, err)
	}
	if !strings.Contains(stdout.String(), "Skipped .omp hooks") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	data, err := os.ReadFile(env.agentsPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "BEGIN BEADS INTEGRATION") {
		t.Fatalf("AGENTS.md section missing:\n%s", data)
	}
	stdout.Reset()
	if err := checkOMP(env); err != nil {
		t.Fatalf("check: %v\n%s", stdout.String(), err)
	}
}

func TestCheckSessionPrimeWithoutHooks(t *testing.T) {
	stubDetectRenderOpts(t)
	env, _, _ := newFactoryTestEnv(t)
	if err := installAgents(env, grokIntegration); err != nil {
		t.Fatal(err)
	}
	err := checkGrok(env)
	if !errors.Is(err, errSessionPrimeIncomplete) {
		t.Fatalf("check = %v", err)
	}
}

func seedUserHooks(t *testing.T, path string, wrapped bool) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	user := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": "echo keep",
			},
		},
	}
	events := map[string]interface{}{
		"PreToolUse": []interface{}{user},
	}
	root := events
	if wrapped {
		root = map[string]interface{}{
			"autoMemoryEnabled": true,
			"hooks":             events,
		}
	}
	if strings.HasSuffix(path, antigravityHooksFile) {
		root = map[string]interface{}{
			"safety-gate": map[string]interface{}{
				"PreToolUse": []interface{}{
					map[string]interface{}{"type": "command", "command": "echo keep"},
				},
			},
		}
	}
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertPrimeCommand(t *testing.T, path string, wrapped bool, command string, want bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("parse %s: %v\n%s", path, err, data)
	}
	if wrapped {
		if enabled, ok := config["autoMemoryEnabled"].(bool); !ok || !enabled {
			t.Fatalf("lost autoMemoryEnabled: %s", data)
		}
	}
	spec := sessionPrimeSpec{shape: primeShapeClaude, wrapped: wrapped, command: command}
	if strings.HasSuffix(path, antigravityHooksFile) {
		spec.shape = primeShapeAntigravity
		gate, ok := config["safety-gate"].(map[string]interface{})
		if !ok {
			t.Fatalf("lost safety-gate: %s", data)
		}
		if !antigravityFlatHas(gate, "echo keep") {
			t.Fatalf("lost user hook: %s", data)
		}
	} else if !claudePreToolKept(config, wrapped) {
		t.Fatalf("lost user hook: %s", data)
	}
	if got := primeHookCurrent(config, spec); got != want {
		t.Fatalf("command present = %v, want %v\n%s", got, want, data)
	}
}

func claudePreToolKept(config map[string]interface{}, wrapped bool) bool {
	hooks := config
	if wrapped {
		var ok bool
		hooks, ok = config["hooks"].(map[string]interface{})
		if !ok {
			return false
		}
	}
	return groupHasCommand(hooks, "PreToolUse", "echo keep")
}

func antigravityFlatHas(def map[string]interface{}, command string) bool {
	for _, entry := range toInterfaceSlice(def["PreToolUse"]) {
		entryMap, ok := entry.(map[string]interface{})
		if ok && entryMap["command"] == command {
			return true
		}
	}
	return false
}

func TestDevinSkipsHookWhenClaudePrimeExists(t *testing.T) {
	stubDetectRenderOpts(t)
	env, stdout, _ := newFactoryTestEnv(t)
	root := filepath.Dir(env.agentsPath)
	claude := filepath.Join(root, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(claude), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claude, []byte("{\n  \"hooks\": {\n    \"SessionStart\": [\n      {\n        \"hooks\": [\n          {\"type\": \"command\", \"command\": \"bd prime --hook-json\"}\n        ]\n      }\n    ]\n  }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	devinHook := filepath.Join(root, devinHooksFile)
	if err := os.MkdirAll(filepath.Dir(devinHook), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(devinHook, []byte("{\n  \"SessionStart\": [\n    {\n      \"hooks\": [\n        {\"type\": \"command\", \"command\": \"bd prime --hook-json\"}\n      ]\n    }\n  ]\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := checkDevinPrime(env); err == nil {
		t.Fatalf("check accepted a duplicate SessionStart hook:\n%s", stdout.String())
	}
	if err := installDevin(env); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(devinHook); !os.IsNotExist(err) {
		t.Fatalf("devin hook still present: %v", err)
	}
	if !strings.Contains(stdout.String(), "Skipped .devin/hooks.v1.json") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	stdout.Reset()
	if err := checkDevin(env); err != nil {
		t.Fatalf("check: %v\n%s", err, stdout.String())
	}
	if !strings.Contains(stdout.String(), "Devin uses the Claude SessionStart hook") {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
}
