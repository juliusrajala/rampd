package app

import (
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/keymap"
)

const sidebarCommandGap = 2
const sidebarCommandInputRow = 1 + navRows + sidebarCommandGap

func (m *model) sidebarCommandHeight() int {
	if m.sidebarWidth() == 0 {
		return 0
	}
	return max(0, m.height-sidebarCommandInputRow-1)
}

func (m *model) paletteInSidebar() bool {
	return m.prompt == palettePrompt && m.sidebarCommandHeight() > 0
}

func (m *model) paletteInMain() bool {
	return m.prompt == palettePrompt && !m.paletteInSidebar()
}

func (m *model) sidebarRows(tabs []string) []string {
	rows := m.navGroups(tabs)
	if m.help != noHelp {
		return rows
	}
	rows = append(rows, make([]string, sidebarCommandGap)...)
	width := m.innerWidth(sidePanelWidth)
	if m.paletteInSidebar() {
		rows = append(rows, m.input.View())
		return append(rows, m.paletteRows(width)...)
	}
	budget := m.height - sidebarCommandInputRow
	items := m.previewCommands(budget)
	return append(rows, m.renderCommands(items, width, -1, budget-len(items))...)
}

// Reserve space for the tools and Quit before using any space for view actions.
func (m *model) previewCommands(budget int) []row {
	var actions, extra, general []row
	for _, item := range m.commandRows() {
		if item.key == string(keymap.Refresh) {
			continue // Keep this maintenance action in search, not in the preview.
		}
		switch m.groupOf(item) {
		case commandGeneral:
			general = append(general, item)
			if item.key == string(keymap.Filter) {
				if key := m.keys.Key(keymap.Commands); key != "" {
					general = append(general, newRow(string(keymap.Commands), "Commands", key))
				}
			}
		case commandEdit:
			extra = append(extra, item)
		default:
			actions = append(actions, item)
		}
	}
	if len(general) >= budget {
		return general[:budget]
	}
	space := budget - len(general)
	if space > 1 {
		space-- // Leave a blank row before the general shortcuts.
	}
	if m.tab == trayTab {
		return append(m.trayPreview(actions, space), general...)
	}
	actions = actions[:min(len(actions), space)]
	// Make room for the group breaks before showing extra actions.
	if room := space - len(actions) - int(commandGroupCount-2); room > 0 {
		actions = append(actions, extra[:min(len(extra), room)]...)
	}
	return append(actions, general...)
}

// Keep both Tray groups visible when the sidebar cannot fit every action.
func (m *model) trayPreview(actions []row, budget int) []row {
	if len(actions) < budget || budget < 2 {
		return actions[:min(len(actions), budget)]
	}
	var edits, whole []row
	for _, item := range actions {
		if m.groupOf(item) == commandView {
			edits = append(edits, item)
		} else {
			whole = append(whole, item)
		}
	}
	if len(edits) == 0 || len(whole) == 0 {
		return actions[:budget]
	}
	if budget > 2 {
		budget-- // Keep a blank row between the two groups.
	}
	n := min(len(edits), max(budget/2, budget-len(whole)))
	return append(edits[:n], whole[:min(len(whole), budget-n)]...)
}

func (m *model) renderCommands(items []row, width, cursor, gaps int) []string {
	var lines []string
	for i, item := range items {
		if gaps > 0 && i > 0 && m.groupOf(item) != m.groupOf(items[i-1]) &&
			(m.groupOf(item) == commandGeneral || gaps > generalGap(items[i:])) {
			lines = append(lines, "")
			gaps--
		}
		key := ansi.Truncate(keymap.Button(item.detail), max(0, width-5), "…")
		if i == cursor {
			line := appendRowColumn("› "+item.label, key, width)
			lines = append(lines, m.styles.Selection.Width(width).Render(line))
		} else {
			label := item.label
			if m.groupOf(item) != commandGeneral {
				label = m.styles.ArtistName.Render(label)
			}
			lines = append(lines, appendRowColumn("  "+label, m.styles.Shortcut.Render(key), width))
		}
	}
	return lines
}

// Reserve the last gap for the boundary between view actions and general shortcuts.
func generalGap(items []row) int {
	if len(items) > 0 && isGeneral(keymap.Action(items[len(items)-1].key)) {
		return 1
	}
	return 0
}

func (m *model) paletteRows(width int) []string {
	p := &m.palettePage
	if len(p.rows) == 0 {
		return []string{m.styles.Muted.Render("No matching commands.")}
	}
	end := m.commandEnd(p, m.paletteHeight())
	return m.renderCommands(p.rows[p.top:end], width, p.cursor-p.top, end-p.top)
}

// The page stores only selectable commands. Count blank group rows when fitting it.
func (m *model) commandEnd(p *page, height int) int {
	end := p.top
	for used := 0; end < len(p.rows) && used < height; end++ {
		if end > p.top && m.groupOf(p.rows[end]) != m.groupOf(p.rows[end-1]) {
			if used+1 >= height {
				break
			}
			used++
		}
		used++
	}
	return end
}

func (m *model) fitPalette() {
	p, height := &m.palettePage, max(1, m.paletteHeight())
	fitList(p, height)
	for p.top < p.cursor && p.cursor >= m.commandEnd(p, height) {
		p.top++
	}
}

func (m *model) paletteHeight() int {
	if m.paletteInSidebar() {
		return m.sidebarCommandHeight()
	}
	return m.listHeight()
}

func (m *model) openPalette() tea.Cmd {
	m.palettePage = page{title: "Commands"}
	m.palettePage.setRows(m.commandRows())
	m.prompt = palettePrompt
	m.input.Prompt = ": "
	m.input.Placeholder = "Find a command"
	m.input.SetValue("")
	m.resizeInput()
	return m.input.Focus()
}

