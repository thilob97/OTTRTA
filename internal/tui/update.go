package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/handyfun97/ottrta/internal/agent"
	"github.com/handyfun97/ottrta/internal/event"
	"github.com/handyfun97/ottrta/internal/session"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.updateKey(msg)
	case animationTickMsg:
		m.animationFrame++
		return m, tickAnimation()
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeRunningPTYs()
		return m, nil
	case event.SessionLogMsg:
		m.manager.AppendLog(msg.SessionID, msg.Line)
		return m, m.pollProcess(msg.SessionID)
	case event.SessionExitedMsg:
		m.manager.AppendExitLog(msg.SessionID, msg.Err)
		m.manager.MarkStopped(msg.SessionID)
		delete(m.processEvents, msg.SessionID)
		return m, nil
	case event.SessionPTYOutputMsg:
		m.manager.AppendOutput(msg.SessionID, string(msg.Data))
		if s, ok := m.manager.SessionByID(msg.SessionID); ok && s.Kind == session.SessionKindAgent {
			if s.AgentKind == session.AgentKindOmp && agent.CheckAttention(string(msg.Data)) {
				m.manager.SetAttention(msg.SessionID, true)
			}
		}
		return m, m.pollPTY(msg.SessionID)
	case event.SessionPTYExitedMsg:
		m.manager.AppendExitLog(msg.SessionID, msg.Err)
		m.manager.MarkStopped(msg.SessionID)
		delete(m.ptyEvents, msg.SessionID)
		if m.attachedSessionID == msg.SessionID {
			m.mode = UIModeMonitor
			m.attachedSessionID = ""
		}
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) updateKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case UIModeAttach:
		return m.updateAttachKey(msg)
	case UIModeRename:
		return m.updateRenameKey(msg)
	case UIModeNewAgent:
		return m.updateNewAgentKey(msg)
	default:
		return m.updateMonitorKey(msg)
	}
}

func (m Model) updateMonitorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.manager.StopAllProcesses()
		return m, tea.Quit
	case "j", "down":
		sessions := m.manager.Sessions()
		if m.selectedSession+2 < len(sessions) {
			m.selectedSession += 2
		}
	case "k", "up":
		if m.selectedSession >= 2 {
			m.selectedSession -= 2
		}
	case "h", "left":
		switch m.focus {
		case focusSessions:
			if m.selectedSession%2 == 1 {
				m.selectedSession--
			}
		case focusLogs:
			m.focus = focusSessions
		}
	case "l", "right":
		switch m.focus {
		case focusSessions:
			sessions := m.manager.Sessions()
			if m.selectedSession%2 == 0 && m.selectedSession+1 < len(sessions) {
				m.selectedSession++
			}
		}
	case "esc":
		if m.focus == focusLogs {
			m.focus = focusSessions
		}
	case "enter":
		return m.attachSelected()
	case " ", "space":
		return m.toggleSelected()
	case "n":
		return m.beginNewAgent()
	case "r":
		return m.beginRenameSelected()
	case "x":
		return m.removeSelected()
	}
	return m, nil
}

func (m Model) updateAttachKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc {
		// Resize PTY back to monitor view size
		if s, ok := m.manager.SessionByID(m.attachedSessionID); ok {
			cols := m.ptyCols()
			rows := m.ptyRows()
			m.manager.ResizePTYSession(s.ID, cols, rows)
		}
		m.mode = UIModeMonitor
		m.attachedSessionID = ""
		return m, nil
	}

	data := keyToBytes(msg)
	if len(data) == 0 {
		return m, nil
	}
	if err := m.manager.WritePTYSession(m.attachedSessionID, data); err != nil {
		m.manager.AppendLog(m.attachedSessionID, fmt.Sprintf("[system] write failed: %v", err))
	}
	return m, nil
}

func (m Model) updateRenameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.manager.StopAllProcesses()
		return m, tea.Quit
	case tea.KeyEsc:
		m.mode = UIModeMonitor
		m.renameSessionID = ""
		m.renameInput = ""
		return m, nil
	case tea.KeyEnter:
		if m.manager.RenameSession(m.renameSessionID, m.renameInput) {
			m.selectedSession = m.manager.ClampIndex(m.selectedSession)
		}
		m.mode = UIModeMonitor
		m.renameSessionID = ""
		m.renameInput = ""
		return m, nil
	case tea.KeyBackspace, tea.KeyCtrlH:
		runes := []rune(m.renameInput)
		if len(runes) > 0 {
			m.renameInput = string(runes[:len(runes)-1])
		}
		return m, nil
	case tea.KeySpace:
		m.renameInput += " "
		return m, nil
	case tea.KeyRunes:
		m.renameInput += string(msg.Runes)
		return m, nil
	default:
		return m, nil
	}
}


