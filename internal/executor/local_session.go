package executor

import (
	"fmt"
	"io"
	"os/exec"
)

// LocalSession 本地命令執行 session
type LocalSession struct {
	cmd         *exec.Cmd
	stdout      io.ReadCloser
	sudoWrapper *SudoWrapper
}

func (s *LocalSession) StdoutPipe() (io.Reader, error) {
	if s.stdout != nil {
		return s.stdout, nil
	}
	return nil, fmt.Errorf("必須先調用 Start")
}

func (s *LocalSession) Start(command string) error {
	// 使用 sudo wrapper 包裝命令
	wrappedCmd := command
	if s.sudoWrapper != nil {
		wrappedCmd = s.sudoWrapper.Wrap(command)
	}
	
	s.cmd = exec.Command("bash", "-c", wrappedCmd)
	
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
