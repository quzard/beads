package setup

import "github.com/steveyegge/beads/internal/templates/agents"

// ompContextFilesURL 是 omp 上下文文件的官方说明。
// omp 推荐把项目约定写在仓库根目录 AGENTS.md，会话开始时自动加载。
// 同一目录若已有 .omp/AGENTS.md，原生文件优先于这份 AGENTS.md，beads 段可能被盖住；
// 不主动创建 .omp/，因为最近的非空 .omp/ 会截断 omp 继续向上查找原生配置。
const ompContextFilesURL = "https://omp.sh/docs/context-files"

var ompIntegration = agentsIntegration{
	name:         "OMP",
	setupCommand: "bd setup omp",
	readHint:     "omp loads AGENTS.md at session start. Start a new omp session to pick it up. A .omp/AGENTS.md in this directory takes priority over this file.",
	docsURL:      ompContextFilesURL,
	profile:      agents.ProfileFull,
}

var ompEnvProvider = defaultAgentsEnv

func InstallOMP() error {
	return installOMP(ompEnvProvider())
}

func installOMP(env agentsEnv) error {
	return installAgents(env, ompIntegration)
}

func CheckOMP() error {
	return checkOMP(ompEnvProvider())
}

func checkOMP(env agentsEnv) error {
	return checkAgents(env, ompIntegration)
}

func RemoveOMP() error {
	return removeOMP(ompEnvProvider())
}

func removeOMP(env agentsEnv) error {
	return removeAgents(env, ompIntegration)
}
