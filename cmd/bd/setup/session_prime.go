package setup

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// 这些常量和下面的 sessionPrimeSpec 是各家「会话开始时注入 bd prime」的共用安装器。
// Claude 形态（Qoder、Grok、Factory）写 SessionStart 命令；Antigravity 的 hooks.json
// 是按钩子名字分组、SessionStart 直接挂命令；OMP 没有这种 JSON，改写 JS 钩子。
// 路径相对项目根（AGENTS.md 所在目录，bd setup 默认就是当前目录）。
const (
	// SessionPrimeCommand 把 bd prime 包进 Claude / Gemini / Codex / Qoder / Factory
	// 都认的 hookSpecificOutput.additionalContext。
	SessionPrimeCommand = "bd prime --hook-json"
	// AntigravityPrimeCommand 把 bd prime 包进 Antigravity SessionStart 要的 injectSteps。
	AntigravityPrimeCommand = "bd prime --hook-antigravity"

	// beadsSetupManagedMarker 出现在 bd 写的规则和脚本里。--remove 只删带这个标记的文件，
	// 避免误删用户自己放在同一路径上的文件。
	beadsSetupManagedMarker = "beads-setup-managed"

	primeShapeClaude      = "claude"
	primeShapeAntigravity = "antigravity"

	grokHooksFile        = ".grok/hooks/beads.json"
	grokRulesFile        = ".grok/rules/beads.md"
	qoderSettingsFile    = ".qoder/settings.json"
	ompHookFile          = ".omp/hooks/pre/beads.js"
	ompRulesFile         = ".omp/rules/beads.md"
	antigravityHooksFile = ".agents/hooks.json"
	antigravityRulesFile = ".agents/rules/beads.md"
	antigravityHookName  = "beads-prime"
	factoryHooksFile     = ".factory/hooks.json"
)

// errSessionPrimeIncomplete 表示 --check 发现钩子或规则缺了、旧了，或被用户文件占了路径。
var errSessionPrimeIncomplete = errors.New("session prime integration incomplete")

// sessionPrimeSpec 描述一个配方除 AGENTS.md 段以外还要写的会话启动文件。
// 空的 shape / ruleRel / scriptRel 表示该配方没有对应的那一份。
type sessionPrimeSpec struct {
	name         string
	setupCommand string
	shape        string
	wrapped      bool
	hooksRel     string
	command      string
	ruleRel      string
	ruleBody     string
	scriptRel    string
	scriptBody   string
	docsURL      string
	note         string
}

func projectRoot(env agentsEnv) string {
	dir := filepath.Dir(env.agentsPath)
	if dir == "" {
		return "."
	}
	return dir
}

func primeAbs(root, rel string) string {
	return filepath.Join(root, rel)
}

func installSessionPrime(env agentsEnv, spec sessionPrimeSpec) error {
	root := projectRoot(env)
	if spec.shape != "" {
		status, err := installPrimeHooks(root, spec)
		if err != nil {
			_, _ = fmt.Fprintf(env.stderr, "Error: %s hooks: %v\n", spec.name, err)
			return err
		}
		path := primeAbs(root, spec.hooksRel)
		switch status {
		case "current":
			_, _ = fmt.Fprintf(env.stdout, "✓ %s SessionStart hook already current: %s\n", spec.name, path)
		default:
			_, _ = fmt.Fprintf(env.stdout, "✓ %s SessionStart hook installed: %s\n", spec.name, path)
		}
	}
	if spec.ruleRel != "" {
		if err := installManagedText(env, spec.name, "rule", primeAbs(root, spec.ruleRel), spec.ruleBody); err != nil {
			return err
		}
	}
	if spec.scriptRel != "" {
		if err := installManagedText(env, spec.name, "hook", primeAbs(root, spec.scriptRel), spec.scriptBody); err != nil {
			return err
		}
	}
	if spec.note != "" {
		_, _ = fmt.Fprintf(env.stdout, "\n%s\n", spec.note)
	}
	if spec.docsURL != "" {
		_, _ = fmt.Fprintf(env.stdout, "Hooks guide: %s\n", spec.docsURL)
	}
	return nil
}

func checkSessionPrime(env agentsEnv, spec sessionPrimeSpec) error {
	root := projectRoot(env)
	var incomplete error
	note := func(err error) {
		if incomplete == nil && err != nil {
			incomplete = err
		}
	}
	if spec.shape != "" {
		note(checkPrimeHooks(env, root, spec))
	}
	if spec.ruleRel != "" {
		note(checkManagedText(env, spec, "rule", primeAbs(root, spec.ruleRel), spec.ruleBody))
	}
	if spec.scriptRel != "" {
		note(checkManagedText(env, spec, "hook", primeAbs(root, spec.scriptRel), spec.scriptBody))
	}
	return incomplete
}

