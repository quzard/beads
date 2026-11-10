package setup

import (
	"strings"

	"github.com/steveyegge/beads/internal/config"
	"github.com/steveyegge/beads/internal/templates"
	"github.com/steveyegge/beads/internal/templates/agents"
)

// primeOverrideFromConfig 报告生成的代理说明是否只写指向 bd prime 的一句：
// 有 PRIME.md 覆盖时，bd prime 输出覆盖文件，生成的说明若再写默认规则，就会和覆盖的工作流互相矛盾。
//
// userLevel 为 true 时只看用户配置目录里的 PRIME.md：用户级文件（~/.codex/AGENTS.md、
// ~/.agents/skills/beads 等）对所有仓库生效，判断不能随当前目录变化，安装和 --check 才一致。
// 项目级文件写在当前目录，按 bd prime 的完整顺序找：当前目录的 .beads、bd 启动时记录的
// beads 目录、用户配置目录。
func primeOverrideFromConfig(userLevel bool) bool {
	if userLevel {
		return config.GlobalPrimePath("") != ""
	}
	_, ok := config.ReadPrimeOverride("", config.WorkspaceBeadsDir())
	return ok
}

// primeOverrideActiveImpl 默认是 primeOverrideFromConfig。测试替换它，
// 使结果不依赖本机的配置目录（见 TestMain）。
var primeOverrideActiveImpl = primeOverrideFromConfig

func primeOverrideActive(userLevel bool) bool {
	return primeOverrideActiveImpl(userLevel)
}

// userLevelRenderOpts 是用户级文件的渲染选项：在 agents.UserLevelRenderOpts 的固定值上，
// 按用户配置目录里的 PRIME.md 决定是否只写指针。
func userLevelRenderOpts() agents.RenderOpts {
	opts := agents.UserLevelRenderOpts()
	opts.CustomPrime = primeOverrideActive(true)
	return opts
}

// agentSkillContent 返回要安装的 SKILL.md。有 PRIME.md 覆盖时，frontmatter（技能名和
// 触发描述）不变，正文只写指向 bd prime 的一句。
func agentSkillContent(userLevel bool) string {
	skill := templates.BeadsAgentSkill()
	if !primeOverrideActive(userLevel) {
		return skill
	}
	frontmatter, ok := skillFrontmatter(skill)
	if !ok {
		// 模板没有 frontmatter 时不拼一个无效的技能出来；TestAgentSkillTemplateHasFrontmatter 防止走到这里。
		return skill
	}
	return frontmatter + "\n# Beads\n\n" + agents.PrimePointerText() + "\n"
}

// skillFrontmatter returns the leading "---" block of a skill file, including
// both delimiter lines and the newline after the closing one.
func skillFrontmatter(skill string) (string, bool) {
	const delim = "---\n"
	if !strings.HasPrefix(skill, delim) {
		return "", false
	}
	end := strings.Index(skill[len(delim):], "\n"+delim)
	if end == -1 {
		return "", false
	}
	return skill[:len(delim)+end+len("\n"+delim)], true
}
