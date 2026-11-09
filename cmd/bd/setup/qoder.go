package setup

import "github.com/steveyegge/beads/internal/templates/agents"

// qoderMemoryURL 是 Qoder CLI 记忆文件的官方说明。
// Qoder CLI 默认把项目 AGENTS.md 当作静态记忆（context.fileName 的默认值就是 AGENTS.md），
// 也会读 ~/.qoder/AGENTS.md 和 .qoder/rules/。国际版与 Qoder CN 都加载项目级 AGENTS.md，
// 因此一份仓库文件两边都生效。用户若把 context.fileName 改成别的名字，这段就不会被读到。
const qoderMemoryURL = "https://docs.qoder.com/cli/memory"

var qoderIntegration = agentsIntegration{
	name:         "Qoder CLI",
	setupCommand: "bd setup qoder",
	readHint:     "Qoder CLI reads project AGENTS.md when a session starts. Restart Qoder CLI if it is already running.",
	docsURL:      qoderMemoryURL,
	profile:      agents.ProfileFull,
}

var qoderEnvProvider = defaultAgentsEnv

func InstallQoder() error {
	return installQoder(qoderEnvProvider())
}

func installQoder(env agentsEnv) error {
	return installAgents(env, qoderIntegration)
}

func CheckQoder() error {
	return checkQoder(qoderEnvProvider())
}

func checkQoder(env agentsEnv) error {
	return checkAgents(env, qoderIntegration)
}

func RemoveQoder() error {
	return removeQoder(qoderEnvProvider())
}

func removeQoder(env agentsEnv) error {
	return removeAgents(env, qoderIntegration)
}
