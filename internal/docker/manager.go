package docker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
	"github.com/laysdragon/go-docker-dev-swap/internal/executor"
)

type Manager struct {
	executor executor.Executor
	config   *config.RuntimeConfig
	cmdBuilder *CommandBuilder
}

type ContainerConfig struct {
	Name       string
	Image      string
	Env        []string
	Volumes    []string
	Ports      []string
	Networks   []string
	Command    string
	WorkingDir string
	Labels     map[string]string
}

type DevContainer struct {
	Name         string
	OriginalName string
}

func NewManager(exec executor.Executor, rc *config.RuntimeConfig) *Manager {
	return &Manager{
		executor:   exec,
		config:     rc,
		cmdBuilder: NewCommandBuilder(rc),
	}
}

func (m *Manager) GetContainerConfig(serviceName string) (*ContainerConfig, error) {
	// 獲取容器 ID
	cmd := m.cmdBuilder.DockerCompose(m.config.Project.ComposeDir, "ps", "-q", serviceName, "-a")
	containerID, err := m.executor.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("獲取容器 ID 失敗: %w", err)
	}
	containerID = strings.TrimSpace(containerID)

	// 獲取容器詳細資訊
	cmd = m.cmdBuilder.Docker("inspect", containerID)
	output, err := m.executor.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("獲取容器資訊失敗: %w", err)
	}

	var inspectData []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &inspectData); err != nil {
		return nil, fmt.Errorf("解析容器資訊失敗: %w", err)
	}

	if len(inspectData) == 0 {
		return nil, fmt.Errorf("找不到容器")
	}

	data := inspectData[0]
	cfg := &ContainerConfig{
		Name:   serviceName,
		Labels: make(map[string]string),
	}

	// 解析配置
	if configData, ok := data["Config"].(map[string]interface{}); ok {
		if image, ok := configData["Image"].(string); ok {
			cfg.Image = image
		}
		if env, ok := configData["Env"].([]interface{}); ok {
			for _, e := range env {
				if envStr, ok := e.(string); ok {
					cfg.Env = append(cfg.Env, envStr)
				}
			}
		}
		if cmd, ok := configData["Cmd"].([]interface{}); ok {
			cmdStrs := make([]string, len(cmd))
			for i, c := range cmd {
				cmdStrs[i] = fmt.Sprint(c)
			}
			cfg.Command = strings.Join(cmdStrs, " ")
		}
		if wd, ok := configData["WorkingDir"].(string); ok {
			cfg.WorkingDir = wd
		}
		// 解析 Labels
		if labels, ok := configData["Labels"].(map[string]interface{}); ok {
			for k, v := range labels {
				if vStr, ok := v.(string); ok {
					cfg.Labels[k] = vStr
				}
			}
		}
	}

	// 解析掛載
	if mounts, ok := data["Mounts"].([]interface{}); ok {
		for _, m := range mounts {
			if mount, ok := m.(map[string]interface{}); ok {
				source := mount["Source"].(string)
				dest := mount["Destination"].(string)
				cfg.Volumes = append(cfg.Volumes, fmt.Sprintf("%s:%s", source, dest))
			}
		}
	}

	// 解析網路
	if networkSettings, ok := data["NetworkSettings"].(map[string]interface{}); ok {
		if networks, ok := networkSettings["Networks"].(map[string]interface{}); ok {
			for name := range networks {
				cfg.Networks = append(cfg.Networks, name)
			}
		}
	}

	return cfg, nil
}

func (m *Manager) StopContainer(serviceName string) error {
	cmd := m.cmdBuilder.DockerCompose(m.config.Project.ComposeDir, "stop", serviceName)
	_, err := m.executor.Execute(cmd)
	return err
}

// CheckDevContainerExists 檢查開發容器是否已存在並驗證是否為本工具創建的
func (m *Manager) CheckDevContainerExists(devName string) (exists bool, isDevSwap bool, containerID string, err error) {
	// 檢查容器是否存在
	cmd := m.cmdBuilder.Docker("ps", "-a", fmt.Sprintf("--filter name=^/%s$", devName), "--format '{{.ID}}'")
	output, err := m.executor.Execute(cmd)
	if err != nil {
		return false, false, "", fmt.Errorf("檢查容器失敗: %w", err)
	}

	containerID = strings.TrimSpace(output)
	if containerID == "" {
		return false, false, "", nil
	}

	// 容器存在，檢查是否有 dev-swap=true 標籤
	cmd = m.cmdBuilder.Docker("inspect", containerID, "--format '{{index .Config.Labels \"dev-swap\"}}'")
	output, err = m.executor.Execute(cmd)
	if err != nil {
		return true, false, containerID, fmt.Errorf("檢查容器標籤失敗: %w", err)
	}

	label := strings.TrimSpace(output)
	return true, label == "true", containerID, nil
}

// RemoveDevContainerIfExists 移除已存在的開發容器（如果存在且為本工具創建）
func (m *Manager) RemoveDevContainerIfExists(devName string) error {
	exists, isDevSwap, containerID, err := m.CheckDevContainerExists(devName)
	if err != nil {
		return err
	}

	if !exists {
		return nil // 容器不存在，無需清理
	}

	if !isDevSwap {
		return fmt.Errorf("容器 %s 存在但不是由 dev-swap 創建的，請手動處理", devName)
	}

	// 移除容器
	cmd := m.cmdBuilder.Docker("rm", "-f", containerID)
	_, err = m.executor.Execute(cmd)
	if err != nil {
		return fmt.Errorf("移除殘留容器失敗: %w", err)
	}

	return nil
}

