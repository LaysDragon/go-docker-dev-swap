package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RemoteHost          RemoteHost `yaml:"remote_host"`
	ComposeDir          string     `yaml:"compose_dir"`
	TargetService       string     `yaml:"target_service"`
	LocalBinary         string     `yaml:"local_binary"`
	RemoteWorkDir       string     `yaml:"remote_work_dir"`        // 遠端工作目錄
	RemoteBinaryName    string     `yaml:"remote_binary_name"`     // 遠端執行檔名稱
	ContainerBinaryPath string     `yaml:"container_binary_path"`
	DebuggerPort        int        `yaml:"debugger_port"`
	ExtraPorts          []int      `yaml:"extra_ports"`
	DlvConfig           DlvConfig  `yaml:"dlv_config"`
	InitialScripts      string     `yaml:"initial_scripts"`
}

type RemoteHost struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	KeyFile  string `yaml:"key_file"`
}

type DlvConfig struct {
	Enabled bool   `yaml:"enabled"`
	Port    int    `yaml:"port"`
	Args    string `yaml:"args"`
}

// GetRemoteBinaryPath 返回完整的遠端執行檔路徑
func (c *Config) GetRemoteBinaryPath() string {
	return fmt.Sprintf("%s/%s", c.RemoteWorkDir, c.RemoteBinaryName)
}

// GetRemoteDlvPath 返回完整的遠端 dlv 路徑
func (c *Config) GetRemoteDlvPath() string {
	return fmt.Sprintf("%s/dlv", c.RemoteWorkDir)
}

// GetRemoteInitScriptPath 返回完整的遠端初始化腳本路徑
func (c *Config) GetRemoteInitScriptPath() string {
	return fmt.Sprintf("%s/init.sh", c.RemoteWorkDir)
}

// GetRemoteEntryScriptPath 返回完整的遠端入口腳本路徑
func (c *Config) GetRemoteEntryScriptPath() string {
	return fmt.Sprintf("%s/entry.sh", c.RemoteWorkDir)
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("讀取配置檔案失敗: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置檔案失敗: %w", err)
	}

	// 設定預設值
	if cfg.RemoteHost.Port == 0 {
		cfg.RemoteHost.Port = 22
	}
	if cfg.DebuggerPort == 0 {
		cfg.DebuggerPort = 2345
	}
	if cfg.RemoteWorkDir == "" {
		cfg.RemoteWorkDir = "/tmp/dev-binaries"
	}
	if cfg.RemoteBinaryName == "" {
		cfg.RemoteBinaryName = "service"
	}

	return &cfg, nil
}
