package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/handyfun97/ottrta/internal/event"
	"github.com/handyfun97/ottrta/internal/session"
	"github.com/handyfun97/ottrta/internal/task"
)

func testModel() Model {
	sessionMgr := session.NewManager([]session.Session{
		{ID: "proc-1", Name: "proc-1", Kind: session.SessionKindProcess, Status: session.StatusStopped, Command: "go", Args: []string{"version"}},
		{ID: "shell-1", Name: "shell-1", Kind: session.SessionKindPTY, Status: session.StatusStopped, Command: session.DefaultShellCommand()},
		{ID: "omp-1", Name: "omp-1", Kind: session.SessionKindAgent, AgentKind: session.AgentKindOmp, Status: session.StatusStopped, Command: "omp"},
	})
	taskMgr := task.NewManager(&sessionMgr)
	return newBaseModel(sessionMgr, taskMgr, "")
}

func TestKeyNavigationIsClamped(t *testing.T) {
	m := testModel()

	m = updateForTest(t, m, keyRune('k'))
	if m.selectedSession != 0 {
		t.Fatalf("selectedSession = %d, want 0", m.selectedSession)
	}

	m = updateForTest(t, m, keyRune('j'))
	m = updateForTest(t, m, keyRune('j'))
	m = updateForTest(t, m, keyRune('j'))
	if m.selectedSession != 2 {
		t.Fatalf("selectedSession = %d, want 2", m.selectedSession)
	}
}

func TestSpaceTogglesSelectedProcessSession(t *testing.T) {
	m := testModel()
	m.selectedSession = 0
	updated, cmd := m.Update(keyRune(' '))
	if cmd == nil {
		t.Fatal("space on stopped process did not return poll command")
	}
	m = modelFromUpdate(t, updated)
	s, _ := m.manager.SessionByID("proc-1")
	if s.Status != session.StatusRunning {
		t.Fatalf("status = %s, want running", s.Status)
	}
}

func TestSpaceStartsProcessAndHandlesEvents(t *testing.T) {
	m := testModel()
	m.selectedSession = 0

	updated, cmd := m.Update(keyRune(' '))
	if cmd == nil {
		t.Fatal("space on process session did not return a process poll command")
	}
	m = modelFromUpdate(t, updated)
	s, _ := m.manager.SessionByID("proc-1")
	if s.Status != session.StatusRunning {
		t.Fatalf("status after start = %s, want running", s.Status)
	}

	deadline := time.After(5 * time.Second)
	for s.Status != session.StatusStopped {
		select {
		case <-deadline:
			t.Fatalf("timed out waiting for process exit: %v", s.Logs)
		default:
		}
		msg := cmd()
		updated, cmd = m.Update(msg)
		m = modelFromUpdate(t, updated)
		s, _ = m.manager.SessionByID("proc-1")
		if cmd == nil && s.Status != session.StatusStopped {
			t.Fatalf("poll command stopped before process exit: %v", s.Logs)
		}
	}

	logs := strings.Join(s.Logs, "\n")
	if !strings.Contains(logs, "go version") {
		t.Fatalf("logs do not contain go version output: %v", s.Logs)
	}
	if !strings.Contains(logs, "[system] exited with code 0") {
		t.Fatalf("logs do not contain exit code: %v", s.Logs)
	}
}

func TestSpaceStopsRunningProcessSession(t *testing.T) {
	m := testModel()
	m.selectedSession = 0

	updated, _ := m.Update(keyRune(' '))
	m = modelFromUpdate(t, updated)
	updated, _ = m.Update(keyRune(' '))
	m = modelFromUpdate(t, updated)

	s, _ := m.manager.SessionByID("proc-1")
	if s.Status != session.StatusStopped {
		t.Fatalf("status after stop = %s, want stopped", s.Status)
	}
	if got := strings.Join(s.Logs, "\n"); !strings.Contains(got, "[system] stopped") {
		t.Fatalf("logs do not contain stopped message: %v", s.Logs)
	}
}

