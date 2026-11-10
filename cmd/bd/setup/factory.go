package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/steveyegge/beads/internal/templates/agents"
)

// factoryHooksURL 是 Factory Droid 钩子的官方说明。项目钩子在 .factory/hooks.json，
// 顶层就是事件名（不包一层 hooks）。没有 hooks.json 时，Droid 改读 settings.json 里的
// hooks；旧路径 .factory/hooks/hooks.json 仍会被加载，下次保存会迁到 hooks.json。
// 所以不能在已有这两处钩子时凭空新建 hooks.json，否则原来的钩子不再执行。
// SessionStart 在启动、恢复、清空和压缩后触发，hookSpecificOutput.additionalContext
// 会追加进新会话。
// https://docs.factory.com/harness/hooks
const factoryHooksURL = "https://docs.factory.com/harness/hooks"

const (
	factorySettingsFile       = ".factory/settings.json"
	factoryLegacyHooksFile    = ".factory/hooks/hooks.json"
	factoryLegacyMigratedFile = ".factory/hooks/hooks.migrated.json"
)

const factoryPrimeNote = `Factory SessionStart injects bd prime via hookSpecificOutput.additionalContext on startup, resume, clear, and compact. bd must be on PATH.`

const factorySettingsPrimeNote = `Factory already declares hooks in .factory/settings.json, and Droid reads that hooks key only while .factory/hooks.json is absent. The SessionStart command was merged into settings.json so those hooks keep running. bd must be on PATH.`

var factoryIntegration = agentsIntegration{
	name:             "Factory.ai (Droid)",
	setupCommand:     "bd setup factory",
	readHint:         "Factory Droid reads AGENTS.md on session start. The SessionStart hook is merged into .factory/hooks.json, or into .factory/settings.json when that file already declares hooks.",
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
	return installFactoryHooks(env)
}

// InstallFactoryProjectTo 只装 SessionStart 钩子，不改写 init 已经写好的 AGENTS.md。
func InstallFactoryProjectTo(out ProjectInstallOutput) error {
	env := factoryEnvProvider()
	env.stdout, env.stderr = out.stdout(), out.stderr()
	return installFactoryHooks(env)
}

func CheckFactory() error {
	return checkFactory(factoryEnvProvider())
}

func checkFactory(env factoryEnv) error {
	sectionErr := checkAgents(env, factoryIntegration)
	primeErr := checkFactoryHooks(env)
	if sectionErr != nil {
		return sectionErr
	}
	return primeErr
}

func RemoveFactory() error {
	return removeFactory(factoryEnvProvider())
}

func removeFactory(env factoryEnv) error {
	primeErr := removeFactoryHooks(env)
	sectionErr := removeAgents(env, factoryIntegration)
	if primeErr != nil {
		return primeErr
	}
	return sectionErr
}

// installFactoryHooks 选择 Droid 真正会读的那份钩子文件再写入。
// hooks.json 已存在就合并进去；否则 settings.json 里已有 hooks 时合并进那一层，
// 避免新建 hooks.json 把 settings 里的钩子挡住。只剩旧路径时，先把内容抄进
// hooks.json，再把旧文件改名为 hooks.migrated.json，和 Droid 自己的迁移一样。
func installFactoryHooks(env factoryEnv) error {
	root := projectRoot(env)
	spec, migrated, err := factoryInstallSpec(root)
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: Factory hooks: %v\n", err)
		return err
	}
	if err := installSessionPrime(env, spec); err != nil {
		return err
	}
	if migrated {
		_, _ = fmt.Fprintf(env.stdout, "Migrated %s to %s so the new %s does not hide those hooks.\n", factoryLegacyHooksFile, factoryLegacyMigratedFile, factoryHooksFile)
	}
	return nil
}

func checkFactoryHooks(env factoryEnv) error {
	spec, err := factoryCheckSpec(projectRoot(env))
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: Factory hooks: %v\n", err)
		return err
	}
	return checkSessionPrime(env, spec)
}

