package docker

import (
	"fmt"
	"strings"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
)

// CommandBuilder 用於構建 Docker 命令的抽象層
// 注意：sudo 包装由 Executor 层统一处理，这里只负责构建 Docker 命令本身
type CommandBuilder struct {
	config *config.Config
}

// NewCommandBuilder 創建命令構建器
func NewCommandBuilder(cfg *config.Config) *CommandBuilder {
	return &CommandBuilder{
		config: cfg,
	}
}

// Docker 構建 docker 命令
func (cb *CommandBuilder) Docker(args ...string) string {
	cmd := cb.config.DockerCommand
	if cmd == "" {
		cmd = "docker"
	}
	
	parts := append([]string{cmd}, args...)
	return strings.Join(parts, " ")
}

// DockerCompose 構建 docker-compose/docker compose 命令
func (cb *CommandBuilder) DockerCompose(workDir string, args ...string) string {
	// 使用配置的 docker-compose 命令
	composeCmd := cb.config.DockerComposeCommand
	if composeCmd == "" {
		composeCmd = "docker compose" // 默認使用新版本
	}
	
	var cmd string
	if workDir != "" {
		cmd = fmt.Sprintf("cd %s && %s %s", workDir, composeCmd, strings.Join(args, " "))
	} else {
		cmd = fmt.Sprintf("%s %s", composeCmd, strings.Join(args, " "))
	}
	
	return cmd
}
