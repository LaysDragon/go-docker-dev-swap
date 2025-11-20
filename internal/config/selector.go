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
		modeLabel := "本地"
		if h.Mode == "remote" {
			modeLabel = fmt.Sprintf("遠端 %s@%s:%d", h.User, h.Host, h.Port)
		}
		hostOptions[i] = huh.NewOption(
			fmt.Sprintf("%s (%s)", h.Name, modeLabel),
			key,
		)
	}

	// 建立表單（先選擇組件和主機）
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("選擇本地組件").
				Options(componentOptions...).
				Value(&componentName),

			huh.NewSelect[string]().
				Title("選擇主機").
				Options(hostOptions...).
				Value(&hostName),
		),
	)

	// 執行表單
	if err := form.Run(); err != nil {
		return nil, fmt.Errorf("選擇失敗: %w", err)
	}

	// 設定選擇的組件和主機
	selectedComponent := cfg.Components[componentName]
	selectedHost := cfg.Hosts[hostName]

	// 選擇該主機上的專案
	if len(selectedHost.Projects) == 0 {
		return nil, fmt.Errorf("主機 '%s' 沒有可用的專案配置", hostName)
	}

	projectKeys := sortedKeys(selectedHost.Projects)
	projectOptions := make([]huh.Option[string], len(projectKeys))
	for i, key := range projectKeys {
		p := selectedHost.Projects[key]
		label := fmt.Sprintf("%s (compose: %s)", p.Name, p.ComposeDir)
		if p.Type == ProjectTypeContainer {
			label = fmt.Sprintf("%s (docker container: %s)", p.Name, selectedComponent.TargetService)
		}
		projectOptions[i] = huh.NewOption(label, key)
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

	selectedProject := selectedHost.Projects[projectName]

	// 合併 Component 和全局預設值（就地修改 selectedComponent）
	// 如果 Component 中的欄位為 nil，使用全局預設值覆蓋
	if selectedComponent.LogFile == nil || *selectedComponent.LogFile == "" {
		selectedComponent.LogFile = &cfg.LogFile
	}

	if selectedComponent.InitialScripts == nil || *selectedComponent.InitialScripts == "" {
		selectedComponent.InitialScripts = &cfg.InitialScripts
	}

	if selectedComponent.DlvConfig == nil {
		selectedComponent.DlvConfig = &cfg.DlvConfig
	}

	// 建立 RuntimeConfig（現在 selectedComponent 保證所有欄位都有值）
	rc := &RuntimeConfig{
		Mode:                 selectedHost.Mode,
		Component:            selectedComponent,
		Host:                 selectedHost,
		Project:              selectedProject,
		UseSudo:              selectedHost.UseSudo,
		SudoPassword:         selectedHost.SudoPassword,
		DockerCommand:        selectedHost.DockerCommand,
		DockerComposeCommand: selectedHost.DockerComposeCommand,
	}

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
