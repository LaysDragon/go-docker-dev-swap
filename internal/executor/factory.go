package executor

import (
	"github.com/laysdragon/go-docker-dev-swap/internal/config"
)

// NewExecutor 根據配置建立對應的 Executor
func NewExecutor(cfg *config.Config) (Executor, error) {
	if cfg.Mode == "local" {
		return NewLocalExecutor(cfg)
	}
	return NewRemoteExecutor(cfg)
}
