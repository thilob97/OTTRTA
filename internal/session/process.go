package session

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/thilob97/ottrta/internal/event"
)

const (
	processEventBuffer       = 128
	maxProcessLogLineBytes   = 1024 * 1024
	initialProcessBufferSize = 64 * 1024
)

// ProcessRuntime owns the OS process handles for one process session.
type ProcessRuntime struct {
	cmd    *exec.Cmd
	cancel context.CancelFunc
	events chan event.ProcessMsg
}

// StartProcess starts the command configured by s and streams stdout/stderr as
// line-oriented process events.
func StartProcess(ctx context.Context, s Session) (*ProcessRuntime, error) {
	if s.Command == "" {
		return nil, fmt.Errorf("session %q has no command", s.ID)
	}

	ctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(ctx, s.Command, s.Args...)
	if s.WorkDir != "" {
		cmd.Dir = s.WorkDir
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, err
	}

	runtime := &ProcessRuntime{
		cmd:    cmd,
		cancel: cancel,
		events: make(chan event.ProcessMsg, processEventBuffer),
	}
	runtime.stream(s.ID, stdout, stderr)
	return runtime, nil
}

func (r *ProcessRuntime) Events() <-chan event.ProcessMsg {
	return r.events
}

func (r *ProcessRuntime) Stop() {
	r.cancel()
	if r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
	}
}

func (r *ProcessRuntime) CommandLine() string {
	if len(r.cmd.Args) == 0 {
		return ""
	}
	return commandLine(r.cmd.Args[0], r.cmd.Args[1:])
}

func (r *ProcessRuntime) stream(sessionID string, stdout io.Reader, stderr io.Reader) {
	var scans sync.WaitGroup
	scans.Add(2)
	go scanLines(&scans, r.events, sessionID, stdout)
	go scanLines(&scans, r.events, sessionID, stderr)

	go func() {
		err := r.cmd.Wait()
		scans.Wait()
		r.events <- event.SessionExitedMsg{SessionID: sessionID, Err: err}
		close(r.events)
	}()
}

func scanLines(wg *sync.WaitGroup, events chan<- event.ProcessMsg, sessionID string, reader io.Reader) {
	defer wg.Done()

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, initialProcessBufferSize), maxProcessLogLineBytes)
	for scanner.Scan() {
		events <- event.SessionLogMsg{SessionID: sessionID, Line: scanner.Text()}
	}
	if err := scanner.Err(); err != nil {
		events <- event.SessionLogMsg{SessionID: sessionID, Line: fmt.Sprintf("[system] log stream error: %v", err)}
	}
}
