package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
	"github.com/laysdragon/go-docker-dev-swap/internal/dlv"
	"github.com/laysdragon/go-docker-dev-swap/internal/docker"
	"github.com/laysdragon/go-docker-dev-swap/internal/executor"
	"github.com/laysdragon/go-docker-dev-swap/internal/logger"
	"github.com/laysdragon/go-docker-dev-swap/internal/watcher"
)

var (
	configFile = flag.String("config", "config.yaml", "配置檔案路徑")
	service    = flag.String("service", "", "目標服務名稱")
)

func main() {
	flag.Parse()

	// 載入配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("載入配置失敗: %v", err)
	}

	// 如果指定了服務名稱，覆蓋配置
	if *service != "" {
		cfg.TargetService = *service
	}

	if cfg.TargetService == "" {
		log.Fatal("必須指定目標服務名稱")
	}

	log.Printf("啟動 docker-dev-swap")
	log.Printf("執行模式: %s", cfg.Mode)
	if cfg.Mode == "remote" {
		log.Printf("遠端主機: %s@%s", cfg.RemoteHost.User, cfg.RemoteHost.Host)
	}
	log.Printf("目標服務: %s", cfg.TargetService)

	// 建立 Executor
	exec, err := executor.NewExecutor(cfg)
	if err != nil {
		log.Fatalf("建立 Executor 失敗: %v", err)
	}
	defer exec.Close()

	// 建立 Docker 管理器
	dockerMgr := docker.NewManager(exec, cfg)

	// 啟動主流程
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 處理 Ctrl+C 信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n收到中斷信號，開始清理...")
		cancel()
	}()

	// 執行主要工作流程
	if err := run(ctx, dockerMgr, cfg, exec); err != nil {
		log.Fatalf("執行失敗: %v", err)
	}

	log.Println("已完成清理，程序退出")
}

