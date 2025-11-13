package executor

import (
	"fmt"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
)

// RemoteExecutor 遠端執行器
type RemoteExecutor struct {
	sshClient   *SSHClient
	config      *config.RuntimeConfig
	sudoWrapper *SudoWrapper
}

// NewRemoteExecutor 創建遠端執行器
func NewRemoteExecutor(rc *config.RuntimeConfig) (*RemoteExecutor, error) {
	// 從 RuntimeConfig 的 Host 創建 RemoteHost
	remoteHost := config.RemoteHost{
		Host:     rc.Host.Host,
		Port:     rc.Host.Port,
		User:     rc.Host.User,
		Password: rc.Host.Password,
		KeyFile:  rc.Host.KeyFile,
	}
	
	sshClient, err := NewSSHClient(remoteHost)
	if err != nil {
		return nil, fmt.Errorf("SSH 連接失敗: %w", err)
	}

	return &RemoteExecutor{
		sshClient:   sshClient,
		config:      rc,
		sudoWrapper: NewSudoWrapper(rc.UseSudo, rc.SudoPassword),
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
func (e *RemoteExecutor) GetSSHClient() *SSHClient {
	return e.sshClient
}
