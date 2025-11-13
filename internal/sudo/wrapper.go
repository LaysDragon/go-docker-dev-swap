package sudo

import (
	"fmt"
	"strings"
)

// Wrapper 提供通用的 sudo 命令包装功能
// 支持临时提权 bash 执行命令，提高命令兼容性
type Wrapper struct {
	enabled  bool   // 是否启用 sudo
	password string // sudo 密码（可选）
}

// NewWrapper 创建 sudo 包装器
func NewWrapper(enabled bool, password string) *Wrapper {
	return &Wrapper{
		enabled:  enabled,
		password: password,
	}
}

// Wrap 将命令包装为带 sudo 的形式
// 使用临时提权 bash 来运行命令，提高兼容性
func (w *Wrapper) Wrap(command string) string {
	if !w.enabled {
		return command
	}

	// 转义命令中的单引号以避免 bash 语法错误
	escapedCmd := strings.ReplaceAll(command, "'", "'\\''")

	if w.password != "" {
		// 有密码：echo 'password' | sudo -S bash -c 'command'
		return fmt.Sprintf("echo '%s' | sudo -S bash -c '%s'", w.password, escapedCmd)
	}

	// 无密码：sudo bash -c 'command'
	return fmt.Sprintf("sudo bash -c '%s'", escapedCmd)
}

// WrapMultiple 包装多个命令参数组成的命令
// 例如: WrapMultiple("docker", "ps", "-a") -> sudo bash -c 'docker ps -a'
func (w *Wrapper) WrapMultiple(parts ...string) string {
	command := strings.Join(parts, " ")
	return w.Wrap(command)
}

// Enabled 返回是否启用 sudo
func (w *Wrapper) Enabled() bool {
	return w.enabled
}

// HasPassword 返回是否配置了密码
func (w *Wrapper) HasPassword() bool {
	return w.password != ""
}
