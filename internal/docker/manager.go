package docker

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/LaysDragonB/docker-dev-swap/internal/config"
	"github.com/LaysDragonB/docker-dev-swap/internal/ssh"
)

type Manager struct {
	ssh    *ssh.Client
	config *config.Config
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

func NewManager(sshClient *ssh.Client, cfg *config.Config) *Manager {
	return &Manager{
		ssh:    sshClient,
		config: cfg,
	}
}

func (m *Manager) GetContainerConfig(serviceName string) (*ContainerConfig, error) {
	// 獲取容器 ID
	cmd := fmt.Sprintf("cd %s && sudo docker compose ps -q %s", m.config.ComposeDir, serviceName)
	containerID, err := m.ssh.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("獲取容器 ID 失敗: %w", err)
	}
	containerID = strings.TrimSpace(containerID)

	// 獲取容器詳細資訊
	cmd = fmt.Sprintf("sudo docker inspect %s", containerID)
	output, err := m.ssh.Execute(cmd)
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
	cmd := fmt.Sprintf("cd %s && sudo docker compose stop %s", m.config.ComposeDir, serviceName)
	_, err := m.ssh.Execute(cmd)
	return err
}

func (m *Manager) CreateDevContainer(original *ContainerConfig, cfg *config.Config) (*DevContainer, error) {
	devName := fmt.Sprintf("%s-dev", original.Name)

	// 構建 docker run 命令
	var cmdParts []string
	cmdParts = append(cmdParts, "sudo docker run -d")
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
		cfg.RemoteBinaryPath, cfg.ContainerBinaryPath))

	// 端口映射
	if cfg.DlvConfig.Enabled {
		cmdParts = append(cmdParts, fmt.Sprintf("-p %d:%d",
			cfg.DlvConfig.Port, cfg.DlvConfig.Port))
	}
	for _, port := range cfg.ExtraPorts {
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

	// 標籤
	cmdParts = append(cmdParts, "-l dev-swap=true")

	// 映像
	cmdParts = append(cmdParts, original.Image)

	// 命令 (使用 dlv 或直接執行)
	if cfg.DlvConfig.Enabled {
		dlvCmd := fmt.Sprintf("dlv exec %s --headless --listen=:%d --api-version=2 --accept-multiclient %s",
			cfg.ContainerBinaryPath, cfg.DlvConfig.Port, cfg.DlvConfig.Args)
		cmdParts = append(cmdParts, fmt.Sprintf("sh -c '%s'", dlvCmd))
	} else {
		cmdParts = append(cmdParts, cfg.ContainerBinaryPath)
	}

	cmd := strings.Join(cmdParts, " ")
	output, err := m.ssh.Execute(cmd)
	if err != nil {
		return nil, fmt.Errorf("建立開發容器失敗: %w, output: %s", err, output)
	}

	return &DevContainer{
		Name:         devName,
		OriginalName: original.Name,
	}, nil
}

func (m *Manager) StartContainer(name string) error {
	cmd := fmt.Sprintf("sudo docker start %s", name)
	_, err := m.ssh.Execute(cmd)
	return err
}

func (m *Manager) RestartContainer(name string) error {
	cmd := fmt.Sprintf("sudo docker restart %s", name)
	_, err := m.ssh.Execute(cmd)
	return err
}

func (m *Manager) RemoveDevContainer(name string) error {
	cmd := fmt.Sprintf("sudo docker rm -f %s", name)
	_, err := m.ssh.Execute(cmd)
	return err
}

func (m *Manager) RestoreOriginalContainer(serviceName string) error {
	cmd := fmt.Sprintf("cd %s && sudo docker compose start %s", m.config.ComposeDir, serviceName)
	_, err := m.ssh.Execute(cmd)
	return err
}
