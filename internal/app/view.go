package app

import (
	"fmt"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/keymap"
)

func (m *model) fitPage() {
	p := m.page()
	height := max(1, m.listHeight())
	p.top = max(0, min(p.top, max(0, len(p.rows)-height)))
	if p.cursor < p.top {
		p.top = p.cursor
	} else if p.cursor >= p.top+height {
		p.top = p.cursor - height + 1
	}
}

func (m *model) View() tea.View {
	if m.width <= 0 || m.height <= 0 {
		view := tea.NewView("Starting rämpd player…")
		view.AltScreen = true
		return view
	}

	var tabs []string
	for i, name := range tabNames {
		label, marker := name, ""
		if i == m.tab {
			label = m.styles.ActiveTab.Render(label)
			marker = m.styles.AccentText.Render("▸") + " "
		} else if m.sidebarWidth() > 0 {
			marker = "  "
		}
		if key := keymap.Button(m.keys.Key(tabActions[i])); key != "" {
			label = m.styles.Shortcut.Render(key) + " " + label
		}
		tabs = append(tabs, marker+label)
	}

	tabLine := strings.Join(tabs, "  ")
	if ansi.StringWidth(tabLine) > m.innerWidth(m.width) {
		tabLine = fmt.Sprintf("Tab: %s", tabNames[m.tab])
	}
	title := clean(m.page().title)
	if m.tab == trayTab && m.prompt != palettePrompt && len(m.trayMarks) > 0 {
		title += fmt.Sprintf(" · %d marked", len(m.trayMarks))
	}
	if m.onMediaList() {
		title += " · by " + m.mediaOrderLabel()
	} else if m.canSwitchTrackColumn() {
		title += " · " + m.trackColumnName()
	}
	var rows, footer []string
	if m.showHelp {
		title = "Keys (press any key to close)"
		rows = m.helpRows()
	} else {
		rows = m.viewRows()
	}
	rows = rows[:min(len(rows), m.listHeight())]
	if m.showFilter() {
		rows = append([]string{m.filterLine()}, rows...)
	}
	rows = append(rows, make([]string, m.bodyHeight()-len(rows))...)
	playback := m.playbackWithProgress()
	var inputRow int
	notice := ""
	footerHeight := m.footerHeight()
	showStatus := m.height >= 5 && (m.prompt != playlistNamePrompt || footerHeight > 2)
	switch {
	case m.prompt == playlistNamePrompt:
		footer = append(footer, m.input.View(), joinHints(m.shortcut("Enter", "Confirm"), m.shortcut("Esc", "Cancel")))
	case m.confirmation != nil:
		question := m.styles.AccentText.Render(m.confirmation.message)
		hints := joinHints(m.shortcut("y", "Confirm"), m.shortcut("Esc", "Cancel"))
		if footerHeight > 2 {
			footer = append(footer, question, hints)
		} else {
			footer = append(footer, joinHints(hints, question))
		}
	case m.tab == trayTab && m.prompt == noPrompt && !m.showHelp:
		footer = append(footer, m.trayFooter())
		notice = m.notice
		if m.traySaveErr != nil {
			notice = fmt.Sprintf("Tray not saved: %v", m.traySaveErr)
		}
	default:
		hint := m.actionHints()
		notice = m.notice
		if m.tab == dreamTab && m.dreamLoading {
			notice = "Generating a Dream preview…"
		}
		switch m.prompt {
		case searchPrompt:
			hint = joinHints(m.shortcut("Enter", "Return to list"), m.shortcut("Esc", "Clear filter"))
		case palettePrompt:
			hint = joinHints(m.styles.Shortcut.Render("[↑] [↓]")+" "+m.styles.Muted.Render("Select"),
				m.shortcut("Enter", "Run"), m.shortcut("Esc", "Close"))
		}
		footer = append(footer, hint)
	}

	var lines []string
	heading := m.styles.Heading.Render(title)
	switch {
	case m.framed():
		if m.sidebarWidth() == 0 {
			lines = []string{m.insetLine(tabLine, 0, m.width)}
		}
		inputRow = len(lines) + 1 // First line inside the content panel.
		lines = append(lines, m.framedBody(title, rows, tabs)...)
	case m.height < 8:
		inputRow = 1
		lines = append([]string{heading}, rows...)
	default:
		m.joinArtwork(rows)
		lines = []string{tabLine, m.pixelRule(m.width), heading}
		inputRow = len(lines)
		lines = append(lines, rows...)
	}
	if showStatus {
		lines = append(lines, m.statusLine(notice))
		footerHeight--
	}
	if m.framed() {
		for _, line := range m.frame("Playback", m.bodyWidth(), []string{playback}) {
			lines = append(lines, strings.Repeat(" ", m.sidebarWidth())+line)
		}
	} else {
		lines = append(lines, ansi.Truncate(playback, m.bodyWidth(), "…"))
	}
	if !m.inlineInput() {
		inputRow = len(lines)
	}
	for _, line := range fitRows(footer, footerHeight) {
		lines = append(lines, m.insetLine(line, m.sidebarWidth(), m.bodyWidth()))
	}
	if len(lines) > m.height {
		if (m.prompt != noPrompt || m.confirmation != nil) && m.height > 0 && inputRow >= m.height {
			lines[m.height-1] = lines[inputRow]
			inputRow = m.height - 1
		}
		lines = lines[:m.height]
	}
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, m.width, "…")
	}
	view := tea.NewView(strings.Join(lines, "\n"))
	view.AltScreen = true
	if m.prompt != noPrompt {
		if cursor := m.input.Cursor(); cursor != nil {
			if m.inlineInput() {
				padding := m.filterPadding()
				cursor.X = min(cursor.X+padding, m.innerWidth(m.bodyWidth())-padding-1)
				if m.framed() {
					cursor.X += panelInset + m.sidebarWidth()
				}
			} else {
				cursor.X = m.sidebarWidth() + m.textInset() + min(cursor.X, m.innerWidth(m.bodyWidth())-1)
			}
			cursor.Y = inputRow
			view.Cursor = cursor
		}
	}
	return view
}

