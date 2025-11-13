package docker

import (
	"fmt"
	"strings"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
	"github.com/laysdragon/go-docker-dev-swap/internal/sudo"
)

// CommandBuilder 用於構建 Docker 命令的抽象層
type CommandBuilder struct {
	config      *config.Config
	sudoWrapper *sudo.Wrapper
}

// NewCommandBuilder 創建命令構建器
func NewCommandBuilder(cfg *config.Config) *CommandBuilder {
	return &CommandBuilder{
		config:      cfg,
		sudoWrapper: sudo.NewWrapper(cfg.UseSudo, cfg.SudoPassword),
	}
}

// Docker 構建 docker 命令
func (cb *CommandBuilder) Docker(args ...string) string {
	cmd := cb.config.DockerCommand
	if cmd == "" {
		cmd = "docker"
	}
	
	parts := append([]string{cmd}, args...)
	return cb.sudoWrapper.WrapMultiple(parts...)
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
	
	return cb.sudoWrapper.Wrap(cmd)
}
