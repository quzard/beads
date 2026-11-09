package setup

import "github.com/steveyegge/beads/internal/templates/agents"

// qoderMemoryURL 是 Qoder CLI 记忆文件的官方说明。
// 项目 AGENTS.md 是默认静态记忆；.qoder/rules/**/*.md 是按主题拆开的规则。
// https://docs.qoder.com/cli/memory
const qoderMemoryURL = "https://docs.qoder.com/cli/memory"

// qoderHooksURL 是 Qoder CLI 钩子的官方说明。项目钩子在 .qoder/settings.json。
// SessionStart 会把 hookSpecificOutput.additionalContext（或纯文本 stdout）注入会话，
// source 含 startup/resume/clear/compact，和 Claude 一样可以在压缩后再注入。
// https://docs.qoder.com/cli/hooks
const qoderHooksURL = "https://docs.qoder.com/cli/hooks"

const qoderPrimeNote = `Qoder injects SessionStart stdout. .qoder/settings.json runs bd prime --hook-json, including after compaction (source=compact). bd must be on PATH.`

var qoderIntegration = agentsIntegration{
	name:             "Qoder CLI",
	setupCommand:     "bd setup qoder",
	readHint:         "Qoder CLI reads project AGENTS.md and runs the SessionStart hook in .qoder/settings.json. Restart Qoder CLI if it is already running.",
	docsURL:          qoderMemoryURL,
	profile:          agents.ProfileFull,
	skipNoConfigLine: true,
}

var qoderEnvProvider = defaultAgentsEnv

func qoderPrimeSpec() sessionPrimeSpec {
	return sessionPrimeSpec{
		name:         "Qoder CLI",
		setupCommand: qoderIntegration.setupCommand,
		shape:        primeShapeClaude,
		wrapped:      true,
		hooksRel:     qoderSettingsFile,
		command:      SessionPrimeCommand,
		docsURL:      qoderHooksURL,
		note:         qoderPrimeNote,
	}
}

func InstallQoder() error {
	return installQoder(qoderEnvProvider())
}

func installQoder(env agentsEnv) error {
	if err := installAgents(env, qoderIntegration); err != nil {
		return err
	}
	return installSessionPrime(env, qoderPrimeSpec())
}

// InstallQoderProjectTo 只装 SessionStart 钩子，不改写 init 已经写好的 AGENTS.md。
func InstallQoderProjectTo(out ProjectInstallOutput) error {
	env := qoderEnvProvider()
	env.stdout, env.stderr = out.stdout(), out.stderr()
	return installSessionPrime(env, qoderPrimeSpec())
}

func CheckQoder() error {
	return checkQoder(qoderEnvProvider())
}

func checkQoder(env agentsEnv) error {
	sectionErr := checkAgents(env, qoderIntegration)
	primeErr := checkSessionPrime(env, qoderPrimeSpec())
	if sectionErr != nil {
		return sectionErr
	}
	return primeErr
}

func RemoveQoder() error {
	return removeQoder(qoderEnvProvider())
}

func removeQoder(env agentsEnv) error {
	primeErr := removeSessionPrime(env, qoderPrimeSpec())
	sectionErr := removeAgents(env, qoderIntegration)
	if primeErr != nil {
		return primeErr
	}
	return sectionErr
}
