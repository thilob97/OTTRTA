package session

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	go_pty "github.com/aymanbagabas/go-pty"
)

const ptyEventBuffer = 128

// PTYSession owns one interactive pseudo-terminal runtime.
type PTYSession interface {
	Start(ctx context.Context, spec PTYSpec) error
	Write([]byte) error
	Resize(cols, rows int) error
	Stop() error
	Events() <-chan PTYEvent
}

// PTYSpec describes the command to start in a pseudo-terminal.
type PTYSpec struct {
	Command string
	Args    []string
	WorkDir string
	Env     map[string]string
	Cols    int
	Rows    int
}

// PTYEvent is emitted by a PTY runtime and consumed by the TUI update loop.
type PTYEvent struct {
	SessionID string
	Data      []byte
	Err       error
}

type goPTYSession struct {
	sessionID string
	pty       go_pty.Pty
	cmd       *go_pty.Cmd
	cancel    context.CancelFunc
	closeOnce sync.Once
	closeErr  error
	events    chan PTYEvent
}

func newPTYSession(sessionID string) PTYSession {
	return &goPTYSession{
		sessionID: sessionID,
		events:    make(chan PTYEvent, ptyEventBuffer),
	}
}

func (p *goPTYSession) Start(ctx context.Context, spec PTYSpec) error {
	if spec.Command == "" {
		return fmt.Errorf("pty session %q has no command", p.sessionID)
	}

	ctx, cancel := context.WithCancel(ctx)
	terminal, err := go_pty.New()
	if err != nil {
		cancel()
		return err
	}
	if spec.Cols > 0 && spec.Rows > 0 {
		_ = terminal.Resize(spec.Cols, spec.Rows)
	}

	cmd := terminal.CommandContext(ctx, spec.Command, spec.Args...)
	cmd.Dir = spec.WorkDir
	if len(spec.Env) > 0 {
		cmd.Env = mergeEnv(spec.Env)
	}

	if err := cmd.Start(); err != nil {
		cancel()
		_ = terminal.Close()
		return err
	}

	p.pty = terminal
	p.cmd = cmd
	p.cancel = cancel
	p.events = make(chan PTYEvent, ptyEventBuffer)
	p.stream()
	return nil
}

func (p *goPTYSession) Write(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if p.pty == nil {
		return fmt.Errorf("pty session %q is not running", p.sessionID)
	}
	_, err := p.pty.Write(data)
	return err
}

func (p *goPTYSession) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	if p.pty == nil {
		return fmt.Errorf("pty session %q is not running", p.sessionID)
	}
	return p.pty.Resize(cols, rows)
}

func (p *goPTYSession) Stop() error {
	if p.cancel != nil {
		p.cancel()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	if p.pty != nil {
		return p.closePTY()
	}
	return nil
}

func (p *goPTYSession) Events() <-chan PTYEvent {
	return p.events
}

func (p *goPTYSession) stream() {
	var reads sync.WaitGroup
	reads.Add(1)
	go func() {
		defer reads.Done()
		buf := make([]byte, 4096)
		for {
			n, err := p.pty.Read(buf)
			if n > 0 {
				data := make([]byte, n)
				copy(data, buf[:n])
				p.events <- PTYEvent{SessionID: p.sessionID, Data: data}
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		err := p.cmd.Wait()
		readsDone := make(chan struct{})
		go func() {
			reads.Wait()
			close(readsDone)
		}()
		select {
		case <-readsDone:
		case <-time.After(100 * time.Millisecond):
			_ = p.closePTY()
			<-readsDone
		}
		p.events <- PTYEvent{SessionID: p.sessionID, Err: err}
		close(p.events)
	}()
}

func (p *goPTYSession) closePTY() error {
	p.closeOnce.Do(func() {
		if p.pty != nil {
			p.closeErr = p.pty.Close()
		}
	})
	return p.closeErr
}

func mergeEnv(overrides map[string]string) []string {
	env := os.Environ()
	for key, value := range overrides {
		env = append(env, key+"="+value)
	}
	return env
}
