package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Mode                string     `mapstructure:"mode"` // "local" 或 "remote"，預設 "remote"
	RemoteHost          RemoteHost `mapstructure:"remote_host"`
	ComposeDir          string     `mapstructure:"compose_dir"`
	TargetService       string     `mapstructure:"target_service"`
	LocalBinary         string     `mapstructure:"local_binary"`
	RemoteWorkDir       string     `mapstructure:"remote_work_dir"`    // 遠端工作目錄（本地模式也使用此路徑）
	RemoteBinaryName    string     `mapstructure:"remote_binary_name"` // 遠端執行檔名稱
	ContainerBinaryPath string     `mapstructure:"container_binary_path"`
	DebuggerPort        int        `mapstructure:"debugger_port"`
	ExtraPorts          []int      `mapstructure:"extra_ports"`
	DlvConfig           DlvConfig  `mapstructure:"dlv_config"`
	InitialScripts      string     `mapstructure:"initial_scripts"`
	LogFile             string     `mapstructure:"log_file"` // 本地日誌文件路徑（可選）
}

type RemoteHost struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	KeyFile  string `mapstructure:"key_file"`
}

type DlvConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Port      int    `mapstructure:"port"`
	Args      string `mapstructure:"args"`
	LocalPath string `mapstructure:"local_path"` // 本地 dlv 路徑，為空則自動搜尋
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

// GetDevContainerName 返回開發容器名稱
func (c *Config) GetDevContainerName() string {
	return fmt.Sprintf("%s-dev", c.TargetService)
}

func Load(path string) (*Config, error) {
	v := viper.New()

	// 設定預設值
	setDefaults(v)

	// 設定配置檔案
	if path != "" {
		v.SetConfigFile(path)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.docker-dev-swap")
		v.AddConfigPath("/etc/docker-dev-swap")
	}

	// 讀取環境變數 (前綴為 DDS_)
	v.SetEnvPrefix("DDS")
	v.AutomaticEnv()

	// 讀取配置檔案
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("讀取配置檔案失敗: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置檔案失敗: %w", err)
	}

	// 驗證必要配置
	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// setDefaults 設定所有非必要選項的預設值
func setDefaults(v *viper.Viper) {
	// 執行模式預設值
	v.SetDefault("mode", "remote")

	// 遠端主機預設值
	v.SetDefault("remote_host.port", 22)

	// 遠端工作目錄預設值
	v.SetDefault("remote_work_dir", "/tmp/dev-binaries")
	v.SetDefault("remote_binary_name", "service")

	// 容器內執行檔路徑預設值
	v.SetDefault("container_binary_path", "/app/service")

	// Debugger 預設值
	v.SetDefault("debugger_port", 2345)
	v.SetDefault("extra_ports", []int{})

	// Delve 配置預設值
	v.SetDefault("dlv_config.enabled", true)
	v.SetDefault("dlv_config.port", 2345)
	v.SetDefault("dlv_config.args", "")
	v.SetDefault("dlv_config.local_path", "") // 空字符串表示自動搜尋

	// 初始化腳本預設值 (空字符串表示不執行)
	v.SetDefault("initial_scripts", "")

	// 日誌文件預設值 (空字符串表示不寫入文件)
	v.SetDefault("log_file", "")
}

// validateConfig 驗證必要配置項
func validateConfig(cfg *Config) error {
	// 驗證執行模式
	if cfg.Mode != "local" && cfg.Mode != "remote" {
		return fmt.Errorf("mode 必須是 'local' 或 'remote'")
	}

	// 遠端模式需要遠端主機配置
	if cfg.Mode == "remote" {
		if cfg.RemoteHost.Host == "" {
			return fmt.Errorf("遠端模式必須配置: remote_host.host")
		}
		if cfg.RemoteHost.User == "" {
			return fmt.Errorf("遠端模式必須配置: remote_host.user")
		}
		if cfg.RemoteHost.Password == "" && cfg.RemoteHost.KeyFile == "" {
			return fmt.Errorf("遠端模式必須提供 remote_host.password 或 remote_host.key_file 其中之一")
		}

		// 驗證 key_file 路徑（如果提供）
		if cfg.RemoteHost.KeyFile != "" {
			keyPath, err := filepath.Abs(cfg.RemoteHost.KeyFile)
			if err != nil {
				return fmt.Errorf("無法解析 key_file 路徑: %w", err)
			}
			cfg.RemoteHost.KeyFile = keyPath
		}
	}

	// 必要配置：Docker Compose 目錄
	if cfg.ComposeDir == "" {
		return fmt.Errorf("必要配置缺失: compose_dir")
	}

	// 必要配置：目標服務
	if cfg.TargetService == "" {
		return fmt.Errorf("必要配置缺失: target_service")
	}

	// 必要配置：本地二進制文件
	if cfg.LocalBinary == "" {
		return fmt.Errorf("必要配置缺失: local_binary")
	}

	// 驗證 local_binary 路徑
	binaryPath, err := filepath.Abs(cfg.LocalBinary)
	if err != nil {
		return fmt.Errorf("無法解析 local_binary 路徑: %w", err)
	}
	cfg.LocalBinary = binaryPath

	return nil
}
