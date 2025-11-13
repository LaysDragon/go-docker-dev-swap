package executor

import (
	"fmt"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
	"github.com/laysdragon/go-docker-dev-swap/internal/ssh"
	"github.com/laysdragon/go-docker-dev-swap/internal/sudo"
)

// RemoteExecutor 遠端執行器
type RemoteExecutor struct {
	sshClient   *ssh.Client
	config      *config.Config
	sudoWrapper *sudo.Wrapper
}

// NewRemoteExecutor 創建遠端執行器
func NewRemoteExecutor(cfg *config.Config) (*RemoteExecutor, error) {
	sshClient, err := ssh.NewClient(cfg.RemoteHost)
	if err != nil {
		return nil, fmt.Errorf("SSH 連接失敗: %w", err)
	}

	return &RemoteExecutor{
		sshClient:   sshClient,
		config:      cfg,
		sudoWrapper: sudo.NewWrapper(cfg.UseSudo, cfg.SudoPassword),
	}, nil
}

func (e *RemoteExecutor) Execute(command string) (string, error) {
	// 使用 sudo wrapper 包装命令
	wrappedCmd := e.sudoWrapper.Wrap(command)
	return e.sshClient.Execute(wrappedCmd)
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
