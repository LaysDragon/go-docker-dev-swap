package tui

import (
	"context"
	"errors"
	"io"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// Options controls the behavior of the Bubble Tea manager.
type Options struct {
	InitialDebuggerEnabled bool
	MaxLines               int
}

// Manager wires the application logs and shortcut actions into a Bubble Tea program.
type Manager struct {
	opts Options

	workWriter *logWriter

	workQueue      chan string
	containerQueue chan string
	actionChan     chan Action

	programMu sync.RWMutex
	program   *tea.Program
}

// NewManager creates a new Manager instance.
func NewManager(opts Options) *Manager {
	if opts.MaxLines <= 0 {
		opts.MaxLines = 400
	}

	m := &Manager{
		opts:           opts,
		workQueue:      make(chan string, 1024),
		containerQueue: make(chan string, 1024),
		actionChan:     make(chan Action, 16),
	}
	m.workWriter = &logWriter{manager: m}
	return m
}

// WorkLogWriter returns an io.Writer that can be plugged into log.SetOutput.
func (m *Manager) WorkLogWriter() io.Writer {
	return m.workWriter
}

// PublishWorkLog appends a work-log line to the UI.
func (m *Manager) PublishWorkLog(line string) {
	m.enqueue(m.workQueue, line)
}

// PublishContainerLog appends a container log line to the UI.
func (m *Manager) PublishContainerLog(line string) {
	m.enqueue(m.containerQueue, line)
}

func (m *Manager) enqueue(queue chan string, line string) {
	if line == "" {
		return
	}
	select {
	case queue <- line:
	default:
		// Drop when the queue is full to keep the UI responsive.
	}
}

// Actions exposes the shortcut action channel for the main workflow.
func (m *Manager) Actions() <-chan Action {
	return m.actionChan
}

// UpdateDebuggerState refreshes the UI state to reflect the actual debugger status.
func (m *Manager) UpdateDebuggerState(enabled bool) {
	m.send(debuggerStateMsg{enabled: enabled})
}

// Start launches the Bubble Tea program and blocks until it terminates or the context is cancelled.
func (m *Manager) Start(ctx context.Context) error {
	mdl := newModel(modelOptions{
		maxLines:               m.opts.MaxLines,
		initialDebuggerEnabled: m.opts.InitialDebuggerEnabled,
		actionChan:             m.actionChan,
	})

	program := tea.NewProgram(mdl, tea.WithAltScreen())
	m.programMu.Lock()
	m.program = program
	m.programMu.Unlock()

	go m.forward(ctx, m.workQueue, func(line string) tea.Msg { return workLogMsg(line) })
	go m.forward(ctx, m.containerQueue, func(line string) tea.Msg { return containerLogMsg(line) })

	go func() {
		<-ctx.Done()
		m.programMu.RLock()
		defer m.programMu.RUnlock()
		if m.program != nil {
			m.program.Quit()
		}
	}()

	if _, err := program.Run(); err != nil {
		if errors.Is(err, tea.ErrProgramKilled) || errors.Is(err, context.Canceled) {
			return nil
		}
		if ctx.Err() != nil && errors.Is(err, ctx.Err()) {
			return nil
		}
		return err
	}

	return nil
}

func (m *Manager) forward(ctx context.Context, queue <-chan string, builder func(string) tea.Msg) {
	for {
		select {
		case <-ctx.Done():
			return
		case line := <-queue:
			if line == "" {
				continue
			}
			m.send(builder(line))
		}
	}
}

func (m *Manager) send(msg tea.Msg) {
	m.programMu.RLock()
	defer m.programMu.RUnlock()
	if m.program != nil {
		m.program.Send(msg)
	}
}
