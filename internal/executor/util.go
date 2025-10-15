package executor

// noopCloser 是一個空的 closer，用於本地模式的 tunnel
type noopCloser struct{}

func (n *noopCloser) Close() error {
	return nil
}