func removeSessionPrime(env agentsEnv, spec sessionPrimeSpec) error {
	root := projectRoot(env)
	var first error
	note := func(err error) {
		if first == nil && err != nil {
			first = err
		}
	}
	if spec.shape != "" {
		note(removePrimeHooks(env, root, spec))
	}
	if spec.scriptRel != "" {
		note(removeManagedText(env, spec.name, "hook", primeAbs(root, spec.scriptRel), root))
	}
	if spec.ruleRel != "" {
		note(removeManagedText(env, spec.name, "rule", primeAbs(root, spec.ruleRel), root))
	}
	return first
}

func installManagedText(env agentsEnv, name, kind, path, body string) error {
	wrote, err := writeManagedText(path, body)
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: write %s %s: %v\n", name, kind, err)
		return err
	}
	if wrote {
		_, _ = fmt.Fprintf(env.stdout, "✓ %s %s installed: %s\n", name, kind, path)
	} else {
		_, _ = fmt.Fprintf(env.stdout, "✓ %s %s already current: %s\n", name, kind, path)
	}
	return nil
}

func checkManagedText(env agentsEnv, spec sessionPrimeSpec, kind, path, body string) error {
	data, err := os.ReadFile(path) // #nosec G304 -- bd 自己的配方路径
	if os.IsNotExist(err) {
		_, _ = fmt.Fprintf(env.stdout, "✗ %s %s not installed: %s\n", spec.name, kind, path)
		_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", spec.setupCommand)
		return errSessionPrimeIncomplete
	}
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: read %s: %v\n", path, err)
		return err
	}
	if string(data) == body {
		_, _ = fmt.Fprintf(env.stdout, "✓ %s %s installed: %s\n", spec.name, kind, path)
		return nil
	}
	_, _ = fmt.Fprintf(env.stdout, "⚠ %s %s is stale: %s\n", spec.name, kind, path)
	_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", spec.setupCommand)
	return errSessionPrimeIncomplete
}

func removeManagedText(env agentsEnv, name, kind, path, root string) error {
	data, err := os.ReadFile(path) // #nosec G304 -- bd 自己的配方路径
	if os.IsNotExist(err) {
		_, _ = fmt.Fprintf(env.stdout, "No %s %s found\n", name, kind)
		return nil
	}
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: read %s: %v\n", path, err)
		return err
	}
	if !bytes.Contains(data, []byte(beadsSetupManagedMarker)) {
		_, _ = fmt.Fprintf(env.stdout, "Left %s (not beads-managed)\n", path)
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		_, _ = fmt.Fprintf(env.stderr, "Error: remove %s: %v\n", path, err)
		return err
	}
	pruneEmptyParents(path, root)
	_, _ = fmt.Fprintf(env.stdout, "✓ Removed %s %s: %s\n", name, kind, path)
	return nil
}

func writeManagedText(path, body string) (bool, error) {
	data := []byte(body)
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) { // #nosec G304 -- bd 自己的配方路径
		return false, nil
	} else if err != nil && !os.IsNotExist(err) {
		return false, err
	}
	if err := EnsureDir(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := atomicWriteFile(path, data); err != nil {
		return false, err
	}
	return true, nil
}

func installPrimeHooks(root string, spec sessionPrimeSpec) (string, error) {
	path := primeAbs(root, spec.hooksRel)
	config, _, err := loadHooksFile(path)
	if err != nil {
		return "", err
	}
	changed, err := ensurePrimeHook(config, spec)
	if err != nil {
		return "", err
	}
	if !changed {
		return "current", nil
	}
	if err := saveHooksFile(path, config); err != nil {
		return "", err
	}
	return "installed", nil
}

func checkPrimeHooks(env agentsEnv, root string, spec sessionPrimeSpec) error {
	path := primeAbs(root, spec.hooksRel)
	config, exists, err := loadHooksFile(path)
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: read %s: %v\n", path, err)
		return err
	}
	if !exists || !primeHookCurrent(config, spec) {
		_, _ = fmt.Fprintf(env.stdout, "✗ %s SessionStart hook not installed: %s\n", spec.name, path)
		_, _ = fmt.Fprintf(env.stdout, "  Run: %s\n", spec.setupCommand)
		return errSessionPrimeIncomplete
	}
	_, _ = fmt.Fprintf(env.stdout, "✓ %s SessionStart hook installed: %s\n", spec.name, path)
	return nil
}