func removeFactoryHooks(env factoryEnv) error {
	root := projectRoot(env)
	var first error
	found := false
	for _, spec := range []sessionPrimeSpec{factoryPrimeSpec(), factorySettingsSpec()} {
		path := primeAbs(root, spec.hooksRel)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		found = true
		if err := removePrimeHooks(env, root, spec); err != nil && first == nil {
			first = err
		}
	}
	if !found {
		_, _ = fmt.Fprintf(env.stdout, "No %s SessionStart hook found\n", factoryIntegration.name)
	}
	return first
}

func factorySettingsSpec() sessionPrimeSpec {
	spec := factoryPrimeSpec()
	spec.wrapped = true
	spec.hooksRel = factorySettingsFile
	spec.note = factorySettingsPrimeNote
	return spec
}

// factoryInstallSpec 决定这次要把 SessionStart 写进哪一份文件。migrated 表示刚刚
// 把旧的 hooks/hooks.json 归档了。
func factoryInstallSpec(root string) (sessionPrimeSpec, bool, error) {
	if _, err := os.Stat(filepath.Join(root, factoryHooksFile)); err == nil {
		return factoryPrimeSpec(), false, nil
	}
	declared, err := factorySettingsDeclaresHooks(filepath.Join(root, factorySettingsFile))
	if err != nil {
		return sessionPrimeSpec{}, false, err
	}
	if declared {
		return factorySettingsSpec(), false, nil
	}
	legacy := filepath.Join(root, factoryLegacyHooksFile)
	if _, err := os.Stat(legacy); err == nil {
		if err := migrateFactoryLegacyHooks(root); err != nil {
			return sessionPrimeSpec{}, false, err
		}
		return factoryPrimeSpec(), true, nil
	}
	return factoryPrimeSpec(), false, nil
}

// factoryCheckSpec 只看当前会生效的那份文件，不迁移、不创建。
func factoryCheckSpec(root string) (sessionPrimeSpec, error) {
	if _, err := os.Stat(filepath.Join(root, factoryHooksFile)); err == nil {
		return factoryPrimeSpec(), nil
	}
	declared, err := factorySettingsDeclaresHooks(filepath.Join(root, factorySettingsFile))
	if err != nil {
		return sessionPrimeSpec{}, err
	}
	if declared {
		return factorySettingsSpec(), nil
	}
	return factoryPrimeSpec(), nil
}

func factorySettingsDeclaresHooks(path string) (bool, error) {
	config, exists, err := loadHooksFile(path)
	if err != nil || !exists {
		return false, err
	}
	raw, ok := config["hooks"]
	if !ok || raw == nil {
		return false, nil
	}
	if _, ok := raw.(map[string]interface{}); !ok {
		return false, fmt.Errorf("%s hooks is not an object", factorySettingsFile)
	}
	return true, nil
}

func migrateFactoryLegacyHooks(root string) error {
	legacy := filepath.Join(root, factoryLegacyHooksFile)
	migrated := filepath.Join(root, factoryLegacyMigratedFile)
	if _, err := os.Stat(migrated); err == nil {
		return fmt.Errorf("%s already exists; refusing to archive %s over it", factoryLegacyMigratedFile, factoryLegacyHooksFile)
	}
	config, exists, err := loadHooksFile(legacy)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	if err := saveHooksFile(filepath.Join(root, factoryHooksFile), config); err != nil {
		return err
	}
	return os.Rename(legacy, migrated)
}

// SessionPrimeCommitPaths 是 bd init 可以放进提交的钩子和规则文件。
// 只列 bd 写的这些路径，不把 .qoder、.grok、.omp、.factory 整目录加进去，
// 避免把 settings.local.json 或其他本地配置一起提交。
func SessionPrimeCommitPaths() []string {
	return []string{
		grokHooksFile,
		grokRulesFile,
		qoderSettingsFile,
		ompHookFile,
		ompRulesFile,
		antigravityHooksFile,
		antigravityRulesFile,
		factoryHooksFile,
		factorySettingsFile,
		factoryLegacyMigratedFile,
		devinHooksFile,
	}
}