func (m *model) insetLine(line string, start, width int) string {
	return strings.Repeat(" ", start+m.textInset()) + ansi.Truncate(line, m.innerWidth(width), "…")
}

func (m *model) statusLine(notice string) string {
	label, style := "connecting", m.styles.Muted
	switch {
	case m.connected:
		label, style = "online", m.styles.Online
	case m.connectionFailed:
		label, style = "offline", m.styles.Offline
	}
	width := m.bodyWidth()
	status := ansi.Truncate("MPD: "+label, width, "…")
	space := width - ansi.StringWidth(status)
	left := ansi.Truncate(clean(notice), max(0, space-2), "…")
	return strings.Repeat(" ", m.sidebarWidth()) + m.styles.Muted.Render(left) +
		strings.Repeat(" ", space-ansi.StringWidth(left)) + style.Render(status)
}

func (m *model) viewRows() []string {
	p := m.page()
	if len(p.rows) == 0 {
		return m.emptyRows()
	}

	width := m.innerWidth(m.bodyWidth())
	browsing := m.prompt != palettePrompt
	artistList := browsing && m.tab == artistsTab && len(m.tabs[artistsTab]) == 1
	var lines []string
	for i := p.top; i < min(len(p.rows), p.top+m.listHeight()); i++ {
		row := p.rows[i]
		selected := i == p.cursor
		marker := " "
		if browsing && m.tab == queueTab && m.status.State != "stop" && row.id == m.status.SongID {
			marker = "♫"
		} else if browsing && m.tab == trayTab && m.trayMarks[m.tray[row.position].key] {
			marker = "✓"
		}
		prefix := "  "
		label := clean(row.label)
		if browsing && !selected {
			label = m.rowNameStyle(row).Render(label)
		}
		if selected {
			prefix = "› "
		}
		title := prefix + marker + " " + label
		var line string
		if artistList {
			line = m.artistRow(title, row, width, selected)
		} else {
			line = m.mediaRow(title, row, width, selected)
		}
		if selected {
			style := m.styles.Selection
			if m.prompt != noPrompt || m.confirmation != nil {
				style = m.styles.InactiveSelection
			}
			// Render the highlight once so inline colors cannot break the background.
			line = style.Width(width).Render(line)
		}
		lines = append(lines, line)
	}
	return lines
}

func (m *model) artistRow(title string, row row, width int, selected bool) string {
	switch {
	case width >= 34:
		const countWidth = 22
		albums, songs := len(row.children), len(row.tracks)
		counts := fmt.Sprintf("Albums: %d - Songs: %d", albums, songs)
		if len(counts) > countWidth {
			counts = fmt.Sprintf("Alb: %d - Songs: %d", albums, songs)
		}
		counts = ansi.Truncate(counts, countWidth, "…")
		counts += strings.Repeat(" ", countWidth-ansi.StringWidth(counts))
		if !selected {
			counts = m.styles.Muted.Render(counts)
		}
		return appendRowColumn(title, counts, width)
	case width >= 21:
		counts := fmt.Sprintf("Songs: %d", len(row.tracks))
		if !selected {
			counts = m.styles.Muted.Render(counts)
		}
		return appendRowColumn(title, counts, width)
	default:
		return ansi.Truncate(title, width, "…")
	}
}

func (m *model) mediaRow(title string, row row, width int, selected bool) string {
	if m.prompt == palettePrompt {
		if row.detail != "" {
			title += "  ·  " + keymap.Button(clean(row.detail))
		}
		return ansi.Truncate(title, width, "…")
	}

	const timeWidth = 8
	separateColumn := row.children != nil || m.tab == songsTab || m.canSwitchTrackColumn()
	timed := len(row.tracks) > 0 || m.tab == queueTab
	mainWidth := width
	if timed && width >= 22 {
		mainWidth -= timeWidth + 2
	}
	detail := m.mediaRowDetail(row, selected)
	if detail != "" {
		switch {
		case separateColumn && mainWidth >= 16:
			title = alignMediaRow(title, detail, mainWidth)
		case !separateColumn:
			title += "  ·  " + detail
		}
	}
	if mainWidth == width {
		return ansi.Truncate(title, width, "…")
	}
	length := fmt.Sprintf("%*s", timeWidth, durationLabel(row.duration))
	if !selected && row.duration > 0 {
		length = m.styles.ProgressFilled.Render(length)
	}
	return appendRowColumn(title, length, width)
}

func appendRowColumn(left, right string, width int) string {
	room := width - ansi.StringWidth(right)
	left = ansi.Truncate(left, room-2, "…")
	gap := strings.Repeat(" ", room-ansi.StringWidth(left))
	return left + gap + right
}

func alignMediaRow(title, artist string, width int) string {
	// Long artist names must leave room for the title.
	artist = ansi.Truncate(artist, width*2/5, "…")
	room := width - ansi.StringWidth(artist)
	title = ansi.Truncate(title, room-2, "…")
	return title + strings.Repeat(" ", room-ansi.StringWidth(title)) + artist
}

func clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}
