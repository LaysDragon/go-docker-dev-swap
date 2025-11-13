package logger

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
	"github.com/laysdragon/go-docker-dev-swap/internal/executor"
	"github.com/laysdragon/go-docker-dev-swap/internal/sudo"
)

type Follower struct {
	executor      executor.Executor
	containerName string
	logFile       *os.File
	enableFile    bool
	logFilePath   string
	sudoWrapper   *sudo.Wrapper
	dockerCmd     string
}

func NewFollower(exec executor.Executor, containerName string, cfg *config.Config) *Follower {
	dockerCmd := cfg.DockerCommand
	if dockerCmd == "" {
		dockerCmd = "docker"
	}
	
	return &Follower{
		executor:      exec,
		containerName: containerName,
		enableFile:    cfg.LogFile != "",
		logFilePath:   cfg.LogFile,
		sudoWrapper:   sudo.NewWrapper(cfg.UseSudo, cfg.SudoPassword),
		dockerCmd:     dockerCmd,
	}
}

// Start 開始監控容器日誌
func (f *Follower) Start(ctx context.Context) error {
	// 如果需要寫入文件，創建日誌文件
	if f.enableFile {
		if err := f.openLogFile(); err != nil {
			return fmt.Errorf("創建日誌文件失敗: %w", err)
		}
		defer f.closeLogFile()
	}

	// 持續監控日誌，應對容器重啟
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := f.followLogs(ctx); err != nil {
				log.Printf("日誌監控中斷: %v", err)
				log.Println("等待 3 秒後重新連接...")

				// 等待一段時間後重試
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(3 * time.Second):
					continue
				}
			}
		}
	}
}

// followLogs 持續跟蹤容器日誌
func (f *Follower) followLogs(ctx context.Context) error {
	// 構建 docker 命令
	checkCmd := f.buildDockerCmd("ps", "-q", fmt.Sprintf("--filter name=^/%s$", f.containerName))
	output, err := f.executor.Execute(checkCmd)
	if err != nil || output == "" {
		return fmt.Errorf("容器 %s 不存在或未運行", f.containerName)
	}

	// 使用 docker logs -f 持續跟蹤
	// 使用 --tail 50 只顯示最近 50 行，避免歷史日誌過多
	logsCmd := f.buildDockerCmd("logs", "-f", "--tail", "50", f.containerName, "2>&1")

	// 創建統一的 session（本地或遠端）
	session, err := f.executor.CreateSession()
	if err != nil {
		return fmt.Errorf("創建 session 失敗: %w", err)
	}
	defer session.Close()

	// 啟動命令（內部會自動處理 StdoutPipe 的獲取時機）
	if err := session.Start(logsCmd); err != nil {
		return fmt.Errorf("啟動日誌監控失敗: %w", err)
	}

	// Start 之後獲取輸出流（此時兩種實現都已準備好）
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("獲取標準輸出失敗: %w", err)
	}

	return f.streamLogs(ctx, stdout, session)
}

// streamLogs 統一處理日誌流
func (f *Follower) streamLogs(ctx context.Context, stdout io.Reader, session interface {
	Close() error
	Wait() error
}) error {
	// 創建一個 goroutine 來處理日誌輸出
	errChan := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			f.processLogLine(line)
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			errChan <- err
		} else {
			errChan <- nil
		}
	}()

	// 等待命令結束或上下文取消
	select {
	case <-ctx.Done():
		session.Close() // 關閉 session
		return ctx.Err()
	case err := <-errChan:
		session.Wait() // 等待命令完成
		return err
	}
}

// processLogLine 處理單行日誌
func (f *Follower) processLogLine(line string) {
	// 輸出到控制台
	fmt.Println(line)

	// 寫入文件（如果啟用）
	if f.enableFile && f.logFile != nil {
		if _, err := f.logFile.WriteString(line + "\n"); err != nil {
			log.Printf("寫入日誌文件失敗: %v", err)
		}
	}
}

// openLogFile 打開日誌文件
func (f *Follower) openLogFile() error {
	// 確保目錄存在
	dir := filepath.Dir(f.logFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("創建日誌目錄失敗: %w", err)
	}

	// 打開文件（追加模式）
	file, err := os.OpenFile(f.logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打開日誌文件失敗: %w", err)
	}

	f.logFile = file

	// 寫入分隔線和時間戳
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	separator := fmt.Sprintf("\n=== 日誌開始 [%s] ===\n", timestamp)
	if _, err := f.logFile.WriteString(separator); err != nil {
		return fmt.Errorf("寫入分隔線失敗: %w", err)
	}

	return nil
}

// buildDockerCmd 構建 docker 命令並根據配置包裝 sudo
func (f *Follower) buildDockerCmd(args ...string) string {
	parts := append([]string{f.dockerCmd}, args...)
	return f.sudoWrapper.WrapMultiple(parts...)
}

// closeLogFile 關閉日誌文件
func (f *Follower) closeLogFile() {
	if f.logFile != nil {
		// 寫入結束標記
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		separator := fmt.Sprintf("=== 日誌結束 [%s] ===\n\n", timestamp)
		f.logFile.WriteString(separator)

		f.logFile.Close()
		f.logFile = nil
	}
}
