package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/handyfun97/ottrta/internal/session"
)

var impArts = [...][4]string{
	{
		`  (\_/)  `,
		` / ^_^ \ `,
		`(  "*"  )`,
		" `--`--` ",
	},
	{
		` /\_/\  `,
		`( o.o ) `,
		` / ^ \  `,
		`  v v   `,
	},
	{
		`  /\/\  `,
		` ( •• ) `,
		` /(><)\ `,
		`  "  "  `,
	},
	{
		`  .-.-. `,
		` ( 0_0 )`,
		`<(  :  )`,
		`  ^^ ^^ `,
	},
	{
		`  /\=/\ `,
		` ( -_- )`,
		` /|:::|\`,
		`  /   \ `,
	},
	{
		`  (\ /) `,
		`  (x_x) `,
		` <( " )>`,
		`  /_|_\ `,
	},
	{
		`  /^ ^\ `,
		` ( @ @ )`,
		` /  *  \`,
		`  m---m `,
	},
	{
		`  (\w/) `,
		` ( >.< )`,
		` /(   )\`,
		`  d   b `,
	},
	{
		`  _/\_  `,
		` ( $.$ )`,
		` /|slp|\`,
		`  /___\ `,
	},
	{
		`  (\/)  `,
		` (o_o)  `,
		` <(~~~)>`,
		`  /   \ `,
	},
	{
		`  /\_/\ `,
		` ( @_@ )`,
		` /{:::}\`,
		`   u u  `,
	},
	{
		`  (^^^) `,
		` ( -o- )`,
		` /|+++|\`,
		"  /`-'\\ ",
	},
}

var impColors = [...]lipgloss.Color{
	lipgloss.Color("39"),
	lipgloss.Color("42"),
	lipgloss.Color("45"),
	lipgloss.Color("81"),
	lipgloss.Color("99"),
	lipgloss.Color("135"),
	lipgloss.Color("171"),
	lipgloss.Color("202"),
	lipgloss.Color("208"),
	lipgloss.Color("213"),
	lipgloss.Color("220"),
	lipgloss.Color("228"),
}

const (
	sessionCardMaxWidth      = 24
	sessionPanelTargetWidth  = 2*sessionCardMaxWidth + 7
	sessionPanelMinimumWidth = 42
	logPanelMinimumWidth     = 30
)

func (m Model) View() string {
	width := m.width
	if width < 60 {
		width = 90
	}
	height := m.height
	if height < 12 {
		height = 24
	}

	// Reserve space for footer and margins
	footerHeight := 1
	panelHeight := height - footerHeight - 2
	if panelHeight < 8 {
		panelHeight = 8
	}

	if m.mode == UIModeAttach {
		return m.renderAttachView(width, height, panelHeight)
	}

	leftWidth := sessionPanelTargetWidth
	if leftWidth < sessionPanelMinimumWidth {
		leftWidth = sessionPanelMinimumWidth
	}
	rightWidth := width - leftWidth - 4
	if rightWidth < logPanelMinimumWidth {
		rightWidth = logPanelMinimumWidth
		leftWidth = width - rightWidth - 4
		if leftWidth < sessionPanelMinimumWidth {
			leftWidth = sessionPanelMinimumWidth
		}
	}

	banner := renderBanner()
	bannerLines := strings.Count(banner, "\n")

	availableSessionHeight := panelHeight - 2 - bannerLines
	if availableSessionHeight < 5 {
		availableSessionHeight = 5
	}

	leftContent := banner + m.renderSessionList(leftWidth, availableSessionHeight)

	left := panelStyle(m.focus == focusSessions).
		Width(leftWidth).
		Height(panelHeight).
		Render(leftContent)
	right := termPanelStyle(m.focus == focusLogs).
		Width(rightWidth).
		Height(panelHeight).
		Render(m.renderLogPanel(rightWidth, panelHeight))

	footer := mutedStyle.Render(m.renderFooter())
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		footer,
	)
	if modal := m.renderInputModal(); modal != "" {
		return centerOverlay(view, modal, width, height)
	}
	return view
}

