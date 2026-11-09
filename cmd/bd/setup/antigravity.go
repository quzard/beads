package setup

import "github.com/steveyegge/beads/internal/templates/agents"

// antigravityRulesURL 是 Google Antigravity 规则发现的官方说明。
// Antigravity（App、CLI agy、IDE）把工作区 AGENTS.md 整篇当作 always_on 规则，
// 并从工作目录向上读到仓库根；GEMINI.md 与 .agents/rules/*.md 也会被加载。
// AGENTS.md 不需要 frontmatter，和 factory/opencode 是同一条写入路径。
// .agents/rules/ 下的文件缺 frontmatter 会被静默丢弃，所以不往那里写。
const antigravityRulesURL = "https://antigravity.google/docs/rules"

var antigravityIntegration = agentsIntegration{
	name:         "Antigravity",
	setupCommand: "bd setup antigravity",
	readHint:     "Antigravity loads AGENTS.md from the workspace up to the repository root. Restart Antigravity if it is already running.",
	docsURL:      antigravityRulesURL,
	profile:      agents.ProfileFull,
}

var antigravityEnvProvider = defaultAgentsEnv

func InstallAntigravity() error {
	return installAntigravity(antigravityEnvProvider())
}

func installAntigravity(env agentsEnv) error {
	return installAgents(env, antigravityIntegration)
}

func CheckAntigravity() error {
	return checkAntigravity(antigravityEnvProvider())
}

func checkAntigravity(env agentsEnv) error {
	return checkAgents(env, antigravityIntegration)
}

func RemoveAntigravity() error {
	return removeAntigravity(antigravityEnvProvider())
}

func removeAntigravity(env agentsEnv) error {
	return removeAgents(env, antigravityIntegration)
}