func run(ctx context.Context, dockerMgr *docker.Manager, cfg *config.Config, exec executor.Executor) error {
	// 1. 獲取原始容器配置
	log.Println("獲取原始容器配置...")
	originalContainer, err := dockerMgr.GetContainerConfig(cfg.TargetService)
	if err != nil {
		return fmt.Errorf("獲取容器配置失敗: %w", err)
	}

	// 2. 查找並上傳 dlv（如果啟用且配置）
	var remoteDlvPath string
	if cfg.DlvConfig.Enabled {
		log.Println("查找本地 dlv...")

		// 查找 dlv
		localDlvPath, err := dlv.FindLocal(cfg.DlvConfig.LocalPath)
		if err != nil {
			log.Printf("查找 dlv 失敗: %v", err)
		} else if localDlvPath != "" {
			log.Printf("找到 dlv: %s", localDlvPath)

			// 上傳 dlv
			log.Println("上傳 dlv 到遠端...")
			remoteDlvPath = cfg.GetRemoteDlvPath()
			if err := exec.UploadFile(localDlvPath, remoteDlvPath); err != nil {
				log.Printf("上傳 dlv 失敗: %v", err)
				remoteDlvPath = "" // 重置，使用容器內的 dlv
			} else {
				log.Printf("dlv 已上傳到遠端: %s", remoteDlvPath)
			}
		} else {
			log.Println("本地未找到 dlv，將使用容器內的 dlv（如果有）")
		}
	}

	// 3. 上傳初始執行檔
	log.Println("上傳初始執行檔...")
	if err := exec.UploadFile(cfg.LocalBinary, cfg.GetRemoteBinaryPath()); err != nil {
		return fmt.Errorf("上傳執行檔失敗: %w", err)
	}
	if err := exec.CreateScript(fmt.Sprintf("%s\nsh ./entry.sh", cfg.InitialScripts), cfg.GetRemoteInitScriptPath()); err != nil {
		return fmt.Errorf("上傳初始腳本失敗: %w", err)
	}

	// 4. 停止原始容器
	log.Println("停止原始容器...")
	if err := dockerMgr.StopContainer(cfg.TargetService); err != nil {
		return fmt.Errorf("停止容器失敗: %w", err)
	}

	// 確保退出時恢復原始容器
	defer func() {
		log.Println("恢復原始容器...")
		if err := dockerMgr.RestoreOriginalContainer(cfg.TargetService); err != nil {
			log.Printf("恢復原始容器失敗: %v", err)
		} else {
			log.Println("原始容器已恢復")
		}
	}()

	// 5. 建立開發容器
	log.Println("建立開發容器...")
	devContainer, err := dockerMgr.CreateDevContainer(originalContainer, cfg, remoteDlvPath)
	if err != nil {
		// 檢查是否為容器名稱衝突錯誤
		if strings.Contains(err.Error(), "發現殘留的開發容器") {
			log.Println("發現殘留的開發容器")
			log.Print("是否要清理殘留容器？(y/N): ")

			var response string
			fmt.Scanln(&response)

			if strings.ToLower(strings.TrimSpace(response)) == "y" {
				log.Println("清理殘留容器...")
				if err := dockerMgr.RemoveDevContainerIfExists(cfg.GetDevContainerName()); err != nil {
					return fmt.Errorf("清理殘留容器失敗: %w", err)
				}
				log.Println("殘留容器已清理")

				// 重試建立開發容器
				log.Println("重新建立開發容器...")
				devContainer, err = dockerMgr.CreateDevContainer(originalContainer, cfg, remoteDlvPath)
				if err != nil {
					return fmt.Errorf("建立開發容器失敗: %w", err)
				}
			} else {
				return fmt.Errorf("用戶取消操作")
			}
		} else {
			return fmt.Errorf("建立開發容器失敗: %w", err)
		}
	}

	// 確保退出時清理開發容器
	defer func() {
		log.Println("清理開發容器...")
		if err := dockerMgr.RemoveDevContainer(devContainer.Name); err != nil {
			log.Printf("清理開發容器失敗: %v", err)
		} else {
			log.Println("開發容器已清理")
		}
	}()

	// 6. 啟動開發容器
	log.Println("啟動開發容器...")
	if err := dockerMgr.StartContainer(devContainer.Name); err != nil {
		return fmt.Errorf("啟動開發容器失敗: %w", err)
	}

	// 7. 建立 SSH Tunnel (用於 Debugger) - 僅遠端模式
	var tunnel executor.TunnelCloser
	if exec.IsRemote() {
		log.Println("建立 SSH Tunnel...")
		tunnel, err = exec.CreateTunnel(cfg.DebuggerPort, cfg.DebuggerPort)
		if err != nil {
			return fmt.Errorf("建立 SSH Tunnel 失敗: %w", err)
		}
		defer tunnel.Close()
		log.Printf("Debugger 可在 localhost:%d 連接", cfg.DebuggerPort)
	} else {
		log.Println("本地模式，跳過建立 SSH Tunnel")
	}

	// 8. 啟動檔案監控
	log.Println("啟動檔案監控...")
	fileWatcher := watcher.New(cfg.LocalBinary, func(path string) {
		log.Printf("偵測到檔案更新: %s", path)

		// 上傳新檔案
		log.Println("上傳新執行檔...")
		if err := exec.UploadFile(cfg.LocalBinary, cfg.GetRemoteBinaryPath()); err != nil {
			log.Printf("上傳失敗: %v", err)
			return
		}

		// 重啟容器
		log.Println("重啟開發容器...")
		if err := dockerMgr.RestartContainer(devContainer.Name); err != nil {
			log.Printf("重啟失敗: %v", err)
			return
		}

		log.Println("容器已重啟，新版本已部署")
	})

	if err := fileWatcher.Start(ctx); err != nil {
		return fmt.Errorf("啟動檔案監控失敗: %w", err)
	}

	// 9. 啟動日誌監控
	log.Println("啟動容器日誌監控...")
	if cfg.LogFile != "" {
		log.Printf("日誌將寫入文件: %s", cfg.LogFile)
	}

	logFollower := logger.NewFollower(exec, devContainer.Name, cfg.LogFile)
	go func() {
		if err := logFollower.Start(ctx); err != nil && err != context.Canceled {
			log.Printf("日誌監控停止: %v", err)
		}
	}()

	log.Println("開發環境已就緒！")
	log.Println("   - 按 Ctrl+C 退出並清理")
	log.Println("   - 修改並編譯二進制檔案會自動部署")
	log.Println("   - 容器日誌將顯示在下方")
	log.Println("==========================================")

	// 等待退出信號
	<-ctx.Done()

	return nil
}
