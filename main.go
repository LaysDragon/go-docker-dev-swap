package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/LaysDragonB/docker-dev-swap/internal/config"
	"github.com/LaysDragonB/docker-dev-swap/internal/docker"
	"github.com/LaysDragonB/docker-dev-swap/internal/ssh"
	"github.com/LaysDragonB/docker-dev-swap/internal/watcher"
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

	log.Printf("🚀 啟動 docker-dev-swap")
	log.Printf("📡 遠端主機: %s@%s", cfg.RemoteHost.User, cfg.RemoteHost.Host)
	log.Printf("🎯 目標服務: %s", cfg.TargetService)

	// 建立 SSH 連接
	sshClient, err := ssh.NewClient(cfg.RemoteHost)
	if err != nil {
		log.Fatalf("SSH 連接失敗: %v", err)
	}
	defer sshClient.Close()

	// 建立 Docker 管理器
	dockerMgr := docker.NewManager(sshClient, cfg)

	// 啟動主流程
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 處理 Ctrl+C 信號
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n⚠️  收到中斷信號，開始清理...")
		cancel()
	}()

	// 執行主要工作流程
	if err := run(ctx, dockerMgr, cfg, sshClient); err != nil {
		log.Fatalf("執行失敗: %v", err)
	}

	log.Println("✅ 已完成清理，程序退出")
}

func run(ctx context.Context, dockerMgr *docker.Manager, cfg *config.Config, sshClient *ssh.Client) error {
	// 1. 獲取原始容器配置
	log.Println("📋 獲取原始容器配置...")
	originalContainer, err := dockerMgr.GetContainerConfig(cfg.TargetService)
	if err != nil {
		return fmt.Errorf("獲取容器配置失敗: %w", err)
	}

	// 2. 上傳初始執行檔
	log.Println("📤 上傳初始執行檔...")
	if err := sshClient.UploadFile(cfg.LocalBinary, cfg.RemoteBinaryPath); err != nil {
		return fmt.Errorf("上傳執行檔失敗: %w", err)
	}
	if err := sshClient.CreateScript(fmt.Sprintf("%s\nsh ./entry.sh", cfg.InitialScripts), "/tmp/dev-binaries/init.sh"); err != nil {
		return fmt.Errorf("上傳初始腳本失敗: %w", err)
	}

	// 3. 停止原始容器
	log.Println("🛑 停止原始容器...")
	if err := dockerMgr.StopContainer(cfg.TargetService); err != nil {
		return fmt.Errorf("停止容器失敗: %w", err)
	}

	// 確保退出時恢復原始容器
	defer func() {
		log.Println("🔄 恢復原始容器...")
		if err := dockerMgr.RestoreOriginalContainer(cfg.TargetService); err != nil {
			log.Printf("❌ 恢復原始容器失敗: %v", err)
		} else {
			log.Println("✅ 原始容器已恢復")
		}
	}()

	// 4. 建立開發容器
	log.Println("🔧 建立開發容器...")
	devContainer, err := dockerMgr.CreateDevContainer(originalContainer, cfg)
	if err != nil {
		return fmt.Errorf("建立開發容器失敗: %w", err)
	}

	// 確保退出時清理開發容器
	defer func() {
		log.Println("🧹 清理開發容器...")
		if err := dockerMgr.RemoveDevContainer(devContainer.Name); err != nil {
			log.Printf("❌ 清理開發容器失敗: %v", err)
		} else {
			log.Println("✅ 開發容器已清理")
		}
	}()

	// 5. 啟動開發容器
	log.Println("▶️  啟動開發容器...")
	if err := dockerMgr.StartContainer(devContainer.Name); err != nil {
		return fmt.Errorf("啟動開發容器失敗: %w", err)
	}

	// 6. 建立 SSH Tunnel (用於 Debugger)
	log.Println("🔌 建立 SSH Tunnel...")
	tunnel, err := sshClient.CreateTunnel(cfg.DebuggerPort, cfg.DebuggerPort)
	if err != nil {
		return fmt.Errorf("建立 SSH Tunnel 失敗: %w", err)
	}
	defer tunnel.Close()

	log.Printf("✅ Debugger 可在 localhost:%d 連接", cfg.DebuggerPort)

	// 7. 啟動檔案監控
	log.Println("👀 啟動檔案監控...")
	fileWatcher := watcher.New(cfg.LocalBinary, func(path string) {
		log.Printf("🔄 偵測到檔案更新: %s", path)

		// 上傳新檔案
		log.Println("📤 上傳新執行檔...")
		if err := sshClient.UploadFile(cfg.LocalBinary, cfg.RemoteBinaryPath); err != nil {
			log.Printf("❌ 上傳失敗: %v", err)
			return
		}

		// 重啟容器
		log.Println("🔄 重啟開發容器...")
		if err := dockerMgr.RestartContainer(devContainer.Name); err != nil {
			log.Printf("❌ 重啟失敗: %v", err)
			return
		}

		log.Println("✅ 容器已重啟，新版本已部署")
	})

	if err := fileWatcher.Start(ctx); err != nil {
		return fmt.Errorf("啟動檔案監控失敗: %w", err)
	}

	log.Println("🎉 開發環境已就緒！")
	log.Println("   - 按 Ctrl+C 退出並清理")
	log.Println("   - 修改並編譯二進制檔案會自動部署")

	// 等待退出信號
	<-ctx.Done()

	return nil
}