func TestFocusKeys(t *testing.T) {
	m := testModel()

	// Starte bei focusSessions (default)
	if m.focus != focusSessions {
		t.Fatalf("initial focus = %d, want sessions", m.focus)
	}

	// Enter wechselt zu focusLogs
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.focus != focusLogs {
		t.Fatalf("focus = %d, want logs after enter", m.focus)
	}

	// esc wechselt zurück zu focusSessions
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.focus != focusSessions {
		t.Fatalf("focus = %d, want sessions after esc", m.focus)
	}
}

func TestRenameModeEditsCommitsRendersAndCancels(t *testing.T) {
	m := testModel()

	m = updateForTest(t, m, keyRune('r'))
	if m.mode != UIModeRename || m.renameSessionID != "proc-1" || m.renameInput != "proc-1" {
		t.Fatalf("rename start = mode %s id %q input %q", m.mode, m.renameSessionID, m.renameInput)
	}
	if footer := m.renderFooter(); !strings.Contains(footer, "RENAME | enter save") {
		t.Fatalf("rename footer = %q", footer)
	}
	if view := m.View(); !strings.Contains(view, "Rename session") || !strings.Contains(view, "proc-1") {
		t.Fatalf("rename modal missing from view:\n%s", view)
	}

	m = updateForTest(t, m, keyRune('x'))
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	s, _ := m.manager.SessionByID("proc-1")
	if m.mode != UIModeMonitor || m.renameSessionID != "" || m.renameInput != "" || s.Name != "proc-1" {
		t.Fatalf("rename cancel left mode/id/input/name = %s/%q/%q/%q", m.mode, m.renameSessionID, m.renameInput, s.Name)
	}

	m = updateForTest(t, m, keyRune('r'))
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	m = updateForTest(t, m, keyRune('x'))
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	s, _ = m.manager.SessionByID("proc-1")
	if m.mode != UIModeMonitor || s.Name != "proc-x" || s.ID != "proc-1" {
		t.Fatalf("rename commit mode/name/id = %s/%q/%q", m.mode, s.Name, s.ID)
	}
	view := m.View()
	if !strings.Contains(view, "proc-x") {
		t.Fatalf("view does not contain renamed session:\n%s", view)
	}
}

func TestNewAgentModeCreatesAgentWithWorkDirAndCancels(t *testing.T) {
	m := testModel()

	m = updateForTest(t, m, keyRune('n'))
	if m.mode != UIModeNewAgent || m.newAgentWorkDirInput != "" {
		t.Fatalf("new agent start = mode %s input %q", m.mode, m.newAgentWorkDirInput)
	}
	if footer := m.renderFooter(); !strings.Contains(footer, "NEW AGENT | tab complete") {
		t.Fatalf("new agent footer = %q", footer)
	}
	if view := m.View(); !strings.Contains(view, "New OMP agent") || !strings.Contains(view, "Working directory") {
		t.Fatalf("new agent modal missing from view:\n%s", view)
	}
	m = updateForTest(t, m, keyRune('x'))
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != UIModeMonitor || m.manager.Count() != 3 {
		t.Fatalf("new agent cancel = mode %s count %d", m.mode, m.manager.Count())
	}

	m = updateForTest(t, m, keyRune('n'))
	for _, r := range "C:/work" {
		m = updateForTest(t, m, keyRune(r))
	}
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeySpace})
	for _, r := range "dir" {
		m = updateForTest(t, m, keyRune(r))
	}
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})

	if m.mode != UIModeMonitor || m.manager.Count() != 4 || m.selectedSession != 3 {
		t.Fatalf("new agent commit = mode %s count %d selected %d", m.mode, m.manager.Count(), m.selectedSession)
	}
	s, _ := m.manager.SessionByID("omp-2")
	if s == nil || s.WorkDir != "C:/work dir" || s.Command != "omp" || s.AgentKind != session.AgentKindOmp {
		t.Fatalf("new agent session mismatch: %+v", s)
	}
	if s.Name == s.ID || !strings.Contains(s.Name, " ") {
		t.Fatalf("new agent display name = %q, want random imp name distinct from id %q", s.Name, s.ID)
	}
	if got := strings.Join(s.Logs, "\n"); !strings.Contains(got, "cwd: C:/work dir") {
		t.Fatalf("new agent log does not include cwd: %v", s.Logs)
	}
	// Note: View may truncate long log lines, so we only check that the session logs contain the cwd
}

