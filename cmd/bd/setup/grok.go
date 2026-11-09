package setup

import "github.com/steveyegge/beads/internal/templates/agents"

// grokProjectRulesURL 是 Grok CLI 项目规则的官方说明。
// Grok 在仓库里每次会话都会加载根目录 AGENTS.md，更深目录的同名文件在冲突时优先；
// 另外也会读 ~/.grok/ 以及 .grok/rules/。项目级 AGENTS.md 是文档推荐的主入口，
// 所以这里不另写 .grok/rules，避免和已有的 factory/opencode 段落分成两套。
// 标记与 factory、mux、opencode 相同，后装的配方只原位更新同一段。
const grokProjectRulesURL = "https://docs.x.ai/build/features/project-rules"

var grokIntegration = agentsIntegration{
	name:         "Grok",
	setupCommand: "bd setup grok",
	readHint:     "Grok loads AGENTS.md for every session in this directory tree. Restart Grok if it is already running.",
	docsURL:      grokProjectRulesURL,
	profile:      agents.ProfileFull,
}

var grokEnvProvider = defaultAgentsEnv

func InstallGrok() error {
	return installGrok(grokEnvProvider())
}

func installGrok(env agentsEnv) error {
	return installAgents(env, grokIntegration)
}

func CheckGrok() error {
	return checkGrok(grokEnvProvider())
}

func checkGrok(env agentsEnv) error {
	return checkAgents(env, grokIntegration)
}

func RemoveGrok() error {
	return removeGrok(grokEnvProvider())
}

func removeGrok(env agentsEnv) error {
	return removeAgents(env, grokIntegration)
}
