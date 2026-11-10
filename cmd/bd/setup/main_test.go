package setup

import (
	"os"
	"testing"
)

// TestMain 让本包的测试默认按「没有 PRIME.md 覆盖」运行：开发机上的
// ~/.config/beads/PRIME.md 不应改变生成的内容。需要覆盖的测试用 withPrimeOverride。
func TestMain(m *testing.M) {
	primeOverrideActiveImpl = func(bool) bool { return false }
	os.Exit(m.Run())
}
