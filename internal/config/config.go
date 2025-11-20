package config

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/viper"
)

const (
	ProjectTypeCompose        = "compose"
	ProjectTypeContainer      = "container"
	defaultContainerProjectID = "docker-container"
)

// Config 主配置結構，支援多組組件、主機和專案配置
type Config struct {
	// 全域預設值（可被 Component 層級覆蓋）
	LogFile        string    `mapstructure:"log_file"`        // 日誌文件路徑預設值
	InitialScripts string    `mapstructure:"initial_scripts"` // 初始化腳本預設值
	DlvConfig      DlvConfig `mapstructure:"dlv_config"`      // Delve 配置預設值

	// 多組配置
	Components map[string]Component `mapstructure:"components"` // 本地組件配置（key 為組件名稱）
	Hosts      map[string]Host      `mapstructure:"hosts"`      // 主機配置（key 為主機名稱）
}

// Component 本地組件配置
type Component struct {
	Name                string     `mapstructure:"name"`                  // 組件名稱（顯示用）
	LocalBinary         string     `mapstructure:"local_binary"`          // 本地編譯的執行檔路徑
	TargetService       string     `mapstructure:"target_service"`        // 目標服務名稱
	ContainerBinaryPath string     `mapstructure:"container_binary_path"` // 容器內的執行檔路徑
	DebuggerPort        int        `mapstructure:"debugger_port"`         // Debugger 端口
	ExtraPorts          []int      `mapstructure:"extra_ports"`           // 額外需要暴露的端口
	DlvConfig           *DlvConfig `mapstructure:"dlv_config"`            // Delve 配置（nil 表示使用全局預設）
	InitialScripts      *string    `mapstructure:"initial_scripts"`       // 容器啟動前執行的初始化腳本（nil 表示使用全局預設）
	LogFile             *string    `mapstructure:"log_file"`              // 日誌文件路徑（nil 表示使用全局預設）
}

// Host 主機配置（包含 mode、sudo、docker、projects）
type Host struct {
	Name     string `mapstructure:"name"`     // 主機名稱（顯示用）
	Mode     string `mapstructure:"mode"`     // "local" 或 "remote"
	Host     string `mapstructure:"host"`     // 主機地址（remote 模式需要）
	Port     int    `mapstructure:"port"`     // SSH 端口（remote 模式）
	User     string `mapstructure:"user"`     // SSH 用戶名（remote 模式需要）
	Password string `mapstructure:"password"` // SSH 密碼（remote 模式）
	KeyFile  string `mapstructure:"key_file"` // SSH 私鑰路徑（remote 模式）

	RemoteWorkDir    string `mapstructure:"remote_work_dir"`    // 遠端工作目錄（remote 模式）
	RemoteBinaryName string `mapstructure:"remote_binary_name"` // 遠端執行檔名稱（remote 模式）

	// Sudo 配置
	UseSudo      bool   `mapstructure:"use_sudo"`      // 是否使用 sudo
	SudoPassword string `mapstructure:"sudo_password"` // sudo 密碼

	// Docker 命令配置
	DockerCommand        string `mapstructure:"docker_command"`         // docker 命令
	DockerComposeCommand string `mapstructure:"docker_compose_command"` // docker-compose 命令

	// 專案列表
	Projects map[string]Project `mapstructure:"projects"` // 該主機上的專案配置
}

// Project 專案配置（對應一個 docker-compose 專案）
type Project struct {
	Name       string `mapstructure:"name"`        // 專案名稱（顯示用）
	Type       string `mapstructure:"type"`        // 專案類型：compose 或 container
	ComposeDir string `mapstructure:"compose_dir"` // docker-compose.yml 所在目錄（compose 類型需要）
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
	Component            Component // 使用原始 Component（已在選擇時合併全局預設值）
	Host                 Host
	Project              Project
	UseSudo              bool
	SudoPassword         string
	DockerCommand        string
	DockerComposeCommand string
}

