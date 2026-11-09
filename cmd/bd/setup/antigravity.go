package setup

import "github.com/steveyegge/beads/internal/templates/agents"

// antigravityRulesURL 是 Antigravity 规则的官方说明。
// AGENTS.md 整篇当作 always_on。 .agents/rules/*.md 缺 frontmatter 会被静默丢掉，
// 所以规则文件必须带 trigger。
// https://antigravity.google/docs/rules
const antigravityRulesURL = "https://antigravity.google/docs/rules"

// antigravityHooksURL 是 Antigravity 钩子的官方说明。工作区钩子在 .agents/hooks.json，
// 顶层键是钩子名字而不是事件名。SessionStart 在第一次模型调用前触发，
// 用 injectSteps.systemMessage 注入，不认 Claude 的 additionalContext。
// https://antigravity.google/docs/hooks
const antigravityHooksURL = "https://antigravity.google/docs/hooks"

const antigravityRulesBody = `---
trigger: always_on
description: Beads workflow priming for this repository
---

<!-- beads-setup-managed -->

# Beads

This repository tracks work with beads (` + "`bd`" + `).

- Follow the beads section in ` + "`AGENTS.md`" + `.
- The SessionStart hook ` + "`beads-prime`" + ` in ` + "`.agents/hooks.json`" + ` runs ` + "`bd prime --hook-antigravity`" + ` and injects that workflow context when a conversation starts.
- If that context is missing, run ` + "`bd prime`" + ` and follow it.
`

const antigravityPrimeNote = `Antigravity SessionStart injects bd prime as injectSteps, not Claude's additionalContext envelope. .agents/rules/beads.md is always_on; a rules file without frontmatter is ignored. bd must be on PATH.`

var antigravityIntegration = agentsIntegration{
	name:             "Antigravity",
	setupCommand:     "bd setup antigravity",
	readHint:         "Antigravity loads AGENTS.md and .agents/rules/beads.md, and runs the SessionStart hook in .agents/hooks.json. Restart Antigravity if it is already running.",
	docsURL:          antigravityRulesURL,
	profile:          agents.ProfileFull,
	skipNoConfigLine: true,
}

var antigravityEnvProvider = defaultAgentsEnv

func antigravityPrimeSpec() sessionPrimeSpec {
	return sessionPrimeSpec{
		name:         "Antigravity",
		setupCommand: antigravityIntegration.setupCommand,
		shape:        primeShapeAntigravity,
		hooksRel:     antigravityHooksFile,
		command:      AntigravityPrimeCommand,
		ruleRel:      antigravityRulesFile,
		ruleBody:     antigravityRulesBody,
		docsURL:      antigravityHooksURL,
		note:         antigravityPrimeNote,
	}
}

func InstallAntigravity() error {
	return installAntigravity(antigravityEnvProvider())
}

func installAntigravity(env agentsEnv) error {
	if err := installAgents(env, antigravityIntegration); err != nil {
		return err
	}
	return installSessionPrime(env, antigravityPrimeSpec())
}

// InstallAntigravityProjectTo 只装钩子和规则，不改写 init 已经写好的 AGENTS.md。
func InstallAntigravityProjectTo(out ProjectInstallOutput) error {
	env := antigravityEnvProvider()
	env.stdout, env.stderr = out.stdout(), out.stderr()
	return installSessionPrime(env, antigravityPrimeSpec())
}

func CheckAntigravity() error {
	return checkAntigravity(antigravityEnvProvider())
}

func checkAntigravity(env agentsEnv) error {
	sectionErr := checkAgents(env, antigravityIntegration)
	primeErr := checkSessionPrime(env, antigravityPrimeSpec())
	if sectionErr != nil {
		return sectionErr
	}
	return primeErr
}

func RemoveAntigravity() error {
	return removeAntigravity(antigravityEnvProvider())
}

func removeAntigravity(env agentsEnv) error {
	primeErr := removeSessionPrime(env, antigravityPrimeSpec())
	sectionErr := removeAgents(env, antigravityIntegration)
	if primeErr != nil {
		return primeErr
	}
	return sectionErr
}
