package setup

import "github.com/steveyegge/beads/internal/templates/agents"

// ompContextFilesURL 是 omp 上下文文件的官方说明。
// 仓库根目录 AGENTS.md 会在会话开始时加载。最近的非空 .omp/ 会截断继续向上查找
// 原生 .omp/AGENTS.md 和 .omp/RULES.md，所以这里不写 .omp/AGENTS.md，避免盖住根上的 AGENTS.md。
// https://omp.sh/docs/context-files
const ompContextFilesURL = "https://omp.sh/docs/context-files"

// ompHooksURL 是 omp 钩子的官方说明。钩子是 .omp/hooks/pre/ 和 post/ 下的 JS/TS 模块。
// session_start 的返回值会被丢掉；能把文字送进上下文的是 before_agent_start。
// https://omp.sh/docs/hooks
const ompHooksURL = "https://omp.sh/docs/hooks"

// ompRulesBody 是 alwaysApply 规则。带 condition 的规则会变成 TTSR，只在匹配时触发，
// 不适合当每次会话都在的提示，所以这里只用 alwaysApply。
// https://omp.sh/docs/ttsr
const ompRulesBody = `---
description: Beads workflow for this repository
alwaysApply: true
---

<!-- beads-setup-managed -->

# Beads

This repository tracks work with beads (` + "`bd`" + `).

- Follow the beads section in ` + "`AGENTS.md`" + `.
- ` + "`.omp/hooks/pre/beads.js`" + ` injects ` + "`bd prime`" + ` on the first turn of a session. omp's ` + "`session_start`" + ` event cannot return context, so the hook injects on ` + "`before_agent_start`" + `, and again after a session switch or compaction.
- If that context is missing, run ` + "`bd prime`" + ` and follow it.
`

// ompHookBody 是 session_start 的等价物：事件本身不能返回上下文，所以在第一次
// before_agent_start 把 bd prime 的输出塞进会话。压缩或切换会话后再注入一次。
const ompHookBody = `// beads-setup-managed
// session_start cannot return context. Inject bd prime on the first
// before_agent_start, and again after session_switch or session_compact.
import { execFileSync } from "node:child_process";

let primed = false;

function resetPrime() {
  primed = false;
}

export default function beadsPrime(pi) {
  pi.on("session_start", resetPrime);
  pi.on("session_switch", resetPrime);
  pi.on("session_compact", resetPrime);
  pi.on("before_agent_start", () => {
    if (primed) {
      return;
    }
    let content = "";
    try {
      content = execFileSync("bd", ["prime"], {
        encoding: "utf8",
        timeout: 15000,
        stdio: ["ignore", "pipe", "ignore"],
      });
    } catch {
      return;
    }
    if (!content || !content.trim()) {
      return;
    }
    primed = true;
    return {
      message: {
        customType: "beads-prime",
        content,
        display: false,
      },
    };
  });
}
`

const ompPrimeNote = `omp session_start cannot return context. .omp/hooks/pre/beads.js injects bd prime on the first turn (before_agent_start), and again after a session switch or compaction. A non-empty .omp/ is the nearest native directory, so native AGENTS.md and RULES.md are read only from here. bd must be on PATH.`

var ompIntegration = agentsIntegration{
	name:             "OMP",
	setupCommand:     "bd setup omp",
	readHint:         "omp loads AGENTS.md at session start. Start a new omp session to pick up the hook. A non-empty .omp/ here is the nearest native directory.",
	docsURL:          ompContextFilesURL,
	profile:          agents.ProfileFull,
	skipNoConfigLine: true,
}

var ompEnvProvider = defaultAgentsEnv

func ompPrimeSpec() sessionPrimeSpec {
	return sessionPrimeSpec{
		name:         "OMP",
		setupCommand: ompIntegration.setupCommand,
		ruleRel:      ompRulesFile,
		ruleBody:     ompRulesBody,
		scriptRel:    ompHookFile,
		scriptBody:   ompHookBody,
		docsURL:      ompHooksURL,
		note:         ompPrimeNote,
	}
}

func InstallOMP() error {
	return installOMP(ompEnvProvider())
}

func installOMP(env agentsEnv) error {
	if err := installAgents(env, ompIntegration); err != nil {
		return err
	}
	return installSessionPrime(env, ompPrimeSpec())
}

// InstallOMPProjectTo 只装钩子和规则，不改写 init 已经写好的 AGENTS.md。
func InstallOMPProjectTo(out ProjectInstallOutput) error {
	env := ompEnvProvider()
	env.stdout, env.stderr = out.stdout(), out.stderr()
	return installSessionPrime(env, ompPrimeSpec())
}

func CheckOMP() error {
	return checkOMP(ompEnvProvider())
}

func checkOMP(env agentsEnv) error {
	sectionErr := checkAgents(env, ompIntegration)
	primeErr := checkSessionPrime(env, ompPrimeSpec())
	if sectionErr != nil {
		return sectionErr
	}
	return primeErr
}

func RemoveOMP() error {
	return removeOMP(ompEnvProvider())
}

func removeOMP(env agentsEnv) error {
	primeErr := removeSessionPrime(env, ompPrimeSpec())
	sectionErr := removeAgents(env, ompIntegration)
	if primeErr != nil {
		return primeErr
	}
	return sectionErr
}