func removePrimeHooks(env agentsEnv, root string, spec sessionPrimeSpec) error {
	path := primeAbs(root, spec.hooksRel)
	config, exists, err := loadHooksFile(path)
	if err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: read %s: %v\n", path, err)
		return err
	}
	if !exists {
		_, _ = fmt.Fprintf(env.stdout, "No %s SessionStart hook found\n", spec.name)
		return nil
	}
	removed, empty := stripPrimeHook(config, spec)
	if !removed {
		_, _ = fmt.Fprintf(env.stdout, "No %s SessionStart hook found\n", spec.name)
		return nil
	}
	if empty {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			_, _ = fmt.Fprintf(env.stderr, "Error: remove %s: %v\n", path, err)
			return err
		}
		pruneEmptyParents(path, root)
	} else if err := saveHooksFile(path, config); err != nil {
		_, _ = fmt.Fprintf(env.stderr, "Error: write %s: %v\n", path, err)
		return err
	}
	_, _ = fmt.Fprintf(env.stdout, "✓ Removed %s SessionStart hook: %s\n", spec.name, path)
	return nil
}

func loadHooksFile(path string) (map[string]interface{}, bool, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- bd 自己的配方路径
	if os.IsNotExist(err) {
		return map[string]interface{}{}, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	config, err := parseHooksJSON(data)
	if err != nil {
		return nil, true, fmt.Errorf("parse %s: %w", path, err)
	}
	return config, true, nil
}

func saveHooksFile(path string, config map[string]interface{}) error {
	if err := EnsureDir(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := marshalHooksJSON(config)
	if err != nil {
		return err
	}
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, data) { // #nosec G304 -- bd 自己的配方路径
		return nil
	}
	return atomicWriteFile(path, data)
}

func ensurePrimeHook(config map[string]interface{}, spec sessionPrimeSpec) (bool, error) {
	switch spec.shape {
	case primeShapeClaude:
		hooks, err := mutableHooks(config, spec.wrapped)
		if err != nil {
			return false, err
		}
		return ensureClaudeStyle(hooks, spec.command)
	case primeShapeAntigravity:
		return ensureAntigravityStyle(config, spec.command)
	default:
		return false, fmt.Errorf("unknown hook shape %q", spec.shape)
	}
}

func primeHookCurrent(config map[string]interface{}, spec sessionPrimeSpec) bool {
	switch spec.shape {
	case primeShapeClaude:
		hooks, err := readHooks(config, spec.wrapped)
		return err == nil && hooks != nil && groupHasCommand(hooks, "SessionStart", spec.command)
	case primeShapeAntigravity:
		return antigravityHasCommand(config, spec.command)
	default:
		return false
	}
}

func stripPrimeHook(config map[string]interface{}, spec sessionPrimeSpec) (removed, empty bool) {
	switch spec.shape {
	case primeShapeClaude:
		hooks, err := readHooks(config, spec.wrapped)
		if err != nil || hooks == nil {
			return false, false
		}
		if !stripClaudeStyle(hooks, spec.command) {
			return false, false
		}
		if spec.wrapped && len(hooks) == 0 {
			delete(config, "hooks")
		}
		return true, len(config) == 0
	case primeShapeAntigravity:
		if !stripAntigravityStyle(config, spec.command) {
			return false, false
		}
		return true, len(config) == 0
	default:
		return false, false
	}
}

func mutableHooks(config map[string]interface{}, wrapped bool) (map[string]interface{}, error) {
	if !wrapped {
		return config, nil
	}
	raw, ok := config["hooks"]
	if !ok || raw == nil {
		hooks := map[string]interface{}{}
		config["hooks"] = hooks
		return hooks, nil
	}
	hooks, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.New("hooks is not an object")
	}
	return hooks, nil
}

func readHooks(config map[string]interface{}, wrapped bool) (map[string]interface{}, error) {
	if !wrapped {
		return config, nil
	}
	raw, ok := config["hooks"]
	if !ok || raw == nil {
		return nil, nil
	}
	hooks, ok := raw.(map[string]interface{})
	if !ok {
		return nil, errors.New("hooks is not an object")
	}
	return hooks, nil
}

func ensureClaudeStyle(hooks map[string]interface{}, command string) (bool, error) {
	if raw, ok := hooks["SessionStart"]; ok && raw != nil {
		if _, isSlice := raw.([]interface{}); !isSlice {
			return false, errors.New("SessionStart is not an array")
		}
	}
	if groupHasCommand(hooks, "SessionStart", command) {
		return false, nil
	}
	hooks["SessionStart"] = append(toInterfaceSlice(hooks["SessionStart"]), claudeStyleGroup(command))
	return true, nil
}

func claudeStyleGroup(command string) map[string]interface{} {
	return map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": command,
			},
		},
	}
}