// defaultValues 定義所有配置項的預設值
var defaultValues = struct {
	// 全局預設值
	LogFile        string
	InitialScripts string
	DlvConfig      struct {
		Enabled   bool
		Port      int
		Args      string
		LocalPath string
	}

	// Component 預設值
	Component struct {
		ContainerBinaryPath string
		DebuggerPort        int
		ExtraPorts          []int
	}

	// Host 預設值
	Host struct {
		Mode                 string
		Port                 int
		RemoteWorkDir        string
		RemoteBinaryName     string
		UseSudo              bool
		SudoPassword         string
		DockerCommand        string
		DockerComposeCommand string
	}
}{
	// 全局預設值
	LogFile:        "",
	InitialScripts: "",
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

	// Component 預設值
	Component: struct {
		ContainerBinaryPath string
		DebuggerPort        int
		ExtraPorts          []int
	}{
		ContainerBinaryPath: "/app/service",
		DebuggerPort:        2345,
		ExtraPorts:          []int{},
	},

	// Host 預設值
	Host: struct {
		Mode                 string
		Port                 int
		RemoteWorkDir        string
		RemoteBinaryName     string
		UseSudo              bool
		SudoPassword         string
		DockerCommand        string
		DockerComposeCommand string
	}{
		Mode:                 "remote",
		Port:                 22,
		RemoteWorkDir:        "/tmp/dev-binaries",
		RemoteBinaryName:     "service",
		UseSudo:              false,
		SudoPassword:         "",
		DockerCommand:        "docker",
		DockerComposeCommand: "docker compose",
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
	// 全局預設值
	v.SetDefault("log_file", defaultValues.LogFile)
	v.SetDefault("initial_scripts", defaultValues.InitialScripts)
	v.SetDefault("dlv_config.enabled", defaultValues.DlvConfig.Enabled)
	v.SetDefault("dlv_config.port", defaultValues.DlvConfig.Port)
	v.SetDefault("dlv_config.args", defaultValues.DlvConfig.Args)
	v.SetDefault("dlv_config.local_path", defaultValues.DlvConfig.LocalPath)

	// 注意：Components、Hosts 是 map，無法在此設定預設值
	// 它們的預設值會在 validateConfig 中針對每個項目設定
}

// validateConfig 驗證配置
// validateConfig 驗證配置
func validateConfig(cfg *Config) error {
	// 至少要有一個組件
	if len(cfg.Components) == 0 {
		return fmt.Errorf("必須至少定義一個 component")
	}

	// 至少要有一個主機
	if len(cfg.Hosts) == 0 {
		return fmt.Errorf("必須至少定義一個 host")
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

		// LogFile, InitialScripts, DlvConfig 如果為 nil，表示使用全局預設值
		// 在 InteractiveSelect 時會處理合併邏輯

		cfg.Components[name] = comp
	}

	// 驗證每個主機
	for name, host := range cfg.Hosts {
		// 驗證 mode
		if host.Mode == "" {
			host.Mode = defaultValues.Host.Mode
		}
		if host.Mode != "local" && host.Mode != "remote" {
			return fmt.Errorf("host '%s': mode 必須是 'local' 或 'remote'", name)
		}

		// 遠端模式需要連接資訊
		if host.Mode == "remote" {
			if host.Host == "" {
				return fmt.Errorf("host '%s': remote 模式下 host 為必要配置", name)
			}
			if host.User == "" {
				return fmt.Errorf("host '%s': remote 模式下 user 為必要配置", name)
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
			}

			// 設定遠端模式預設值
			if host.Port == 0 {
				host.Port = defaultValues.Host.Port
			}

		}

		if host.RemoteWorkDir == "" {
			host.RemoteWorkDir = defaultValues.Host.RemoteWorkDir
		}
		if host.RemoteBinaryName == "" {
			host.RemoteBinaryName = defaultValues.Host.RemoteBinaryName
		}

		// 設定 Docker 命令預設值
		if host.DockerCommand == "" {
			host.DockerCommand = defaultValues.Host.DockerCommand
		}
		if host.DockerComposeCommand == "" {
			host.DockerComposeCommand = defaultValues.Host.DockerComposeCommand
		}

		// 確保 Projects map 存在
		if host.Projects == nil {
			host.Projects = make(map[string]Project)
		}

		for projName, proj := range host.Projects {
			if proj.Type == "" {
				proj.Type = ProjectTypeCompose
			}
			switch proj.Type {
			case ProjectTypeCompose:
				if proj.ComposeDir == "" {
					return fmt.Errorf("host '%s', project '%s': compose_dir 為必要配置", name, projName)
				}
			case ProjectTypeContainer:
				// container 模式不需要 compose_dir，亦不再接受額外容器名稱
			default:
				return fmt.Errorf("host '%s', project '%s': type 必須是 'compose' 或 'container'", name, projName)
			}

			if proj.Name == "" {
				proj.Name = projName
			}

			host.Projects[projName] = proj
		}

		// 為每個 host 追加預設的 container 專案，方便直接操作現有容器
		if _, exists := host.Projects[defaultContainerProjectID]; !exists {
			host.Projects[defaultContainerProjectID] = Project{
				Name: "Docker Container",
				Type: ProjectTypeContainer,
			}
		}

		cfg.Hosts[name] = host
	}

	return nil
}
