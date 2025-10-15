package executor

import (
	"io"
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
