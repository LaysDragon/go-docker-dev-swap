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

	"github.com/LaysDragonB/docker-dev-swap/internal/ssh"
)

type Follower struct {
	ssh           *ssh.Client
	containerName string
	logFile       *os.File
	enableFile    bool
	logFilePath   string
}

func NewFollower(sshClient *ssh.Client, containerName string, logFilePath string) *Follower {
	return &Follower{
		ssh:           sshClient,
		containerName: containerName,
		enableFile:    logFilePath != "",
		logFilePath:   logFilePath,
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
				log.Printf("⚠️  日誌監控中斷: %v", err)
				log.Println("🔄 等待 3 秒後重新連接...")

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
	// 檢查容器是否存在
	checkCmd := fmt.Sprintf("sudo docker ps -q --filter name=^/%s$", f.containerName)
	output, err := f.ssh.Execute(checkCmd)
	if err != nil || output == "" {
		return fmt.Errorf("容器 %s 不存在或未運行", f.containerName)
	}

	// 使用 docker logs -f 持續跟蹤
	// 使用 --tail 50 只顯示最近 50 行，避免歷史日誌過多
	logsCmd := fmt.Sprintf("sudo docker logs -f --tail 50 %s 2>&1", f.containerName)

	// 創建一個 SSH session 用於執行命令
	session, err := f.ssh.CreateSession()
	if err != nil {
		return fmt.Errorf("創建 SSH session 失敗: %w", err)
	}
	defer session.Close()

	// 獲取標準輸出管道
	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("獲取標準輸出失敗: %w", err)
	}

	// 啟動命令
	if err := session.Start(logsCmd); err != nil {
		return fmt.Errorf("啟動日誌監控失敗: %w", err)
	}

	// 創建一個 goroutine 來處理日誌輸出
	errChan := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()

			// 輸出到控制台
			fmt.Println(line)

			// 寫入文件（如果啟用）
			if f.enableFile && f.logFile != nil {
				if _, err := f.logFile.WriteString(line + "\n"); err != nil {
					log.Printf("⚠️  寫入日誌文件失敗: %v", err)
				}
			}
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
		session.Close() // 關閉 session 來中斷命令
		return ctx.Err()
	case err := <-errChan:
		session.Wait()
		return err
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
