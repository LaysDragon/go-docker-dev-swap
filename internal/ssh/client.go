package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/laysdragon/go-docker-dev-swap/internal/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	client *ssh.Client
	config *config.RemoteHost
}

func NewClient(cfg config.RemoteHost) (*Client, error) {
	var authMethods []ssh.AuthMethod

	// 密碼認證
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	// 金鑰認證
	if cfg.KeyFile != "" {
		key, err := os.ReadFile(cfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("讀取金鑰檔案失敗: %w", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("解析私鑰失敗: %w", err)
		}

		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH 連接失敗: %w", err)
	}

	return &Client{
		client: client,
		config: &cfg,
	}, nil
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) CreateSession() (*ssh.Session, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("建立 SSH session 失敗: %w", err)
	}
	return session, nil
}

func (c *Client) CreateScript(script, path string) error {
	if _, err := c.Execute(fmt.Sprintf("echo -e \"%s\" > %s", script, path)); err != nil {
		return fmt.Errorf("建立腳本 %s 失敗: %w", path, err)
	}

	if _, err := c.Execute(fmt.Sprintf("chmod +x %s", path)); err != nil {
		return fmt.Errorf("賦予腳本 %s 執行權限失敗: %w", path, err)
	}
	return nil
}

func (c *Client) Execute(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("建立 SSH session 失敗: %w", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return string(output),fmt.Errorf("執行命令失敗: %w (%s)=>(%s)", err, command,output)
	}

	return string(output), nil
}

func (c *Client) UploadFile(localPath, remotePath string) error {
	sftpClient, err := sftp.NewClient(c.client)
	if err != nil {
		return fmt.Errorf("建立 SFTP 客戶端失敗: %w", err)
	}
	defer sftpClient.Close()

	// 建立遠端目錄
	remoteDir := filepath.Dir(remotePath)
	if err := sftpClient.MkdirAll(remoteDir); err != nil {
		return fmt.Errorf("建立遠端目錄失敗: %w", err)
	}

	// 打開本地檔案
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("打開本地檔案失敗: %w", err)
	}
	defer localFile.Close()

	// 檢查遠端檔案是否存在，如果存在則刪除
	if _, err := sftpClient.Stat(remotePath); err == nil {
		// 檔案存在，先刪除
		if err := sftpClient.Remove(remotePath); err != nil {
			return fmt.Errorf("刪除已存在的遠端檔案失敗: %w", err)
		}
	}

	// 建立遠端檔案
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("建立遠端檔案失敗: %w", err)
	}
	defer remoteFile.Close()

	// 複製檔案內容
	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("上傳檔案失敗: %w", err)
	}

	// 設定執行權限
	if err := sftpClient.Chmod(remotePath, 0755); err != nil {
		return fmt.Errorf("設定檔案權限失敗: %w", err)
	}

	return nil
}

type Tunnel struct {
	listener net.Listener
	client   *ssh.Client
}

func (c *Client) CreateTunnel(localPort, remotePort int) (*Tunnel, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", localPort))
	if err != nil {
		return nil, fmt.Errorf("建立本地監聽失敗: %w", err)
	}

	tunnel := &Tunnel{
		listener: listener,
		client:   c.client,
	}

	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				return
			}
			fmt.Printf("接受本地連接: %s\n", localConn.RemoteAddr().String())

			go func(local net.Conn) {
				defer local.Close()
				defer fmt.Printf("關閉本地連接: %s\n", local.RemoteAddr().String())

				remote, err := c.client.Dial("tcp", fmt.Sprintf("localhost:%d", remotePort))
				if err != nil {
					return
				}
				defer remote.Close()

				// 雙向複製數據
				done := make(chan struct{}, 2)
				go func() {
					io.Copy(remote, local)
					done <- struct{}{}
				}()
				go func() {
					io.Copy(local, remote)
					done <- struct{}{}
				}()
				<-done
			}(localConn)
		}
	}()

	return tunnel, nil
}

func (t *Tunnel) Close() error {
	return t.listener.Close()
}