func (m *Manager) CreateDevContainer(original *ContainerConfig, remoteDlvPath string) (*DevContainer, error) {
	devName := m.config.GetDevContainerName()

	// 檢查是否有殘留的開發容器
	exists, isDevSwap, containerID, err := m.CheckDevContainerExists(devName)
	if err != nil {
		return nil, fmt.Errorf("檢查容器狀態失敗: %w", err)
	}

	if exists {
		if isDevSwap {
			return nil, fmt.Errorf("發現殘留的開發容器 (ID: %s)，請使用清理選項", containerID)
		} else {
			return nil, fmt.Errorf("容器名稱 %s 已被使用但不是由 dev-swap 創建，請手動處理", devName)
		}
	}

	// 構建 docker run 命令
	var cmdParts []string
	cmdParts = append(cmdParts, "run -d")
	cmdParts = append(cmdParts, fmt.Sprintf("--name %s", devName))

	// 環境變數
	for _, env := range original.Env {
		cmdParts = append(cmdParts, fmt.Sprintf("-e '%s'", env))
	}

	// 原始掛載
	for _, vol := range original.Volumes {
		cmdParts = append(cmdParts, fmt.Sprintf("-v %s", vol))
	}

	// 新增執行檔掛載
	cmdParts = append(cmdParts, fmt.Sprintf("-v %s:%s",
		m.config.GetRemoteBinaryPath(), m.config.Component.ContainerBinaryPath))

	cmdParts = append(cmdParts, fmt.Sprintf("-v %s:/app/entry.sh", m.config.GetRemoteEntryScriptPath()))
	cmdParts = append(cmdParts, fmt.Sprintf("-v %s:/app/init.sh", m.config.GetRemoteInitScriptPath()))

	// 如果有 dlv，也掛載進去
	if remoteDlvPath != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-v %s:%s/dlv", remoteDlvPath, original.WorkingDir))
	}

	// 端口映射
	if m.config.Component.DlvConfig.Enabled {
		cmdParts = append(cmdParts, fmt.Sprintf("-p %d:%d",
			m.config.Component.DlvConfig.Port, m.config.Component.DlvConfig.Port))
	}
	for _, port := range m.config.Component.ExtraPorts {
		cmdParts = append(cmdParts, fmt.Sprintf("-p %d:%d", port, port))
	}

	// 網路
	for _, network := range original.Networks {
		cmdParts = append(cmdParts, fmt.Sprintf("--network %s", network))
	}

	// Working Directory
	if original.WorkingDir != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("-w %s", original.WorkingDir))
	}

	// 繼承原始容器的標籤
	for k, v := range original.Labels {
		// 跳過某些可能會造成衝突的標籤
		if strings.HasPrefix(k, "com.docker.compose") {
			continue
		}
		cmdParts = append(cmdParts, fmt.Sprintf("-l '%s=%s'", k, v))
	}

	// 添加開發容器標籤
	cmdParts = append(cmdParts, "-l dev-swap=true")

	// 映像
	cmdParts = append(cmdParts, original.Image)

	var entryParts []string
	// 命令 (使用 dlv 或直接執行)
	if m.config.Component.DlvConfig.Enabled {
		// 需要 continue 不然需要連線兩次應用才會正式開始執行，原因不明
		dlvCmd := fmt.Sprintf("./dlv exec %s --headless --listen=:%d --api-version=2 --accept-multiclient --continue %s",
			m.config.Component.ContainerBinaryPath, m.config.Component.DlvConfig.Port, m.config.Component.DlvConfig.Args)
		entryParts = append(entryParts, dlvCmd)
		//entryParts = append(entryParts, fmt.Sprintf("sh -c '%s'", dlvCmd))
	} else {
		entryParts = append(entryParts, m.config.Component.ContainerBinaryPath)
	}
	cmdParts = append(cmdParts, "sh /app/init.sh")

	if err := m.executor.CreateScript(strings.Join(entryParts, " "), m.config.GetRemoteEntryScriptPath()); err != nil {
		return nil, fmt.Errorf("上傳入口腳本失敗: %w", err)
	}

	// 使用 CommandBuilder 構建完整的 docker 命令
	cmd := m.cmdBuilder.Docker(cmdParts...)
	fmt.Printf("執行命令: %s\n", cmd)
	output, err := m.executor.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("建立開發容器失敗: %w, output: %s", err, output)
	}

	return &DevContainer{
		Name:         devName,
		OriginalName: original.Name,
	}, nil
}

func (m *Manager) StartContainer(name string) error {
	cmd := m.cmdBuilder.Docker("start", name)
	_, err := m.executor.Execute(cmd)
	return err
}

func (m *Manager) RestartContainer(name string) error {
	cmd := m.cmdBuilder.Docker("restart", name)
	_, err := m.executor.Execute(cmd)
	return err
}

func (m *Manager) RemoveDevContainer(name string) error {
	cmd := m.cmdBuilder.Docker("rm", "-f", name)
	_, err := m.executor.Execute(cmd)
	return err
}

func (m *Manager) RestoreOriginalContainer(serviceName string) error {
	cmd := m.cmdBuilder.DockerCompose(m.config.Project.ComposeDir, "start", serviceName)
	_, err := m.executor.Execute(cmd)
	return err
}

// CheckContainerRunning 檢查容器是否正在運行
func (m *Manager) CheckContainerRunning(containerName string) (bool, error) {
	cmd := m.cmdBuilder.Docker("ps", "-q", fmt.Sprintf("--filter name=^/%s$", containerName))
	output, err := m.executor.Execute(cmd)
	if err != nil {
		return false, fmt.Errorf("檢查容器運行狀態失敗: %w", err)
	}
	
	containerID := strings.TrimSpace(output)
	return containerID != "", nil
}
