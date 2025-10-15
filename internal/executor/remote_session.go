package executor

import (
	"fmt"
	"io"

	gossh "golang.org/x/crypto/ssh"
)

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