func TestNewAgentWorkDirTabCompletion(t *testing.T) {
	root := t.TempDir()
	alpha := filepath.Join(root, "alpha")
	alpine := filepath.Join(root, "alpine")
	beta := filepath.Join(root, "beta")
	for _, dir := range []string{alpha, alpine, beta} {
		if err := os.Mkdir(dir, 0o755); err != nil {
			t.Fatalf("Mkdir(%q) returned error: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "alphabet.txt"), []byte("file"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	completed, hint := completeDirectoryPath(filepath.Join(root, "al"))
	if completed != filepath.Join(root, "alp") {
		t.Fatalf("multi completion = %q, want common prefix %q", completed, filepath.Join(root, "alp"))
	}
	if !strings.Contains(hint, "alpha") || !strings.Contains(hint, "alpine") || strings.Contains(hint, "alphabet.txt") {
		t.Fatalf("multi completion hint = %q", hint)
	}

	m := testModel()
	m = updateForTest(t, m, keyRune('n'))
	m.newAgentWorkDirInput = filepath.Join(root, "alph")
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyTab})
	wantCompleted := alpha + string(os.PathSeparator)
	if m.newAgentWorkDirInput != wantCompleted {
		t.Fatalf("tab completed input = %q, want %q", m.newAgentWorkDirInput, wantCompleted)
	}
	if !strings.Contains(m.newAgentCompletionHint, "completed alpha") {
		t.Fatalf("completion hint = %q", m.newAgentCompletionHint)
	}
	if view := m.View(); !strings.Contains(view, "completed alpha") || !strings.Contains(view, "tab complete") {
		t.Fatalf("completion hint missing from modal:\n%s", view)
	}
}

func TestNewModelWithStorePathLoadsPersistedSessionsWithoutDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	if err := session.SaveSessions(path, []session.Session{{
		ID:        "saved-1",
		Name:      "Saved One",
		Kind:      session.SessionKindAgent,
		Status:    session.StatusRunning,
		Command:   "omp",
		WorkDir:   "C:/saved",
		AgentKind: session.AgentKindOmp,
		Logs:      []string{"not persisted"},
	}}); err != nil {
		t.Fatalf("SaveSessions returned error: %v", err)
	}

	m := newModelWithStorePath(path)
	if m.storePath != path {
		t.Fatalf("storePath = %q, want %q", m.storePath, path)
	}
	if m.manager.Count() != 1 {
		t.Fatalf("session count = %d, want 1 loaded session", m.manager.Count())
	}
	if len(m.taskManager.ListTasks()) != 0 {
		t.Fatalf("loaded model created default tasks: %v", m.taskManager.ListTasks())
	}
	s, _ := m.manager.SessionByID("saved-1")
	if s == nil || s.Name != "Saved One" || s.WorkDir != "C:/saved" || s.Status != session.StatusStopped || len(s.Logs) != 0 {
		t.Fatalf("loaded session mismatch: %+v", s)
	}
}