func (m *model) paletteKey(msg tea.KeyPressMsg) tea.Cmd {
	p := &m.palettePage
	switch msg.String() {
	case "up":
		p.move(-1)
	case "down":
		p.move(1)
	case "pgup":
		p.move(-m.paletteHeight())
	case "pgdown":
		p.move(m.paletteHeight())
	case "esc":
		m.closePalette()
	case "enter":
		if item, ok := p.selected(); ok {
			m.closePalette()
			return m.dispatch(keymap.Action(item.key))
		}
	default:
		return m.updateInput(msg)
	}
	return nil
}

func (m *model) closePalette() {
	m.prompt = noPrompt
	m.input.Blur()
}

type commandGroup int

const (
	commandView commandGroup = iota
	commandAdd
	commandEdit
	commandGeneral
	commandGroupCount
)

func (m *model) groupOf(item row) commandGroup {
	action := keymap.Action(item.key)
	if m.tab == trayTab {
		switch action {
		case keymap.MarkTray, keymap.Remove, keymap.MoveUp, keymap.MoveDown, keymap.AddPlaylist:
			return commandView
		case keymap.Shuffle, keymap.DedupeTray, keymap.Replace, keymap.SeedDream:
			return commandAdd
		}
	}
	switch action {
	case keymap.Cancel, keymap.SortAlbums, keymap.Open, keymap.First:
		return commandView
	case keymap.AddTray, keymap.CopyTray, keymap.Append, keymap.AppendAll, keymap.AddPlaylist, keymap.Save:
		return commandAdd
	default:
		if isGeneral(action) {
			return commandGeneral
		}
		return commandEdit
	}
}

func isGeneral(action keymap.Action) bool {
	switch action {
	case keymap.Filter, keymap.Commands, keymap.Help, keymap.Guide, keymap.Quit, keymap.Refresh:
		return true
	}
	return false
}

// Navigation and transport keys have their own controls. Keep them out of the palette.
func (m *model) commandRows() []row {
	var groups [commandGroupCount][]row
	add := func(action keymap.Action, label string) {
		item := newRow(string(action), label, m.keys.Key(action))
		group := m.groupOf(item)
		groups[group] = append(groups[group], item)
	}
	page := m.browsePage()
	item, selected := page.selected()
	if m.tab == trayTab && selected {
		add(keymap.MarkTray, "Mark")
	}
	add(keymap.Filter, "Filter")
	if page.filter != "" {
		add(keymap.Cancel, "Clear filter")
	}
	if m.onMediaList() {
		add(keymap.SortAlbums, m.nextMediaSort())
	} else if m.canSwitchTrackColumn() {
		add(keymap.SortAlbums, m.trackColumnCommand())
	}
	if selected && len(m.selectionTracks(item)) > 0 && m.tab != trayTab {
		add(keymap.AddTray, "Add to Tray")
	}
	switch m.tab {
	case dreamTab:
		add(keymap.First, "New preview")
		if len(m.dreamTracks) > 0 {
			add(keymap.Open, "Play Dream")
			add(keymap.Append, "Add to queue")
			add(keymap.Replace, "Replace queue")
			add(keymap.Save, "Save playlist")
		}
	case trayTab:
		if len(m.tray) > 1 {
			add(keymap.Shuffle, "Shuffle Tray")
			add(keymap.DedupeTray, "Remove duplicates")
		}
		add(keymap.CopyTray, "Copy queue")
		if len(m.tray) > 0 {
			add(keymap.Append, "Add to queue")
			add(keymap.Replace, "Replace queue")
			add(keymap.Save, "Save playlist")
			add(keymap.SeedDream, "Seed Dream")
		}
		if len(m.trayMarks) > 0 {
			add(keymap.Remove, "Remove marked")
			add(keymap.MoveUp, "Move marked ↑")
			add(keymap.MoveDown, "Move marked ↓")
		} else if selected {
			add(keymap.Remove, "Remove track")
			add(keymap.MoveUp, "Move track up")
			add(keymap.MoveDown, "Move track down")
		}
	case queueTab:
		add(keymap.CopyTray, "Copy to Tray")
		if selected {
			add(keymap.Open, "Play track")
			add(keymap.Remove, "Remove track")
			add(keymap.MoveUp, "Move up")
			add(keymap.MoveDown, "Move down")
		}
		if len(m.queue) > 0 {
			add(keymap.Clear, "Clear queue")
			add(keymap.Save, "Save playlist")
		}
		if len(m.queue) > 1 {
			add(keymap.Shuffle, "Shuffle queue")
		}
	case playlistsTab:
		if page.playlist == "" {
			add(keymap.NewPlaylist, "New playlist")
			if selected {
				add(keymap.Open, "Open playlist")
				add(keymap.Rename, "Rename")
				add(keymap.Clear, "Delete list")
			}
		} else if selected {
			add(keymap.Open, "Play track")
			add(keymap.Append, "Add to queue")
			add(keymap.AppendAll, "Queue all")
			add(keymap.MoveUp, "Move up")
			add(keymap.MoveDown, "Move down")
			add(keymap.Remove, "Remove track")
		}
	default:
		if selected {
			label := "Play track"
			if item.children != nil {
				label = "Open"
			}
			add(keymap.Open, label)
			add(keymap.Append, "Add to queue")
		}
	}
	if selected && len(m.selectionTracks(item)) > 0 {
		add(keymap.AddPlaylist, "Add to playlist")
	}
	add(keymap.Refresh, "Refresh")
	add(keymap.Help, "Keybindings")
	add(keymap.Guide, "Help")
	add(keymap.Quit, "Quit")
	var rows []row
	for _, group := range groups {
		rows = append(rows, group...)
	}
	return rows
}
