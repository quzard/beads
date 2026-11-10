package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/steveyegge/beads/internal/templates/agents"
)

// grokProjectRulesURL 是 Grok 项目规则的官方说明。
// Grok 每次会话都会加载仓库里的 AGENTS.md，以及 .grok/rules/*.md。
// https://docs.x.ai/build/features/project-rules
const grokProjectRulesURL = "https://docs.x.ai/build/features/project-rules"

// grokHooksURL 是 Grok 钩子的官方说明。项目钩子在 .grok/hooks/*.json，
// 也认 Claude / Cursor 的钩子文件。SessionStart 存在，但被动事件的 stdout 会被丢掉，
// 所以规则文件才是真正进上下文的那一份。项目钩子要先信任（/hooks-trust 或 --trust）。
// https://docs.x.ai/build/features/hooks
const grokHooksURL = "https://docs.x.ai/build/features/hooks"

// grokRulesBody 是薄的 always-loaded 规则。Grok 不解析 frontmatter，整篇 markdown 都会进上下文。
// 标记给 --remove 用，避免删掉用户自己的同名文件。
// 认领、关闭事项的做法只来自 bd prime：这里再写一遍会落后于 PRIME.md 覆盖的工作流。
const grokRulesBody = `<!-- beads-setup-managed -->

# Beads

This repository contains ` + "`.beads`" + `. Before the first code, config, or doc edit, run ` + "`bd prime`" + ` and follow its output. Grok does not inject SessionStart stdout.

Keep this file and ` + "`AGENTS.md`" + ` out of ` + "`.gitignore`" + `. Grok skips ignored project rules.
`

const grokPrimeNote = `Grok ignores stdout from SessionStart, so .grok/hooks/beads.json does not inject context. .grok/rules/beads.md tells the session to run bd prime. Trust project hooks with /hooks-trust or grok --trust before they run.`

const grokUsesClaudeHookNote = `Skipped .grok/hooks/beads.json. .claude/settings.json already runs bd prime, and Grok loads Claude hooks, so a second SessionStart would run it twice. .grok/rules/beads.md still loads. Grok does not inject SessionStart stdout.`

var grokIntegration = agentsIntegration{
	name:             "Grok",
	setupCommand:     "bd setup grok",
	readHint:         "Grok loads AGENTS.md and .grok/rules/ for every session in this directory tree. Restart Grok if it is already running.",
	docsURL:          grokProjectRulesURL,
	profile:          agents.ProfileFull,
	skipNoConfigLine: true,
}

var grokEnvProvider = defaultAgentsEnv

func grokPrimeSpec() sessionPrimeSpec {
	return sessionPrimeSpec{
		name:         "Grok",
		setupCommand: grokIntegration.setupCommand,
		shape:        primeShapeClaude,
		wrapped:      true,
		hooksRel:     grokHooksFile,
		command:      SessionPrimeCommand,
		ruleRel:      grokRulesFile,
		ruleBody:     grokRulesBody,
		docsURL:      grokHooksURL,
		note:         grokPrimeNote,
	}
}

func InstallGrok() error {
	return installGrok(grokEnvProvider())
}

func installGrok(env agentsEnv) error {
	if err := installAgents(env, grokIntegration); err != nil {
		return err
	}
	return installGrokPrime(env)
}

// InstallGrokProjectTo 只装钩子和规则。bd init 已经按用户选的剖面写过 AGENTS.md，
// 这里再走完整配方会把 minimal 段升级成 full。
func InstallGrokProjectTo(out ProjectInstallOutput) error {
	env := grokEnvProvider()
	env.stdout, env.stderr = out.stdout(), out.stderr()
	return installGrokPrime(env)
}

// installGrokPrime 在项目 Claude 钩子已经会跑 bd prime 时不再写 Grok 的 SessionStart。
// Grok 会读 .claude/settings.json，两份钩子会各跑一次。
func installGrokPrime(env agentsEnv) error {
	root := projectRoot(env)
	if claudePrimeHookInstalled(root) {
		path := primeAbs(root, grokHooksFile)
		if _, err := os.Stat(path); err == nil {
			if err := removePrimeHooks(env, root, grokPrimeSpec()); err != nil {
				return err
			}
		}
		spec := grokPrimeSpec()
		spec.shape = ""
		spec.hooksRel = ""
		spec.note = grokUsesClaudeHookNote
		return installSessionPrime(env, spec)
	}
	return installSessionPrime(env, grokPrimeSpec())
}

func CheckGrok() error {
	return checkGrok(grokEnvProvider())
}

func checkGrok(env agentsEnv) error {
	sectionErr := checkAgents(env, grokIntegration)
	primeErr := checkGrokPrime(env)
	if sectionErr != nil {
		return sectionErr
	}
	return primeErr
}

func checkGrokPrime(env agentsEnv) error {
	root := projectRoot(env)
	spec := grokPrimeSpec()
	if claudePrimeHookInstalled(root) {
		config, exists, err := loadHooksFile(primeAbs(root, grokHooksFile))
		if err != nil {
			_, _ = fmt.Fprintf(env.stderr, "Error: Grok hooks: %v\n", err)
			return err
		}
		if exists && primeHookCurrent(config, spec) {
			_, _ = fmt.Fprintf(env.stdout, "✗ Grok SessionStart duplicates the Claude hook in .claude/settings.json\n")
			_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", spec.setupCommand)
			spec.shape = ""
			spec.hooksRel = ""
			if ruleErr := checkSessionPrime(env, spec); ruleErr != nil {
				return ruleErr
			}
			return errSessionPrimeIncomplete
		}
		_, _ = fmt.Fprintln(env.stdout, "✓ Grok uses the Claude SessionStart hook in .claude/settings.json")
		spec.shape = ""
		spec.hooksRel = ""
		return checkSessionPrime(env, spec)
	}
	return checkSessionPrime(env, spec)
}

// claudePrimeHookInstalled 看项目 .claude/settings.json 里有没有 bd prime 的 SessionStart。
// Grok 会加载这份 Claude 钩子，再写一份 Grok 钩子就会跑两遍。
func claudePrimeHookInstalled(root string) bool {
	config, exists, err := loadHooksFile(filepath.Join(root, ".claude", "settings.json"))
	if err != nil || !exists {
		return false
	}
	hooks, err := readHooks(config, true)
	if err != nil || hooks == nil {
		return false
	}
	for _, command := range []string{
		SessionPrimeCommand,
		"bd prime --stealth --hook-json",
		"bd prime",
		"bd prime --stealth",
	} {
		if groupHasCommand(hooks, "SessionStart", command) {
			return true
		}
	}
	return false
}

func RemoveGrok() error {
	return removeGrok(grokEnvProvider())
}

func removeGrok(env agentsEnv) error {
	primeErr := removeSessionPrime(env, grokPrimeSpec())
	sectionErr := removeAgents(env, grokIntegration)
	if primeErr != nil {
		return primeErr
	}
	return sectionErr
}
