package dlv

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FindLocal 查找本地 dlv 路徑
// 如果 configPath 不為空，則使用配置的路徑；否則使用 which 命令搜尋
func FindLocal(configPath string) (string, error) {
	// 如果配置中指定了 dlv 路徑，直接使用
	if configPath != "" {
		// 檢查配置的路徑是否存在
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return "", fmt.Errorf("配置的 dlv 路徑不存在: %s", configPath)
		}
		return configPath, nil
	}

	// 使用 which 命令查找 dlv
	cmd := exec.Command("which", "dlv")
	output, err := cmd.Output()
	if err != nil {
		// 找不到 dlv
		return "", nil
	}

	localDlvPath := strings.TrimSpace(string(output))
	if localDlvPath == "" {
		return "", nil
	}

	// 檢查文件是否真的存在
	if _, err := os.Stat(localDlvPath); os.IsNotExist(err) {
		return "", nil
	}

	return localDlvPath, nil
}
