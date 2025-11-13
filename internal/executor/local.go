package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
)

// LocalExecutor 本地執行器
type LocalExecutor struct {
	config      *config.RuntimeConfig
	sudoWrapper *SudoWrapper
}

// NewLocalExecutor 創建本地執行器
func NewLocalExecutor(rc *config.RuntimeConfig) (*LocalExecutor, error) {
	return &LocalExecutor{
		config:      rc,
		sudoWrapper: NewSudoWrapper(rc.UseSudo, rc.SudoPassword),
	}, nil
}

func (e *LocalExecutor) Execute(command string) (string, error) {
	// 使用 sudo wrapper 包装命令
	wrappedCmd := e.sudoWrapper.Wrap(command)
	
	cmd := exec.Command("bash", "-c", wrappedCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("執行命令失敗: %w (%s)=>(%s)", err, wrappedCmd, output)
	}
	return string(output), nil
}

func (e *LocalExecutor) CreateSession() (Session, error) {
	return &LocalSession{
		sudoWrapper: e.sudoWrapper,
	}, nil
}

func (e *LocalExecutor) UploadFile(localPath, destPath string) error {
	// 本地模式直接複製檔案
	sourceFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打開源檔案失敗: %w", err)
	}
	defer sourceFile.Close()

	// 建立目標目錄
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("建立目標目錄失敗: %w", err)
	}

	// 如果目標檔案存在，先刪除
	if _, err := os.Stat(destPath); err == nil {
		if err := os.Remove(destPath); err != nil {
			return fmt.Errorf("刪除已存在的檔案失敗: %w", err)
		}
	}

	// 建立目標檔案
	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("建立目標檔案失敗: %w", err)
	}
	defer destFile.Close()

	// 複製檔案內容
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("複製檔案失敗: %w", err)
	}

	// 設定執行權限
	if err := os.Chmod(destPath, 0755); err != nil {
		return fmt.Errorf("設定檔案權限失敗: %w", err)
	}

	return nil
}

func (e *LocalExecutor) CreateScript(script, path string) error {
	// 建立目錄
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("建立目錄失敗: %w", err)
	}

	// 寫入腳本
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		return fmt.Errorf("寫入腳本失敗: %w", err)
	}

	return nil
}

func (e *LocalExecutor) CreateTunnel(localPort, remotePort int) (TunnelCloser, error) {
	// 本地模式不需要 tunnel
	return &noopCloser{}, nil
}

func (e *LocalExecutor) Close() error {
	return nil
}

func (e *LocalExecutor) IsRemote() bool {
	return false
}
