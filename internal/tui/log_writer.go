package tui

import (
	"bytes"
	"sync"
)

// logWriter converts the standard library logger output into UI events line by line.
type logWriter struct {
	manager *Manager
	mu      sync.Mutex
	buf     bytes.Buffer
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	total := len(p)
	data := p

	for len(data) > 0 {
		idx := bytes.IndexByte(data, '\n')
		if idx == -1 {
			w.buf.Write(data)
			break
		}

		w.buf.Write(data[:idx])
		w.flush()
		data = data[idx+1:]
	}

	return total, nil
}

func (w *logWriter) flush() {
	if w.buf.Len() == 0 {
		return
	}
	line := w.buf.String()
	w.buf.Reset()
	w.manager.PublishWorkLog(line)
}