func groupHasCommand(hooks map[string]interface{}, event, command string) bool {
	for _, entry := range toInterfaceSlice(hooks[event]) {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		for _, cmd := range toInterfaceSlice(entryMap["hooks"]) {
			cmdMap, ok := cmd.(map[string]interface{})
			if !ok {
				continue
			}
			if cmdMap["command"] == command {
				return true
			}
		}
	}
	return false
}

func stripClaudeStyle(hooks map[string]interface{}, command string) bool {
	entries := toInterfaceSlice(hooks["SessionStart"])
	if entries == nil {
		return false
	}
	filtered := make([]interface{}, 0, len(entries))
	removed := false
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			filtered = append(filtered, entry)
			continue
		}
		cmds := toInterfaceSlice(entryMap["hooks"])
		if cmds == nil {
			filtered = append(filtered, entry)
			continue
		}
		kept := make([]interface{}, 0, len(cmds))
		for _, cmd := range cmds {
			cmdMap, ok := cmd.(map[string]interface{})
			if !ok {
				kept = append(kept, cmd)
				continue
			}
			if cmdMap["command"] == command {
				removed = true
				continue
			}
			kept = append(kept, cmd)
		}
		if len(kept) == 0 {
			continue
		}
		entryMap["hooks"] = kept
		filtered = append(filtered, entryMap)
	}
	if !removed {
		return false
	}
	if len(filtered) == 0 {
		delete(hooks, "SessionStart")
	} else {
		hooks["SessionStart"] = filtered
	}
	return true
}

func ensureAntigravityStyle(config map[string]interface{}, command string) (bool, error) {
	raw, ok := config[antigravityHookName]
	var def map[string]interface{}
	switch {
	case !ok || raw == nil:
		def = map[string]interface{}{}
		config[antigravityHookName] = def
	default:
		var isMap bool
		def, isMap = raw.(map[string]interface{})
		if !isMap {
			return false, fmt.Errorf("%s is not an object", antigravityHookName)
		}
	}
	if rawEvents, ok := def["SessionStart"]; ok && rawEvents != nil {
		if _, isSlice := rawEvents.([]interface{}); !isSlice {
			return false, errors.New("SessionStart is not an array")
		}
	}
	// 旧的 additionalContext 命令留在同一条 beads-prime 上会和 injectSteps 各跑一次。
	entries := stripFlatCommands(toInterfaceSlice(def["SessionStart"]), SessionPrimeCommand)
	changed := len(entries) != len(toInterfaceSlice(def["SessionStart"]))
	if !flatHasCommand(entries, command) {
		entries = append(entries, map[string]interface{}{
			"type":    "command",
			"command": command,
		})
		changed = true
	}
	if !changed {
		return false, nil
	}
	def["SessionStart"] = entries
	return true, nil
}

func antigravityHasCommand(config map[string]interface{}, command string) bool {
	def, ok := config[antigravityHookName].(map[string]interface{})
	if !ok {
		return false
	}
	for _, entry := range toInterfaceSlice(def["SessionStart"]) {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if entryMap["command"] == command {
			return true
		}
	}
	return false
}

func stripAntigravityStyle(config map[string]interface{}, command string) bool {
	def, ok := config[antigravityHookName].(map[string]interface{})
	if !ok {
		return false
	}
	entries := toInterfaceSlice(def["SessionStart"])
	filtered := stripFlatCommands(entries, command, SessionPrimeCommand)
	if len(filtered) == len(entries) {
		return false
	}
	if len(filtered) == 0 {
		delete(def, "SessionStart")
	} else {
		def["SessionStart"] = filtered
	}
	if len(def) == 0 {
		delete(config, antigravityHookName)
	}
	return true
}

func flatHasCommand(entries []interface{}, command string) bool {
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if entryMap["command"] == command {
			return true
		}
	}
	return false
}

func stripFlatCommands(entries []interface{}, commands ...string) []interface{} {
	owned := map[string]bool{}
	for _, command := range commands {
		owned[command] = true
	}
	filtered := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			filtered = append(filtered, entry)
			continue
		}
		command, _ := entryMap["command"].(string)
		if owned[command] {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

// pruneEmptyParents 从 path 的父目录往上删空目录，停在 root，也不删还装着别的文件的目录。
func pruneEmptyParents(path, root string) {
	dir := filepath.Dir(path)
	root = filepath.Clean(root)
	for {
		cleaned := filepath.Clean(dir)
		if cleaned == root || cleaned == "." || cleaned == string(filepath.Separator) {
			return
		}
		if err := RemoveEmptyDir(dir); err != nil {
			return
		}
		if DirExists(dir) {
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}