func (m Model) renderAttachView(width, height, panelHeight int) string {
	s, ok := m.manager.Session(m.selectedSession)
	if !ok {
		return ""
	}

	termWidth := width - 4
	if termWidth < 30 {
		termWidth = 30
	}
	contentWidth := termWidth - 4

	headerLeft := titleStyle.Render(fmt.Sprintf(" ATTACHED: %s ", s.Name))
	headerRight := mutedStyle.Render("  esc detach")
	headerPad := contentWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if headerPad < 0 {
		headerPad = 0
	}
	header := headerLeft + strings.Repeat(" ", headerPad) + headerRight

	visible := panelHeight - 1
	if visible < 1 {
		visible = 1
	}
	displayLogs := displayLogLines(s.Logs)
	if len(displayLogs) > visible {
		displayLogs = displayLogs[len(displayLogs)-visible:]
	}

	var lines []string
	for _, line := range displayLogs {
		lines = append(lines, truncateANSI(line, contentWidth))
	}
	for len(lines) < visible {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	term := attachPanelStyle.
		Width(termWidth).
		Height(panelHeight).
		Render(header + "\n" + content)

	return lipgloss.JoinVertical(lipgloss.Left, term)
}

func (m Model) renderSessionList(width int, maxHeight int) string {
	sessions := m.manager.Sessions()
	if len(sessions) == 0 {
		return strings.Join([]string{titleStyle.Render("Sessions"), mutedStyle.Render("No sessions")}, "\n")
	}

	cardWidth := (width - 7) / 2
	if cardWidth > sessionCardMaxWidth {
		cardWidth = sessionCardMaxWidth
	}
	if cardWidth < 16 {
		cardWidth = 16
	}

	type rowInfo struct {
		height   int
		rendered string
	}
	var rows []rowInfo

	for i := 0; i < len(sessions); i += 2 {
		left := m.renderSessionCard(&sessions[i], cardWidth, i == m.selectedSession)
		leftHeight := strings.Count(left, "\n") + 1

		var row string
		var rowHeight int
		if i+1 >= len(sessions) {
			row = left
			rowHeight = leftHeight
		} else {
			right := m.renderSessionCard(&sessions[i+1], cardWidth, i+1 == m.selectedSession)
			row = lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right)
			rightHeight := strings.Count(right, "\n") + 1
			if leftHeight > rightHeight {
				rowHeight = leftHeight
			} else {
				rowHeight = rightHeight
			}
		}
		rows = append(rows, rowInfo{
			height:   rowHeight,
			rendered: row,
		})
	}

	availHeight := maxHeight - 1 // 1 line for the "Sessions" title
	if availHeight < 5 {
		availHeight = 5
	}

	selectedRow := m.selectedSession / 2
	if selectedRow >= len(rows) {
		selectedRow = len(rows) - 1
	}
	if selectedRow < 0 {
		selectedRow = 0
	}

	// Calculate startRow going backwards from selectedRow to fit within availHeight
	startRow := selectedRow
	sum := 0
	if selectedRow < len(rows) {
		sum = rows[selectedRow].height
	}
	for r := selectedRow - 1; r >= 0; r-- {
		if sum+rows[r].height > availHeight {
			break
		}
		startRow = r
		sum += rows[r].height
	}

	lines := []string{titleStyle.Render("Sessions")}
	usedHeight := 1
	for r := startRow; r < len(rows); r++ {
		if usedHeight+rows[r].height > maxHeight {
			break
		}
		lines = append(lines, rows[r].rendered)
		usedHeight += rows[r].height
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderSessionCard(s *session.Session, width int, selected bool) string {
	status := renderStatus(s.Status)
	if s.NeedsAttention {
		status += " !"
	}

	avatarStr := impArt(s.ID, s.Status, m.animationFrame)
	avatarWidth := lipgloss.Width(avatarStr)

	textWidth := width - avatarWidth - 5
	if textWidth < 8 {
		textWidth = 8
	}

	nameLines := cardNameLines(s.Name, textWidth)
	idLine := mutedStyle.Render(truncateANSI(s.ID, textWidth))
	statusLine := status

	var rightRows []string
	rightRows = append(rightRows, nameLines...)
	rightRows = append(rightRows, idLine, statusLine)
	rightCol := strings.Join(rightRows, "\n")

	cardContent := lipgloss.JoinHorizontal(lipgloss.Top, avatarStr, " ", rightCol)

	style := sessionCardStyle
	if selected {
		style = selectedSessionCardStyle
	}
	return style.Width(width).Render(cardContent)
}

var runningImpBanners = [...]string{
	"* slop *",
	"✦ slop ✦",
	"> slop <",
	"~ slop ~",
}

func impArt(sessionID string, status session.Status, frame int) string {
	index := stableImpIndex(sessionID)
	avatar := impArts[index]
	color := impColors[index%len(impColors)]
	banner := "       "
	if status == session.StatusStopped {
		banner = " zZzZ  "
	} else if status == session.StatusRunning {
		banner = runningImpBanners[frame%len(runningImpBanners)]
	} else if status == session.StatusFailed {
		banner = "  !!!  "
	}
	art := banner + "\n" + strings.Join(avatar[:], "\n")
	return fmt.Sprintf("\x1b[38;5;%sm%s\x1b[0m", string(color), art)
}

func cardNameLines(name string, width int) []string {
	if width < 8 {
		width = 8
	}
	if lipgloss.Width(name) <= width {
		return []string{titleStyle.Render(name)}
	}
	words := strings.Fields(name)
	if len(words) < 2 {
		return []string{titleStyle.Render(truncateANSI(name, width))}
	}
	first := words[0]
	second := strings.Join(words[1:], " ")
	return []string{
		titleStyle.Render(truncateANSI(first, width)),
		titleStyle.Render(truncateANSI(second, width)),
	}
}

func stableImpIndex(sessionID string) int {
	if len(impArts) == 1 {
		return 0
	}
	var hash uint32 = 2166136261
	for i := 0; i < len(sessionID); i++ {
		hash ^= uint32(sessionID[i])
		hash *= 16777619
	}
	return int(hash % uint32(len(impArts)))
}

func renderBanner() string {
	banner := " ___ _____ _____ ____ _____  _\n" +
		"| . |_   _|_   _|  _ \\_   _|/ \\\n" +
		"| | | | |   | | |    / | | / _ \\\n" +
		"|___| |_|   |_| |_|_\\  |_|/_/ \\_\\"
	return titleStyle.Render(banner) + "\n"
}

func (m Model) renderLogPanel(width int, height int) string {
	s, ok := m.manager.Session(m.selectedSession)
	if !ok {
		return mutedStyle.Render("No session selected")
	}

	contentWidth := width - 4
	visible := height - 2
	if visible < 1 {
		visible = 1
	}

	headerLeft := titleStyle.Render(fmt.Sprintf(" %s ", s.Name))
	headerRight := renderStatus(s.Status)
	if s.NeedsAttention {
		headerRight += " !"
	}
	headerPad := contentWidth - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight)
	if headerPad < 0 {
		headerPad = 0
	}
	header := headerLeft + strings.Repeat(" ", headerPad) + headerRight

	displayLogs := displayLogLines(s.Logs)
	if len(displayLogs) > visible {
		displayLogs = displayLogs[len(displayLogs)-visible:]
	}

	var lines []string
	for _, line := range displayLogs {
		lines = append(lines, truncateANSI(line, contentWidth))
	}
	for len(lines) < visible {
		lines = append(lines, "")
	}

	content := strings.Join(lines, "\n")
	return header + "\n" + content
}

func displayLogLines(logs []string) []string {
	if len(logs) == 0 {
		return []string{mutedStyle.Render("No logs yet")}
	}
	sanitized := make([]string, len(logs))
	for i, log := range logs {
		sanitized[i] = sanitizeLogText(log)
	}
	return sanitized
}

// sanitizeLogText strips destructive terminal control sequences (clear screen,
// cursor movement, BEL, etc.) but keeps SGR color/style sequences (ending in 'm')
// so log output retains its colors.
func sanitizeLogText(text string) string {
	text = strings.ReplaceAll(text, "\r", "")
	text = strings.ReplaceAll(text, "\t", "    ")
	var builder strings.Builder
	i := 0
	for i < len(text) {
		c := text[i]
		// Detect start of an ANSI escape sequence
		if c == '\x1b' && i+1 < len(text) && text[i+1] == '[' {
			// Capture the full CSI sequence: ESC [ <params> <letter>
			start := i
			j := i + 2 // skip ESC and [
			for j < len(text) && !((text[j] >= 'A' && text[j] <= 'Z') || (text[j] >= 'a' && text[j] <= 'z')) {
				j++
			}
			if j < len(text) {
				terminator := text[j]
				seq := text[start : j+1]
				if terminator == 'm' {
					// SGR (color/style) — keep it
					builder.WriteString(seq)
				}
				// Everything else (J=clear, H/f=cursor, K=erase line, etc.) — drop
				i = j + 1
				continue
			}
			// Malformed sequence — drop the ESC
			i++
			continue
		}
		// Drop bare ESC not followed by [
		if c == '\x1b' {
			i++
			continue
		}
		if c >= 32 && c != 127 {
			builder.WriteByte(c)
		} else if c == '\n' {
			builder.WriteByte(c)
		}
		// Drop BEL (\x07), other control chars
		i++
	}
	return builder.String()
}

// truncateANSI truncates a string containing ANSI escapes to a visible width.
func truncateANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	visible := 0
	var out strings.Builder
	inEscape := false
	for _, r := range s {
		if inEscape {
			out.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		if r == '\x1b' {
			inEscape = true
			out.WriteRune(r)
			continue
		}
		if visible >= maxWidth {
			break
		}
		out.WriteRune(r)
		visible++
	}
	result := out.String()
	if strings.Contains(result, "\x1b[") {
		result += "\x1b[0m"
	}
	return result
}

func (m Model) renderInputModal() string {
	switch m.mode {
	case UIModeRename:
		name := m.renameSessionID
		if s, ok := m.manager.SessionByID(m.renameSessionID); ok {
			name = s.Name
		}
		return renderModal("Rename session", "Current: "+name, "Name", m.renameInput, "", "enter save  esc cancel")
	case UIModeNewAgent:
		return renderModal("New OMP agent", "Create a stopped agent session.", "Working directory", m.newAgentWorkDirInput, m.newAgentCompletionHint, "tab complete  enter create  esc cancel")
	default:
		return ""
	}
}

func renderModal(title, subtitle, label, value, hint, help string) string {
	const width = 52
	input := value
	if input == "" {
		input = mutedStyle.Render("(default current directory)")
	}
	lines := []string{
		titleStyle.Render(title),
		mutedStyle.Render(subtitle),
		"",
		label,
		"› " + input,
	}
	if hint != "" {
		lines = append(lines, mutedStyle.Render(hint))
	}
	lines = append(lines, "", mutedStyle.Render(help))
	body := strings.Join(lines, "\n")
	return modalStyle.Width(width).Render(body)
}

func centerOverlay(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	if len(baseLines) < height {
		for len(baseLines) < height {
			baseLines = append(baseLines, "")
		}
	}

	overlayWidth := 0
	for _, line := range overlayLines {
		if lineWidth := lipgloss.Width(line); lineWidth > overlayWidth {
			overlayWidth = lineWidth
		}
	}

	row := (height - len(overlayLines)) / 2
	if row < 0 {
		row = 0
	}
	col := (width - overlayWidth) / 2
	if col < 0 {
		col = 0
	}

	// Erstelle einen Hintergrund-Stil für die Leerzeichen
	bgStyle := lipgloss.NewStyle().Background(lipgloss.Color("0"))

	for i, line := range overlayLines {
		target := row + i
		if target >= len(baseLines) {
			break
		}
		// Fülle links und rechts mit Hintergrundfarbe
		leftPad := bgStyle.Render(strings.Repeat(" ", col))
		lineWidth := lipgloss.Width(line)
		remainingWidth := width - col - lineWidth
		rightPad := ""
		if remainingWidth > 0 {
			rightPad = bgStyle.Render(strings.Repeat(" ", remainingWidth))
		}
		baseLines[target] = leftPad + line + rightPad
	}
	return strings.Join(baseLines, "\n")
}

func (m Model) renderFooter() string {
	if m.mode == UIModeAttach {
		name := m.attachedSessionID
		if s, ok := m.manager.SessionByID(m.attachedSessionID); ok {
			name = s.Name
		}
		return fmt.Sprintf("ATTACHED to %s | esc detach", name)
	}
	if m.mode == UIModeRename {
		return "RENAME | enter save  esc cancel"
	}
	if m.mode == UIModeNewAgent {
		return "NEW AGENT | tab complete  enter create  esc cancel"
	}
	if s, ok := m.manager.Session(m.selectedSession); ok && s.NeedsAttention {
		return "selected session needs attention: press enter to attach"
	}
	return "MONITOR | j/k select  h/l focus  space start/stop  enter attach  n new agent  r rename  x remove  q quit"
}

func renderStatus(status session.Status) string {
	switch status {
	case session.StatusRunning:
		return runningStyle.Render("running")
	case session.StatusStopped:
		return stoppedStyle.Render("stopped")
	case session.StatusFailed:
		return failedStyle.Render("failed")
	default:
		return string(status)
	}
}
