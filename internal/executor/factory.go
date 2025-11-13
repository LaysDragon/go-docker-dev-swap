package executor

import (
	"github.com/laysdragon/go-docker-dev-swap/internal/config"
)

// NewExecutor 根據配置建立對應的 Executor
func NewExecutor(rc *config.RuntimeConfig) (Executor, error) {
	if rc.Mode == "local" {
		return NewLocalExecutor(rc)
	}
	return NewRemoteExecutor(rc)
}
