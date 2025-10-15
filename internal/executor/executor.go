package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
	"github.com/laysdragon/go-docker-dev-swap/internal/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// Session 定義了流式命令執行的統一接口
type Session interface {
	// StdoutPipe 返回標準輸出管道
	StdoutPipe() (io.Reader, error)
	
	// Start 啟動命令
	Start(command string) error
	
	// Wait 等待命令完成
	Wait() error
	
	// Close 關閉 session
	Close() error
}

// Executor 定義了執行操作的抽象接口
type Executor interface {
	// Execute 執行 shell 指令
	Execute(command string) (string, error)
	
	// CreateSession 建立一個流式執行 session
	CreateSession() (Session, error)
	
	// UploadFile 上傳/複製檔案
	UploadFile(localPath, remotePath string) error
	
	// CreateScript 建立腳本檔案
	CreateScript(script, path string) error
	
	// CreateTunnel 建立 SSH tunnel (僅遠端模式)
	CreateTunnel(localPort, remotePort int) (TunnelCloser, error)
	
	// Close 關閉連接
	Close() error
	
	// IsRemote 判斷是否為遠端模式
	IsRemote() bool
}

// TunnelCloser 定義了 tunnel 的關閉接口
type TunnelCloser interface {
	Close() error
}

// NewExecutor 根據配置建立對應的 Executor
func NewExecutor(cfg *config.Config) (Executor, error) {
	if cfg.Mode == "local" {
		return NewLocalExecutor(cfg)
	}
	return NewRemoteExecutor(cfg)
}

// LocalExecutor 本地執行器
type LocalExecutor struct {
	config *config.Config
}

func NewLocalExecutor(cfg *config.Config) (*LocalExecutor, error) {
	return &LocalExecutor{
		config: cfg,
	}, nil
}

func (e *LocalExecutor) Execute(command string) (string, error) {
	cmd := exec.Command("bash", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("執行命令失敗: %w (%s)", err, output)
	}
	return string(output), nil
}

func (e *LocalExecutor) CreateSession() (Session, error) {
	return &LocalSession{}, nil
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

// RemoteExecutor 遠端執行器
type RemoteExecutor struct {
	sshClient *ssh.Client
	config    *config.Config
}

func NewRemoteExecutor(cfg *config.Config) (*RemoteExecutor, error) {
	sshClient, err := ssh.NewClient(cfg.RemoteHost)
	if err != nil {
		return nil, fmt.Errorf("SSH 連接失敗: %w", err)
	}

	return &RemoteExecutor{
		sshClient: sshClient,
		config:    cfg,
	}, nil
}

func (e *RemoteExecutor) Execute(command string) (string, error) {
	return e.sshClient.Execute(command)
}

func (e *RemoteExecutor) CreateSession() (Session, error) {
	sshSession, err := e.sshClient.CreateSession()
	if err != nil {
		return nil, err
	}
	return &RemoteSession{session: sshSession}, nil
}

func (e *RemoteExecutor) UploadFile(localPath, remotePath string) error {
	return e.sshClient.UploadFile(localPath, remotePath)
}

func (e *RemoteExecutor) CreateScript(script, path string) error {
	return e.sshClient.CreateScript(script, path)
}

func (e *RemoteExecutor) CreateTunnel(localPort, remotePort int) (TunnelCloser, error) {
	return e.sshClient.CreateTunnel(localPort, remotePort)
}

func (e *RemoteExecutor) Close() error {
	return e.sshClient.Close()
}

func (e *RemoteExecutor) IsRemote() bool {
	return true
}

// GetSSHClient 返回底層的 SSH client (僅用於需要直接訪問的場景)
func (e *RemoteExecutor) GetSSHClient() *ssh.Client {
	return e.sshClient
}

// noopCloser 是一個空的 closer，用於本地模式的 tunnel
type noopCloser struct{}

func (n *noopCloser) Close() error {
	return nil
}

// LocalSession 本地命令執行 session
type LocalSession struct {
	cmd    *exec.Cmd
	stdout io.ReadCloser
}

func (s *LocalSession) StdoutPipe() (io.Reader, error) {
	if s.stdout != nil {
		return s.stdout, nil
	}
	return nil, fmt.Errorf("必須先調用 Start")
}

func (s *LocalSession) Start(command string) error {
	s.cmd = exec.Command("bash", "-c", command)
	
	stdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("獲取標準輸出失敗: %w", err)
	}
	s.stdout = stdout
	
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("啟動命令失敗: %w", err)
	}
	
	return nil
}

func (s *LocalSession) Wait() error {
	if s.cmd == nil {
		return nil
	}
	return s.cmd.Wait()
}

func (s *LocalSession) Close() error {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Kill()
	}
	return nil
}

// RemoteSession SSH 遠端執行 session
type RemoteSession struct {
	session *gossh.Session
	stdout  io.Reader
}

func (s *RemoteSession) StdoutPipe() (io.Reader, error) {
	if s.stdout != nil {
		return s.stdout, nil
	}
	return nil, fmt.Errorf("必須先調用 Start")
}

func (s *RemoteSession) Start(command string) error {
	// SSH 必須在 Start 之前設置 StdoutPipe
	if s.stdout == nil {
		stdout, err := s.session.StdoutPipe()
		if err != nil {
			return fmt.Errorf("獲取標準輸出失敗: %w", err)
		}
		s.stdout = stdout
	}
	
	if err := s.session.Start(command); err != nil {
		return fmt.Errorf("啟動命令失敗: %w", err)
	}
	return nil
}

func (s *RemoteSession) Wait() error {
	if s.session == nil {
		return nil
	}
	return s.session.Wait()
}

func (s *RemoteSession) Close() error {
	if s.session != nil {
		return s.session.Close()
	}
	return nil
}