func TestNewModelWithStorePathFallsBackToDefaults(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing", "sessions.json")
	m := newModelWithStorePath(missingPath)
	if m.manager.Count() != 3 {
		t.Fatalf("missing store session count = %d, want default demo sessions", m.manager.Count())
	}
	if len(m.taskManager.ListTasks()) != 1 {
		t.Fatalf("missing store tasks = %d, want default demo task", len(m.taskManager.ListTasks()))
	}

	corruptPath := filepath.Join(t.TempDir(), "sessions.json")
	if err := os.WriteFile(corruptPath, []byte("{"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	m = newModelWithStorePath(corruptPath)
	if m.manager.Count() != 3 {
		t.Fatalf("corrupt store session count = %d, want default demo sessions", m.manager.Count())
	}
	s, ok := m.manager.Session(0)
	if !ok || !strings.Contains(strings.Join(s.Logs, "\n"), "[system] failed to load sessions:") {
		t.Fatalf("corrupt store did not surface load error: %+v", s)
	}
}

func TestQuitKeyReturnsCommand(t *testing.T) {
	m := testModel()

	_, cmd := m.Update(keyRune('q'))
	if cmd == nil {
		t.Fatal("q did not return a quit command")
	}
}

func TestAnimationTickAdvancesFrame(t *testing.T) {
	m := testModel()
	updated, cmd := m.Update(animationTickMsg{})
	if cmd == nil {
		t.Fatal("animation tick did not schedule next tick")
	}
	m = modelFromUpdate(t, updated)
	if m.animationFrame != 1 {
		t.Fatalf("animationFrame = %d, want 1", m.animationFrame)
	}
}

func TestImpAvatarLayoutAndColor(t *testing.T) {
	for avatarIndex, avatar := range impArts {
		width := lipgloss.Width(avatar[0])
		for lineIndex, line := range avatar[1:] {
			if got := lipgloss.Width(line); got != width {
				t.Fatalf("avatar %d line %d width = %d, want %d: %q", avatarIndex, lineIndex+1, got, width, line)
			}
		}
	}
	if len(impColors) < len(impArts) {
		t.Fatalf("imp color count = %d, want at least avatar count %d", len(impColors), len(impArts))
	}
	if art := impArt("left", session.StatusStopped, 0); !strings.Contains(art, "\x1b[") {
		t.Fatalf("imp art is not colorized: %q", art)
	}
}

func TestImpAvatarsHaveStableVariety(t *testing.T) {
	if len(impArts) < 10 {
		t.Fatalf("imp avatar count = %d, want at least 10", len(impArts))
	}
	first := impArt("left", session.StatusStopped, 0)
	if first != impArt("left", session.StatusStopped, 0) {
		t.Fatal("imp avatar selection is not stable for the same session id")
	}
	running0 := impArt("left", session.StatusRunning, 0)
	running1 := impArt("left", session.StatusRunning, 1)
	if running0 == running1 || !strings.Contains(running0, "slop") || !strings.Contains(running1, "slop") {
		t.Fatalf("running imp art is not animated: %q / %q", running0, running1)
	}
	if stopped := impArt("left", session.StatusStopped, 0); !strings.Contains(stopped, "zZzZ") {
		t.Fatalf("stopped imp art does not show sleep marker: %q", stopped)
	}
	seen := map[string]bool{}
	for _, id := range []string{"left", "right", "omp-1", "omp-2", "task-001-omp-1", "task-001-omp-2", "shell-1", "proc-1"} {
		seen[impArt(id, session.StatusStopped, 0)] = true
	}
	if len(seen) < 3 {
		t.Fatalf("imp avatar selection produced %d variants, want at least 3", len(seen))
	}
}

func TestSessionListWidthStaysAtTwoCards(t *testing.T) {
	m := testModel()
	rendered := m.renderSessionList(120, 0)
	for _, line := range strings.Split(rendered, "\n") {
		if lipgloss.Width(line) > sessionPanelTargetWidth {
			t.Fatalf("session list line width = %d, want <= %d: %q", lipgloss.Width(line), sessionPanelTargetWidth, line)
		}
	}
}

func TestViewShowsSessionListAndSelectedLogsOnly(t *testing.T) {
	m := testModel()
	m.width = 100
	m.height = 24
	m.manager = session.NewManager([]session.Session{
		{ID: "left", Name: "left", Kind: session.SessionKindPTY, Status: session.StatusRunning, Logs: []string{"left-only"}},
		{ID: "right", Name: "right", Kind: session.SessionKindPTY, Status: session.StatusRunning, Logs: []string{"right-only"}},
	})

	view := m.View()
	for _, want := range []string{"Sessions", "left", "right", "left-only", "_/\\_", "$.$", "slp", "(o_o)"} {
		if !strings.Contains(view, want) {
			t.Fatalf("view does not contain %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "right-only") {
		t.Fatalf("view shows non-selected session log:\n%s", view)
	}
}

func TestDisplayLogLinesSanitizesPTYControlSequences(t *testing.T) {
	lines := displayLogLines([]string{"\x1b[2J\x1b[H\r\nprompt> go version\r\n\x07"})
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "\x07") || strings.Contains(got, "\r") {
		t.Fatalf("display log still contains terminal controls: %q", got)
	}
	if !strings.Contains(got, "prompt> go version") {
		t.Fatalf("display log lost printable output: %q", got)
	}
}

func TestEnterAttachesOnlyRunningPTYSession(t *testing.T) {
	m := testModel()
	m.selectedSession = 1

	// Erster Enter: wechselt von focusSessions zu focusLogs
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != UIModeMonitor || m.attachedSessionID != "" {
		t.Fatalf("stopped PTY attach mode = %s/%q, want monitor/empty", m.mode, m.attachedSessionID)
	}
	if m.focus != focusLogs {
		t.Fatalf("stopped PTY enter focus = %d, want logs", m.focus)
	}

	// Session auf running setzen
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning

	// Zweiter Enter: von focusLogs zu attach
	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != UIModeAttach || m.attachedSessionID != "shell-1" {
		t.Fatalf("running PTY attach mode = %s/%q, want attach/shell-1", m.mode, m.attachedSessionID)
	}
}

func TestAttachModeForwardsKeysAndEscDetaches(t *testing.T) {
	m := testModel()
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning
	m.mode = UIModeAttach
	m.attachedSessionID = "shell-1"

	updated, cmd := m.Update(keyRune('q'))
	if cmd != nil {
		t.Fatal("q in attach mode returned a command; want forwarded input")
	}
	m = modelFromUpdate(t, updated)
	s, _ = m.manager.SessionByID("shell-1")
	if got := strings.Join(s.Logs, "\n"); !strings.Contains(got, "[system] write failed:") {
		t.Fatalf("attach key did not attempt PTY write: %v", s.Logs)
	}

	m = updateForTest(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != UIModeMonitor || m.attachedSessionID != "" {
		t.Fatalf("esc detach mode = %s/%q, want monitor/empty", m.mode, m.attachedSessionID)
	}
}

func TestSpaceStoppingPTYDoesNotQuitTUI(t *testing.T) {
	m := testModel()
	m.selectedSession = 1
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning

	updated, cmd := m.Update(keyRune(' '))
	if cmd != nil {
		t.Fatal("space on running PTY returned a command; want no quit command")
	}
	m = modelFromUpdate(t, updated)
	if m.mode != UIModeMonitor {
		t.Fatalf("mode after PTY stop = %s, want monitor", m.mode)
	}
}

func TestPTYOutputMessagesCoalesceTypedCharacters(t *testing.T) {
	m := testModel()
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning
	s.Logs = []string{"[system] started PTY: shell"}

	m = updateForTest(t, m, event.SessionPTYOutputMsg{SessionID: "shell-1", Data: []byte("a")})
	m = updateForTest(t, m, event.SessionPTYOutputMsg{SessionID: "shell-1", Data: []byte("b")})
	m = updateForTest(t, m, event.SessionPTYOutputMsg{SessionID: "shell-1", Data: []byte("c")})

	s, _ = m.manager.SessionByID("shell-1")
	if got := strings.Join(s.Logs, "|"); got != "[system] started PTY: shell|abc" {
		t.Fatalf("PTY typed output logs = %q, want one coalesced line", got)
	}
}

func TestPTYMessagesAppendAndExit(t *testing.T) {
	m := testModel()
	s, _ := m.manager.SessionByID("shell-1")
	s.Status = session.StatusRunning
	m.mode = UIModeAttach
	m.attachedSessionID = "shell-1"

	m = updateForTest(t, m, event.SessionPTYOutputMsg{SessionID: "shell-1", Data: []byte("hello\r\n")})
	s, _ = m.manager.SessionByID("shell-1")
	if got := strings.Join(s.Logs, "\n"); !strings.Contains(got, "hello") {
		t.Fatalf("PTY output not appended: %v", s.Logs)
	}

	m = updateForTest(t, m, event.SessionPTYExitedMsg{SessionID: "shell-1"})
	s, _ = m.manager.SessionByID("shell-1")
	if s.Status != session.StatusStopped {
		t.Fatalf("status after PTY exit = %s, want stopped", s.Status)
	}
	if m.mode != UIModeMonitor || m.attachedSessionID != "" {
		t.Fatalf("mode after attached PTY exit = %s/%q, want monitor/empty", m.mode, m.attachedSessionID)
	}
}

func TestFooterShowsMonitorAndAttachModes(t *testing.T) {
	m := testModel()
	if footer := m.renderFooter(); !strings.Contains(footer, "MONITOR") || !strings.Contains(footer, "r rename") {
		t.Fatalf("monitor footer = %q", footer)
	}
	m.mode = UIModeAttach
	m.attachedSessionID = "shell-1"
	if footer := m.renderFooter(); !strings.Contains(footer, "ATTACHED to shell-1 | esc detach") {
		t.Fatalf("attach footer = %q", footer)
	}
	m.mode = UIModeRename
	m.renameSessionID = "shell-1"
	m.renameInput = "shell renamed"
	if footer := m.renderFooter(); !strings.Contains(footer, "RENAME | enter save") {
		t.Fatalf("rename footer = %q", footer)
	}
	m.mode = UIModeNewAgent
	m.newAgentWorkDirInput = "C:/work"
	if footer := m.renderFooter(); !strings.Contains(footer, "NEW AGENT | tab complete") {
		t.Fatalf("new agent footer = %q", footer)
	}
}

func updateForTest(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	updated, _ := m.Update(msg)
	return modelFromUpdate(t, updated)
}

func modelFromUpdate(t *testing.T, updated tea.Model) Model {
	t.Helper()
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("updated model has type %T", updated)
	}
	return next
}

func keyRune(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

func TestSessionListScrolling(t *testing.T) {
	m := testModel()
	var sessions []session.Session
	for i := 0; i < 10; i++ {
		sessions = append(sessions, session.Session{
			ID:     fmt.Sprintf("session-%d", i),
			Name:   fmt.Sprintf("Session %d", i),
			Kind:   session.SessionKindPTY,
			Status: session.StatusStopped,
		})
	}
	m.manager = session.NewManager(sessions)

	maxHeight := 22

	m.selectedSession = 0
	rendered := m.renderSessionList(120, maxHeight)
	if !strings.Contains(rendered, "session-0") || !strings.Contains(rendered, "session-1") {
		t.Fatalf("rendered sessions did not include session-0 or session-1 when selected: %s", rendered)
	}
	if strings.Contains(rendered, "session-8") {
		t.Fatalf("rendered sessions included off-screen session-8: %s", rendered)
	}

	m.selectedSession = 8
	rendered = m.renderSessionList(120, maxHeight)
	if !strings.Contains(rendered, "session-8") || !strings.Contains(rendered, "session-9") {
		t.Fatalf("rendered sessions did not include session-8 or session-9 when selected: %s", rendered)
	}
	if strings.Contains(rendered, "session-0") {
		t.Fatalf("rendered sessions included off-screen session-0 after scrolling: %s", rendered)
	}
}
