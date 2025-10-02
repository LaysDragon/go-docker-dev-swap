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
	RemoteBinaryPath    string     `yaml:"remote_binary_path"`
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
	if cfg.RemoteBinaryPath == "" {
		cfg.RemoteBinaryPath = "/tmp/dev-binary"
	}

	return &cfg, nil
}
