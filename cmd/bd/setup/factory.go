package setup

import "github.com/steveyegge/beads/internal/templates/agents"

// factoryHooksURL 是 Factory Droid 钩子的官方说明。项目钩子在 .factory/hooks.json，
// 顶层就是事件名（不包一层 hooks）。SessionStart 在启动、恢复、清空和压缩后触发，
// hookSpecificOutput.additionalContext 会追加进新会话。
// https://docs.factory.com/harness/hooks
const factoryHooksURL = "https://docs.factory.com/harness/hooks"

const factoryPrimeNote = `Factory SessionStart injects bd prime via hookSpecificOutput.additionalContext on startup, resume, clear, and compact. bd must be on PATH.`

var factoryIntegration = agentsIntegration{
	name:             "Factory.ai (Droid)",
	setupCommand:     "bd setup factory",
	readHint:         "Factory Droid reads AGENTS.md on session start and runs the SessionStart hook in .factory/hooks.json.",
	profile:          agents.ProfileFull,
	skipNoConfigLine: true,
}

type factoryEnv = agentsEnv

var factoryEnvProvider = defaultAgentsEnv

func factoryPrimeSpec() sessionPrimeSpec {
	return sessionPrimeSpec{
		name:         "Factory.ai (Droid)",
		setupCommand: factoryIntegration.setupCommand,
		shape:        primeShapeClaude,
		wrapped:      false,
		hooksRel:     factoryHooksFile,
		command:      SessionPrimeCommand,
		docsURL:      factoryHooksURL,
		note:         factoryPrimeNote,
	}
}

func InstallFactory() error {
	return installFactory(factoryEnvProvider())
}

func installFactory(env factoryEnv) error {
	if err := installAgents(env, factoryIntegration); err != nil {
		return err
	}
	return installSessionPrime(env, factoryPrimeSpec())
}

// InstallFactoryProjectTo 只装 SessionStart 钩子，不改写 init 已经写好的 AGENTS.md。
func InstallFactoryProjectTo(out ProjectInstallOutput) error {
	env := factoryEnvProvider()
	env.stdout, env.stderr = out.stdout(), out.stderr()
	return installSessionPrime(env, factoryPrimeSpec())
}

func CheckFactory() error {
	return checkFactory(factoryEnvProvider())
}

func checkFactory(env factoryEnv) error {
	sectionErr := checkAgents(env, factoryIntegration)
	primeErr := checkSessionPrime(env, factoryPrimeSpec())
	if sectionErr != nil {
		return sectionErr
	}
	return primeErr
}

func RemoveFactory() error {
	return removeFactory(factoryEnvProvider())
}

func removeFactory(env factoryEnv) error {
	primeErr := removeSessionPrime(env, factoryPrimeSpec())
	sectionErr := removeAgents(env, factoryIntegration)
	if primeErr != nil {
		return primeErr
	}
	return sectionErr
}
