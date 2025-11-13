package config

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/huh"
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
	
	// 如果是本地模式，只需要選擇組件
	if cfg.Mode == "local" {
		componentName, err := selectFromMap("選擇本地組件", cfg.Components, func(c Component) string {
			return fmt.Sprintf("%s (service: %s, binary: %s)", c.Name, c.TargetService, c.LocalBinary)
		})
		if err != nil {
			return nil, err
		}
		rc.Component = cfg.Components[componentName]
		return rc, nil
	}
	
	// 遠端模式：建立完整的表單
	var componentName, hostName, projectName string
	
	// 準備組件選項
	componentKeys := sortedKeys(cfg.Components)
	componentOptions := make([]huh.Option[string], len(componentKeys))
	for i, key := range componentKeys {
		c := cfg.Components[key]
		componentOptions[i] = huh.NewOption(
			fmt.Sprintf("%s (service: %s, binary: %s)", c.Name, c.TargetService, c.LocalBinary),
			key,
		)
	}
	
	// 準備主機選項
	hostKeys := sortedKeys(cfg.Hosts)
	hostOptions := make([]huh.Option[string], len(hostKeys))
	for i, key := range hostKeys {
		h := cfg.Hosts[key]
		hostOptions[i] = huh.NewOption(
			fmt.Sprintf("%s (%s@%s:%d)", h.Name, h.User, h.Host, h.Port),
			key,
		)
	}
	
	// 建立表單（先不包含專案選擇，因為需要根據主機過濾）
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("選擇本地組件").
				Options(componentOptions...).
				Value(&componentName),
			
			huh.NewSelect[string]().
				Title("選擇遠端主機").
				Options(hostOptions...).
				Value(&hostName),
		),
	)
	
	// 執行表單
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("選擇失敗: %w", err)
	}
	
	// 設定選擇的組件和主機
	rc.Component = cfg.Components[componentName]
	rc.Host = cfg.Hosts[hostName]
	
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
	
	// 選擇專案（單獨的表單，因為選項依賴於主機選擇）
	projectKeys := sortedKeys(availableProjects)
	projectOptions := make([]huh.Option[string], len(projectKeys))
	for i, key := range projectKeys {
		p := availableProjects[key]
		projectOptions[i] = huh.NewOption(
			fmt.Sprintf("%s (compose: %s)", p.Name, p.ComposeDir),
			key,
		)
	}
	
	projectForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("選擇專案").
				Options(projectOptions...).
				Value(&projectName),
		),
	)
	
	if err := projectForm.Run(); err != nil {
		return nil, fmt.Errorf("選擇專案失敗: %w", err)
	}
	
	rc.Project = cfg.Projects[projectName]
	
	return rc, nil
}

// selectFromMap 從 map 中選擇一項，使用 huh 互動式選擇器
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
	
	// 建立選項列表
	options := make([]huh.Option[string], len(keys))
	for i, key := range keys {
		item := items[key]
		options[i] = huh.NewOption(displayFunc(item), key)
	}
	
	// 選擇的結果
	var selected string
	
	// 建立選擇表單
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title(prompt).
				Options(options...).
				Value(&selected),
		),
	)
	
	// 執行表單
	if err := form.Run(); err != nil {
		return "", fmt.Errorf("選擇失敗: %w", err)
	}
	
	return selected, nil
}

// sortedKeys 返回 map 的排序後的 keys
func sortedKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
