package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config 主配置結構，支援多組組件、主機和專案配置
type Config struct {
	// 全域設定
	Mode                 string `mapstructure:"mode"`                   // "local" 或 "remote"，預設 "remote"
	UseSudo              bool   `mapstructure:"use_sudo"`               // 是否使用 sudo
	SudoPassword         string `mapstructure:"sudo_password"`          // sudo 密碼（可選）
	DockerCommand        string `mapstructure:"docker_command"`         // docker 命令（默認 "docker"）
	DockerComposeCommand string `mapstructure:"docker_compose_command"` // docker-compose 命令（默認 "docker compose"）
	
	// 多組配置
	Components map[string]Component `mapstructure:"components"` // 本地組件配置（key 為組件名稱）
	Hosts      map[string]Host      `mapstructure:"hosts"`      // 遠端主機配置（key 為主機名稱）
	Projects   map[string]Project   `mapstructure:"projects"`   // 專案配置（key 為專案名稱）
}

// Component 本地組件配置
type Component struct {
	Name                string    `mapstructure:"name"`                  // 組件名稱（顯示用）
	LocalBinary         string    `mapstructure:"local_binary"`          // 本地編譯的執行檔路徑
	TargetService       string    `mapstructure:"target_service"`        // 目標服務名稱
	ContainerBinaryPath string    `mapstructure:"container_binary_path"` // 容器內的執行檔路徑
	DebuggerPort        int       `mapstructure:"debugger_port"`         // Debugger 端口
	ExtraPorts          []int     `mapstructure:"extra_ports"`           // 額外需要暴露的端口
	DlvConfig           DlvConfig `mapstructure:"dlv_config"`            // Delve 配置
	InitialScripts      string    `mapstructure:"initial_scripts"`       // 容器啟動前執行的初始化腳本
	LogFile             string    `mapstructure:"log_file"`              // 日誌文件路徑
}

// Host 遠端主機配置
type Host struct {
	Name     string `mapstructure:"name"`      // 主機名稱（顯示用）
	Host     string `mapstructure:"host"`      // 主機地址
	Port     int    `mapstructure:"port"`      // SSH 端口
	User     string `mapstructure:"user"`      // SSH 用戶名
	Password string `mapstructure:"password"`  // SSH 密碼
	KeyFile  string `mapstructure:"key_file"`  // SSH 私鑰路徑
	
	RemoteWorkDir    string `mapstructure:"remote_work_dir"`    // 遠端工作目錄
	RemoteBinaryName string `mapstructure:"remote_binary_name"` // 遠端執行檔名稱
}

// Project 專案配置（對應一個 docker-compose 專案）
type Project struct {
	Name       string `mapstructure:"name"`        // 專案名稱（顯示用）
	HostRef    string `mapstructure:"host"`        // 關聯的主機名稱（hosts 中的 key）
	ComposeDir string `mapstructure:"compose_dir"` // docker-compose.yml 所在目錄
}

// DlvConfig Delve 調試器配置
type DlvConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Port      int    `mapstructure:"port"`
	Args      string `mapstructure:"args"`
	LocalPath string `mapstructure:"local_path"` // 本地 dlv 路徑，為空則自動搜尋
}

// RemoteHost SSH 連接配置（用於 executor）
type RemoteHost struct {
	Host     string
	Port     int
	User     string
	Password string
	KeyFile  string
}

// RuntimeConfig 運行時選擇的配置組合
type RuntimeConfig struct {
	Mode                 string
	Component            Component
	Host                 Host
	Project              Project
	UseSudo              bool
	SudoPassword         string
	DockerCommand        string
	DockerComposeCommand string
}

// defaultValues 定義所有配置項的預設值
var defaultValues = struct {
	Mode                 string
	UseSudo              bool
	SudoPassword         string
	DockerCommand        string
	DockerComposeCommand string
	
	// Component 預設值
	Component struct {
		ContainerBinaryPath string
		DebuggerPort        int
		ExtraPorts          []int
		InitialScripts      string
		LogFile             string
		
		DlvConfig struct {
			Enabled   bool
			Port      int
			Args      string
			LocalPath string
		}
	}
	
	// Host 預設值
	Host struct {
		Port             int
		RemoteWorkDir    string
		RemoteBinaryName string
	}
}{
	// 全域預設值
	Mode:                 "remote",
	UseSudo:              false,
	SudoPassword:         "",
	DockerCommand:        "docker",
	DockerComposeCommand: "docker compose",
	
	// Component 預設值
	Component: struct {
		ContainerBinaryPath string
		DebuggerPort        int
		ExtraPorts          []int
		InitialScripts      string
		LogFile             string
		DlvConfig           struct {
			Enabled   bool
			Port      int
			Args      string
			LocalPath string
		}
	}{
		ContainerBinaryPath: "/app/service",
		DebuggerPort:        2345,
		ExtraPorts:          []int{},
		InitialScripts:      "",
		LogFile:             "",
		DlvConfig: struct {
			Enabled   bool
			Port      int
			Args      string
			LocalPath string
		}{
			Enabled:   false,
			Port:      2345,
			Args:      "",
			LocalPath: "",
		},
	},
	
	// Host 預設值
	Host: struct {
		Port             int
		RemoteWorkDir    string
		RemoteBinaryName string
	}{
		Port:             22,
		RemoteWorkDir:    "/tmp/dev-binaries",
		RemoteBinaryName: "service",
	},
}

