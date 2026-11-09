package setup

import "github.com/steveyegge/beads/internal/templates/agents"

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
const grokRulesBody = `<!-- beads-setup-managed -->

# Beads

This repository tracks work with beads (` + "`bd`" + `). Grok loads every Markdown file in ` + "`.grok/rules/`" + `.

- Run ` + "`bd prime`" + ` at the start of a session and follow that workflow context.
- ` + "`.grok/hooks/beads.json`" + ` runs ` + "`bd prime --hook-json`" + ` on SessionStart. Grok does not inject stdout from that event, so the hook output is not context.
- Trust project hooks once with ` + "`/hooks-trust`" + ` or ` + "`grok --trust`" + ` before they run.

The beads section in ` + "`AGENTS.md`" + ` is the standing command reference.
`

const grokPrimeNote = `Grok ignores stdout from SessionStart, so .grok/hooks/beads.json does not inject context. .grok/rules/beads.md tells the session to run bd prime. Trust project hooks with /hooks-trust or grok --trust before they run.`

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
	return installSessionPrime(env, grokPrimeSpec())
}

// InstallGrokProjectTo 只装钩子和规则。bd init 已经按用户选的剖面写过 AGENTS.md，
// 这里再走完整配方会把 minimal 段升级成 full。
func InstallGrokProjectTo(out ProjectInstallOutput) error {
	env := grokEnvProvider()
	env.stdout, env.stderr = out.stdout(), out.stderr()
	return installSessionPrime(env, grokPrimeSpec())
}

func CheckGrok() error {
	return checkGrok(grokEnvProvider())
}

func checkGrok(env agentsEnv) error {
	sectionErr := checkAgents(env, grokIntegration)
	primeErr := checkSessionPrime(env, grokPrimeSpec())
	if sectionErr != nil {
		return sectionErr
	}
	return primeErr
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
