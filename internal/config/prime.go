package config

import (
	"os"
	"path/filepath"
)

// PRIME.md 覆盖让 bd prime 输出覆盖文件的内容，代替默认的工作流文字。
// bd prime 读取覆盖的内容；bd init 和 bd setup 只判断有没有覆盖：有覆盖时，生成的代理说明
// 只指向 bd prime。两边都调用这里，判断「有没有覆盖」的查找顺序才不会分叉。

const primeOverrideFile = "PRIME.md"

// GlobalPrimePath returns the user-level PRIME.md
// (<os.UserConfigDir()>/beads/PRIME.md) when it exists, or "".
// configDirOverride replaces os.UserConfigDir() in tests; pass "" in production.
func GlobalPrimePath(configDirOverride string) string {
	configDir := configDirOverride
	if configDir == "" {
		var err error
		configDir, err = os.UserConfigDir()
		if err != nil {
			return ""
		}
	}
	p := filepath.Join(configDir, "beads", primeOverrideFile)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return ""
}

// ReadPrimeOverride returns the PRIME.md override that applies to a workspace
// and true, or ("", false) when there is none. It checks, in priority order:
// .beads/PRIME.md under workspaceDir (clone-specific customization; "" means
// the cwd), PRIME.md in beadsDir (the resolved beads directory, which can be a
// redirect target: shared customization), then GlobalPrimePath.
// An empty beadsDir skips the second location.
func ReadPrimeOverride(workspaceDir, beadsDir string) (string, bool) {
	// #nosec G304 -- fixed file name under the workspace root (-C target or cwd)
	if content, err := os.ReadFile(filepath.Join(workspaceDir, ".beads", primeOverrideFile)); err == nil {
		return string(content), true
	}
	if beadsDir != "" {
		// #nosec G304 -- fixed file name under the resolved beads directory
		if content, err := os.ReadFile(filepath.Join(beadsDir, primeOverrideFile)); err == nil {
			return string(content), true
		}
	}
	if globalPath := GlobalPrimePath(""); globalPath != "" {
		// #nosec G304 -- fixed file name under the user config directory
		if content, err := os.ReadFile(globalPath); err == nil {
			return string(content), true
		}
	}
	return "", false
}