// GetRemoteBinaryPath 返回完整的遠端執行檔路徑
func (rc *RuntimeConfig) GetRemoteBinaryPath() string {
	return fmt.Sprintf("%s/%s", rc.Host.RemoteWorkDir, rc.Host.RemoteBinaryName)
}

// GetRemoteDlvPath 返回完整的遠端 dlv 路徑
func (rc *RuntimeConfig) GetRemoteDlvPath() string {
	return fmt.Sprintf("%s/dlv", rc.Host.RemoteWorkDir)
}

// GetRemoteInitScriptPath 返回完整的遠端初始化腳本路徑
func (rc *RuntimeConfig) GetRemoteInitScriptPath() string {
	return fmt.Sprintf("%s/init.sh", rc.Host.RemoteWorkDir)
}

// GetRemoteEntryScriptPath 返回完整的遠端入口腳本路徑
func (rc *RuntimeConfig) GetRemoteEntryScriptPath() string {
	return fmt.Sprintf("%s/entry.sh", rc.Host.RemoteWorkDir)
}

// GetDevContainerName 返回開發容器名稱
func (rc *RuntimeConfig) GetDevContainerName() string {
	return fmt.Sprintf("%s-dev", rc.Component.TargetService)
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
	// 全域預設值
	v.SetDefault("mode", defaultValues.Mode)
	v.SetDefault("use_sudo", defaultValues.UseSudo)
	v.SetDefault("sudo_password", defaultValues.SudoPassword)
	v.SetDefault("docker_command", defaultValues.DockerCommand)
	v.SetDefault("docker_compose_command", defaultValues.DockerComposeCommand)
	
	// 注意：Components、Hosts、Projects 是 map，無法在此設定預設值
	// 它們的預設值會在 validateConfig 中針對每個項目設定
}

// validateConfig 驗證配置
func validateConfig(cfg *Config) error {
	// 驗證執行模式
	if cfg.Mode != "local" && cfg.Mode != "remote" {
		return fmt.Errorf("mode 必須是 'local' 或 'remote'")
	}

	// 至少要有一個組件
	if len(cfg.Components) == 0 {
		return fmt.Errorf("必須至少定義一個 component")
	}
	
	// 驗證每個組件
	for name, comp := range cfg.Components {
		if comp.LocalBinary == "" {
			return fmt.Errorf("component '%s': local_binary 為必要配置", name)
		}
		if comp.TargetService == "" {
			return fmt.Errorf("component '%s': target_service 為必要配置", name)
		}
		
		// 驗證並轉換路徑
		binaryPath, err := filepath.Abs(comp.LocalBinary)
		if err != nil {
			return fmt.Errorf("component '%s': 無法解析 local_binary 路徑: %w", name, err)
		}
		comp.LocalBinary = binaryPath
		
		// 設定組件預設值
		if comp.ContainerBinaryPath == "" {
			comp.ContainerBinaryPath = defaultValues.Component.ContainerBinaryPath
		}
		if comp.DebuggerPort == 0 {
			comp.DebuggerPort = defaultValues.Component.DebuggerPort
		}
		if comp.ExtraPorts == nil {
			comp.ExtraPorts = defaultValues.Component.ExtraPorts
		}
		
		// 設定 DlvConfig 預設值
		if comp.DlvConfig.Port == 0 {
			comp.DlvConfig.Port = defaultValues.Component.DlvConfig.Port
		}
		// DlvConfig.Enabled 默認為 false（Go 零值）
		// DlvConfig.Args 和 LocalPath 默認為空字符串（Go 零值）
		
		cfg.Components[name] = comp
	}
	
	// 遠端模式需要主機和專案配置
	if cfg.Mode == "remote" {
		if len(cfg.Hosts) == 0 {
			return fmt.Errorf("遠端模式下，必須至少定義一個 host")
		}
		if len(cfg.Projects) == 0 {
			return fmt.Errorf("遠端模式下，必須至少定義一個 project")
		}
		
		// 驗證每個主機
		for name, host := range cfg.Hosts {
			if host.Host == "" {
				return fmt.Errorf("host '%s': host 為必要配置", name)
			}
			if host.User == "" {
				return fmt.Errorf("host '%s': user 為必要配置", name)
			}
			if host.Password == "" && host.KeyFile == "" {
				return fmt.Errorf("host '%s': 必須提供 password 或 key_file 其中之一", name)
			}
			
			// 驗證 key_file 路徑
			if host.KeyFile != "" {
				keyPath, err := filepath.Abs(host.KeyFile)
				if err != nil {
					return fmt.Errorf("host '%s': 無法解析 key_file 路徑: %w", name, err)
				}
				host.KeyFile = keyPath
				cfg.Hosts[name] = host
			}
			
			// 設定主機預設值
			if host.Port == 0 {
				host.Port = defaultValues.Host.Port
			}
			if host.RemoteWorkDir == "" {
				host.RemoteWorkDir = defaultValues.Host.RemoteWorkDir
			}
			if host.RemoteBinaryName == "" {
				host.RemoteBinaryName = defaultValues.Host.RemoteBinaryName
			}
			
			cfg.Hosts[name] = host
		}
		
		// 驗證每個專案
		for name, proj := range cfg.Projects {
			if proj.ComposeDir == "" {
				return fmt.Errorf("project '%s': compose_dir 為必要配置", name)
			}
			if proj.HostRef == "" {
				return fmt.Errorf("project '%s': host 為必要配置", name)
			}
			if _, exists := cfg.Hosts[proj.HostRef]; !exists {
				return fmt.Errorf("project '%s': 引用的 host '%s' 不存在", name, proj.HostRef)
			}
		}
	}
	
	return nil
}

