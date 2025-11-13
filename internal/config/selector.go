package config

import (
	"fmt"
	"sort"
)

// InteractiveSelect 互動式選擇配置組合
// 返回 RuntimeConfig 供運行時使用
func (cfg *Config) InteractiveSelect() (*RuntimeConfig, error) {
	return cfg.selectMultiConfig()
}

// selectMultiConfig 互動式選擇多組配置
func (cfg *Config) selectMultiConfig() (*RuntimeConfig, error) {
	rc := &RuntimeConfig{
		Mode:                 cfg.Mode,
		UseSudo:              cfg.UseSudo,
		SudoPassword:         cfg.SudoPassword,
		DockerCommand:        cfg.DockerCommand,
		DockerComposeCommand: cfg.DockerComposeCommand,
	}
	
	// 步驟 1: 選擇本地組件
	componentName, err := selectFromMap("選擇本地組件", cfg.Components, func(c Component) string {
		return fmt.Sprintf("%s (service: %s, binary: %s)", c.Name, c.TargetService, c.LocalBinary)
	})
	if err != nil {
		return nil, err
	}
	rc.Component = cfg.Components[componentName]
	
	// 如果是本地模式，直接返回
	if cfg.Mode == "local" {
		return rc, nil
	}
	
	// 步驟 2: 選擇遠端主機
	hostName, err := selectFromMap("選擇遠端主機", cfg.Hosts, func(h Host) string {
		return fmt.Sprintf("%s (%s@%s:%d)", h.Name, h.User, h.Host, h.Port)
	})
	if err != nil {
		return nil, err
	}
	rc.Host = cfg.Hosts[hostName]
	
	// 步驟 3: 選擇該主機上的專案
	// 過濾出屬於選定主機的專案
	availableProjects := make(map[string]Project)
	for name, proj := range cfg.Projects {
		if proj.HostRef == hostName {
			availableProjects[name] = proj
		}
	}
	
	if len(availableProjects) == 0 {
		return nil, fmt.Errorf("主機 '%s' 沒有可用的專案配置", hostName)
	}
	
	projectName, err := selectFromMap("選擇專案", availableProjects, func(p Project) string {
		return fmt.Sprintf("%s (compose: %s)", p.Name, p.ComposeDir)
	})
	if err != nil {
		return nil, err
	}
	rc.Project = cfg.Projects[projectName]
	
	return rc, nil
}

// selectFromMap 從 map 中選擇一項
func selectFromMap[T any](prompt string, items map[string]T, displayFunc func(T) string) (string, error) {
	if len(items) == 0 {
		return "", fmt.Errorf("沒有可選項")
	}
	
	// 如果只有一個選項，直接返回
	if len(items) == 1 {
		for key := range items {
			return key, nil
		}
	}
	
	// 收集所有選項並排序（確保順序一致）
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	
	// 顯示選項
	fmt.Printf("\n%s:\n", prompt)
	for i, key := range keys {
		item := items[key]
		fmt.Printf("  [%d] %s\n", i+1, displayFunc(item))
	}
	
	// 讀取用戶輸入
	fmt.Print("\n請選擇 (輸入編號): ")
	var choice int
	if _, err := fmt.Scanf("%d", &choice); err != nil {
		return "", fmt.Errorf("無效的輸入: %w", err)
	}
	
	if choice < 1 || choice > len(keys) {
		return "", fmt.Errorf("選擇超出範圍: %d", choice)
	}
	
	return keys[choice-1], nil
}
