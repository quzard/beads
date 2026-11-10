package setup

import (
	"fmt"
	"os"

	"github.com/steveyegge/beads/internal/templates/agents"
)

// devinRulesURL 是 Devin CLI 规则文件的官方说明。Devin 每次会话都会加载仓库里的 AGENTS.md。
// https://docs.devin.ai/cli/extensibility/rules
const devinRulesURL = "https://docs.devin.ai/cli/extensibility/rules"

// devinHooksURL 是 Devin CLI 钩子的官方说明。项目钩子在 .devin/hooks.v1.json，
// 整个文件就是 hooks 对象，没有外层的 "hooks" 键。SessionStart 在启动和恢复时触发，
// 会把 hookSpecificOutput.additionalContext 注入会话。
// Devin 默认还会加载 .claude/settings.json 里的钩子（read_config_from.claude）。
// https://docs.devin.ai/cli/extensibility/hooks/overview
const devinHooksURL = "https://docs.devin.ai/cli/extensibility/hooks/overview"

const devinPrimeNote = `Devin injects SessionStart additionalContext. .devin/hooks.v1.json runs bd prime --hook-json on startup and resume. bd must be on PATH.`

const devinUsesClaudeHookNote = `Skipped .devin/hooks.v1.json. .claude/settings.json already runs bd prime, and Devin loads Claude hooks, so a second SessionStart would run it twice.`

var devinIntegration = agentsIntegration{
	name:             "Devin CLI",
	setupCommand:     "bd setup devin",
	readHint:         "Devin CLI reads project AGENTS.md and runs SessionStart hooks from .devin/hooks.v1.json and .claude/settings.json. Restart Devin CLI if it is already running.",
	docsURL:          devinRulesURL,
	profile:          agents.ProfileFull,
	skipNoConfigLine: true,
}

var devinEnvProvider = defaultAgentsEnv

func devinPrimeSpec() sessionPrimeSpec {
	return sessionPrimeSpec{
		name:         "Devin CLI",
		setupCommand: devinIntegration.setupCommand,
		shape:        primeShapeClaude,
		wrapped:      false,
		hooksRel:     devinHooksFile,
		command:      SessionPrimeCommand,
		docsURL:      devinHooksURL,
		note:         devinPrimeNote,
	}
}

func InstallDevin() error {
	return installDevin(devinEnvProvider())
}

func installDevin(env agentsEnv) error {
	if err := installAgents(env, devinIntegration); err != nil {
		return err
	}
	return installDevinPrime(env)
}

// InstallDevinProjectTo 只装 SessionStart 钩子，不改写 init 已经写好的 AGENTS.md。
func InstallDevinProjectTo(out ProjectInstallOutput) error {
	env := devinEnvProvider()
	env.stdout, env.stderr = out.stdout(), out.stderr()
	return installDevinPrime(env)
}

// installDevinPrime 在项目 Claude 钩子已经会跑 bd prime 时不再写 Devin 的 SessionStart。
// Devin 会读 .claude/settings.json，两份钩子会各跑一次。
func installDevinPrime(env agentsEnv) error {
	root := projectRoot(env)
	spec := devinPrimeSpec()
	if !claudePrimeHookInstalled(root) {
		return installSessionPrime(env, spec)
	}
	if _, err := os.Stat(primeAbs(root, devinHooksFile)); err == nil {
		if err := removePrimeHooks(env, root, spec); err != nil {
			return err
		}
	}
	spec.shape, spec.hooksRel, spec.note = "", "", devinUsesClaudeHookNote
	return installSessionPrime(env, spec)
}

func CheckDevin() error {
	return checkDevin(devinEnvProvider())
}

func checkDevin(env agentsEnv) error {
	sectionErr := checkAgents(env, devinIntegration)
	primeErr := checkDevinPrime(env)
	if sectionErr != nil {
		return sectionErr
	}
	return primeErr
}

func checkDevinPrime(env agentsEnv) error {
	root := projectRoot(env)
	spec := devinPrimeSpec()
	if !claudePrimeHookInstalled(root) {
		return checkSessionPrime(env, spec)
	}
	config, exists, err := loadHooksFile(primeAbs(root, devinHooksFile))
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: Devin hooks: %v\n", err)
		return err
	}
	if exists && primeHookCurrent(config, spec) {
		_, _ = fmt.Fprintf(env.stdout, "✗ Devin SessionStart duplicates the Claude hook in .claude/settings.json\n")
		_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", spec.setupCommand)
		return errSessionPrimeIncomplete
	}
	_, _ = fmt.Fprintln(env.stdout, "✓ Devin uses the Claude SessionStart hook in .claude/settings.json")
	return nil
}

func RemoveDevin() error {
	return removeDevin(devinEnvProvider())
}

func removeDevin(env agentsEnv) error {
	primeErr := removeSessionPrime(env, devinPrimeSpec())
	sectionErr := removeAgents(env, devinIntegration)
	if primeErr != nil {
		return primeErr
	}
	return sectionErr
}