func (m Model) updateNewAgentKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.newAgentStep == 0 {
		switch msg.Type {
		case tea.KeyCtrlC:
			m.manager.StopAllProcesses()
			return m, tea.Quit
		case tea.KeyEsc:
			m.mode = UIModeMonitor
			m.newAgentCommandInput = ""
			m.newAgentWorkDirInput = ""
			m.newAgentCompletionHint = ""
			return m, nil
		case tea.KeyEnter:
			cmd := strings.TrimSpace(m.newAgentCommandInput)
			if cmd != "" {
				m.newAgentStep = 1
			}
			return m, nil
		case tea.KeyBackspace, tea.KeyCtrlH:
			runes := []rune(m.newAgentCommandInput)
			if len(runes) > 0 {
				m.newAgentCommandInput = string(runes[:len(runes)-1])
			}
			return m, nil
		case tea.KeySpace:
			m.newAgentCommandInput += " "
			return m, nil
		case tea.KeyRunes:
			m.newAgentCommandInput += string(msg.Runes)
			return m, nil
		default:
			return m, nil
		}
	}

	// Step 1: WorkDir
	switch msg.Type {
	case tea.KeyCtrlC:
		m.manager.StopAllProcesses()
		return m, tea.Quit
	case tea.KeyEsc:
		m.newAgentStep = 0
		return m, nil
	case tea.KeyEnter:
		cmd := strings.TrimSpace(m.newAgentCommandInput)
		workdir := strings.TrimSpace(m.newAgentWorkDirInput)
		m.mode = UIModeMonitor
		m.newAgentCommandInput = ""
		m.newAgentWorkDirInput = ""
		m.newAgentCompletionHint = ""
		return m.addNewAgent(cmd, workdir)
	case tea.KeyBackspace, tea.KeyCtrlH:
		runes := []rune(m.newAgentWorkDirInput)
		if len(runes) > 0 {
			m.newAgentWorkDirInput = string(runes[:len(runes)-1])
		}
		m.newAgentCompletionHint = ""
		return m, nil
	case tea.KeyTab:
		m.newAgentWorkDirInput, m.newAgentCompletionHint = completeDirectoryPath(m.newAgentWorkDirInput)
		return m, nil
	case tea.KeySpace:
		m.newAgentWorkDirInput += " "
		m.newAgentCompletionHint = ""
		return m, nil
	case tea.KeyRunes:
		m.newAgentWorkDirInput += string(msg.Runes)
		m.newAgentCompletionHint = ""
		return m, nil
	default:
		return m, nil
	}
}

func (m Model) beginNewAgent() (tea.Model, tea.Cmd) {
	m.mode = UIModeNewAgent
	m.newAgentCommandInput = ""
	m.newAgentWorkDirInput = ""
	m.newAgentCompletionHint = ""
	m.newAgentStep = 0
	return m, nil
}

func (m Model) beginRenameSelected() (tea.Model, tea.Cmd) {
	s, ok := m.manager.Session(m.selectedSession)
	if !ok {
		return m, nil
	}
	m.mode = UIModeRename
	m.renameSessionID = s.ID
	m.renameInput = s.Name
	return m, nil
}

func (m Model) attachSelected() (tea.Model, tea.Cmd) {
	s, ok := m.manager.Session(m.selectedSession)
	if !ok {
		return m, nil
	}
	if (s.Kind == session.SessionKindPTY || s.Kind == session.SessionKindAgent) && s.Status == session.StatusRunning {
		m.manager.SetAttention(s.ID, false)
		m.mode = UIModeAttach
		m.attachedSessionID = s.ID
		// Resize PTY to match attach view width
		termWidth := m.width - 4
		if termWidth < 30 {
			termWidth = 30
		}
		contentWidth := termWidth - 4
		rows := m.ptyRows()
		m.manager.ResizePTYSession(s.ID, contentWidth, rows)
		return m, nil
	}
	m.focus = focusLogs
	return m, nil
}
func (m Model) addNewAgent(command string, workdir string) (tea.Model, tea.Cmd) {
	cmdPrefix := command
	if cmdPrefix == "" {
		cmdPrefix = "agent"
	}
	var sb strings.Builder
	for _, r := range cmdPrefix {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	prefix := sb.String()
	if prefix == "" {
		prefix = "agent"
	}

	maxID := 0
	for _, s := range m.manager.Sessions() {
		if s.Kind == session.SessionKindAgent {
			var id int
			if _, err := fmt.Sscanf(s.ID, prefix+"-%d", &id); err == nil {
				if id > maxID {
					maxID = id
				}
			}
		}
	}
	nextID := maxID + 1
	name := fmt.Sprintf("%s-%d", prefix, nextID)
	displayName := session.RandomImpName()
	log := fmt.Sprintf("%s ready", displayName)
	if workdir != "" {
		log = fmt.Sprintf("%s ready (cwd: %s)", displayName, workdir)
	}

	agentKind := session.AgentKind(prefix)
	m.manager.AddSession(session.Session{
		ID:        name,
		Name:      displayName,
		Kind:      session.SessionKindAgent,
		AgentKind: agentKind,
		Status:    session.StatusStopped,
		Command:   command,
		WorkDir:   workdir,
		Logs:      []string{log},
	})
	m.selectedSession = m.manager.Count() - 1
	return m, nil
}
func (m Model) removeSelected() (tea.Model, tea.Cmd) {
	m.manager.RemoveSession(m.selectedSession)
	m.selectedSession = m.manager.ClampIndex(m.selectedSession)
	return m, nil
}

func (m Model) toggleSelected() (tea.Model, tea.Cmd) {
	s, ok := m.manager.Session(m.selectedSession)
	if !ok {
		return m, nil
	}

	switch s.Kind {
	case session.SessionKindProcess:
		if s.Status == session.StatusRunning {
			if err := m.manager.StopSession(s.ID); err != nil {
				m.manager.AppendLog(s.ID, fmt.Sprintf("[system] stop failed: %v", err))
			}
			return m, nil
		}

		events, err := m.manager.StartSession(processContext(), s.ID)
		if err != nil {
			m.manager.AppendLog(s.ID, fmt.Sprintf("[system] start failed: %v", err))
			return m, nil
		}
		if m.processEvents == nil {
			m.processEvents = make(map[string]<-chan event.ProcessMsg)
		}
		m.processEvents[s.ID] = events
		return m, pollProcessEvent(s.ID, events)
	case session.SessionKindPTY, session.SessionKindAgent:
		if s.Kind == session.SessionKindAgent && s.AgentKind == session.AgentKindOmp && !agent.OmpExists() {
			if s.Status != session.StatusRunning {
				m.manager.AppendLog(s.ID, "[system] omp executable not found in PATH")
				m.manager.SetStatus(s.ID, session.StatusFailed)
			}
			return m, nil
		}

		if s.Status == session.StatusRunning {
			if err := m.manager.StopPTYSession(s.ID); err != nil {
				m.manager.AppendLog(s.ID, fmt.Sprintf("[system] stop failed: %v", err))
			}
			if m.attachedSessionID == s.ID {
				m.mode = UIModeMonitor
				m.attachedSessionID = ""
			}
			delete(m.ptyEvents, s.ID)
			return m, nil
		}

		events, err := m.manager.StartPTYSession(processContext(), s.ID, m.ptyCols(), m.ptyRows())
		if err != nil {
			m.manager.AppendLog(s.ID, fmt.Sprintf("[system] start failed: %v", err))
			return m, nil
		}
		if m.ptyEvents == nil {
			m.ptyEvents = make(map[string]<-chan event.PTYMsg)
		}
		ptyEvents := adaptPTYEvents(events)
		m.ptyEvents[s.ID] = ptyEvents
		return m, pollPTYEvent(s.ID, ptyEvents)
	default:
		return m, nil
	}
}

func (m Model) pollProcess(sessionID string) tea.Cmd {
	events, ok := m.processEvents[sessionID]
	if !ok {
		return nil
	}
	return pollProcessEvent(sessionID, events)
}

func (m Model) pollPTY(sessionID string) tea.Cmd {
	events, ok := m.ptyEvents[sessionID]
	if !ok {
		return nil
	}
	return pollPTYEvent(sessionID, events)
}

func (m Model) resizeRunningPTYs() {
	var cols, rows int
	if m.mode == UIModeAttach && m.attachedSessionID != "" {
		// In attach mode, resize the attached PTY to full width
		termWidth := m.width - 4
		if termWidth < 30 {
			termWidth = 30
		}
		cols = termWidth - 4 // border(2) + padding(2)
		rows = m.ptyRows()
		if err := m.manager.ResizePTYSession(m.attachedSessionID, cols, rows); err != nil {
			m.manager.AppendLog(m.attachedSessionID, fmt.Sprintf("[system] resize failed: %v", err))
		}
		// Keep other PTYs at monitor size
		cols = m.ptyCols()
		rows = m.ptyRows()
		for _, s := range m.manager.Sessions() {
			if s.ID == m.attachedSessionID {
				continue
			}
			if (s.Kind != session.SessionKindPTY && s.Kind != session.SessionKindAgent) || s.Status != session.StatusRunning {
				continue
			}
			if err := m.manager.ResizePTYSession(s.ID, cols, rows); err != nil {
				m.manager.AppendLog(s.ID, fmt.Sprintf("[system] resize failed: %v", err))
			}
		}
	} else {
		// In monitor mode, all PTYs use monitor view size
		cols = m.ptyCols()
		rows = m.ptyRows()
		for _, s := range m.manager.Sessions() {
			if (s.Kind != session.SessionKindPTY && s.Kind != session.SessionKindAgent) || s.Status != session.StatusRunning {
				continue
			}
			if err := m.manager.ResizePTYSession(s.ID, cols, rows); err != nil {
				m.manager.AppendLog(s.ID, fmt.Sprintf("[system] resize failed: %v", err))
			}
		}
	}
}

func (m Model) ptyCols() int {
	width := m.width
	if width < 60 {
		width = 90
	}
	leftWidth := width / 3
	if leftWidth < 24 {
		leftWidth = 24
	}
	cols := width - leftWidth - 6
	if cols < 20 {
		return 20
	}
	return cols
}

func (m Model) ptyRows() int {
	height := m.height
	if height < 12 {
		height = 24
	}
	rows := height - 7
	if rows < 5 {
		return 5
	}
	return rows
}
