package app

import (
	tea "charm.land/bubbletea/v2"

	"rampd/internal/keymap"
)

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
	switch msg.String() {
	case "up":
		m.page().move(-1)
	case "down":
		m.page().move(1)
	case "pgup":
		m.page().move(-m.listHeight())
	case "pgdown":
		m.page().move(m.listHeight())
	case "esc":
		m.closePalette()
	case "enter":
		if item, ok := m.selected(); ok {
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

func (m *model) commandRows() []row {
	var rows []row
	add := func(action keymap.Action, label string) {
		if !m.trayActionAvailable(action) {
			return
		}
		rows = append(rows, newRow(string(action), label, m.keys.Key(action)))
	}
	add(keymap.Filter, "Filter this list")
	add(keymap.Refresh, "Refresh this view")
	add(keymap.PlayPause, "Play or pause")
	add(keymap.Next, "Next track")
	add(keymap.Previous, "Previous track")
	add(keymap.SeekBack, "Seek backward five seconds")
	add(keymap.SeekForward, "Seek forward five seconds")
	add(keymap.Random, "Toggle random playback")
	add(keymap.Repeat, "Toggle repeat playback")

	item, selected := m.selected()
	if m.canSwitchTrackColumn() {
		add(keymap.SortAlbums, m.trackColumnCommand())
	}
	switch m.tab {
	case dreamTab:
		add(keymap.First, "Generate a Dream preview")
		if len(m.dreamTracks) > 0 {
			add(keymap.Open, "Play Dream session")
			add(keymap.Append, "Append Dream to queue")
			add(keymap.Replace, "Replace queue with Dream")
			add(keymap.Save, "Save Dream as playlist")
		}
	case trayTab:
		add(keymap.TrayHints, "Switch Tray mode")
		add(keymap.CopyTray, "Copy live queue to Tray")
		if len(m.tray) > 1 {
			add(keymap.ShuffleTray, "Shuffle Tray")
		}
		if len(m.tray) > 0 {
			add(keymap.Append, "Append Tray to queue")
			add(keymap.Replace, "Replace queue with Tray")
			add(keymap.Save, "Save Tray as playlist")
			add(keymap.SeedDream, "Generate Dream from Tray")
		}
		if selected {
			add(keymap.MarkTray, "Toggle mark on highlighted Tray track")
		}
		if len(m.trayMarks) > 0 {
			add(keymap.Remove, "Remove marked Tray tracks")
			add(keymap.MoveUp, "Move marked Tray tracks up")
			add(keymap.MoveDown, "Move marked Tray tracks down")
		} else if selected {
			add(keymap.Remove, "Remove highlighted Tray track")
			add(keymap.MoveUp, "Move Tray track up")
			add(keymap.MoveDown, "Move Tray track down")
		}
	case queueTab:
		add(keymap.CopyTray, "Copy live queue to Tray")
		if selected {
			add(keymap.Open, "Play selected queue track")
			add(keymap.Remove, "Remove selected queue track")
			add(keymap.MoveUp, "Move queue track up")
			add(keymap.MoveDown, "Move queue track down")
		}
		if len(m.queue) > 0 {
			add(keymap.Clear, "Clear active queue")
			add(keymap.Save, "Save queue as playlist")
		}
		if len(m.queue) > 1 {
			add(keymap.Shuffle, "Shuffle active queue")
		}
	case playlistsTab:
		if m.page().playlist == "" {
			add(keymap.NewPlaylist, "Create a playlist")
			if selected {
				add(keymap.Open, "Open selected playlist")
				add(keymap.Rename, "Rename selected playlist")
				add(keymap.Clear, "Delete selected playlist")
			}
		} else if selected {
			add(keymap.Open, "Play selected playlist track")
			add(keymap.Append, "Append selected track to queue")
			add(keymap.AppendAll, "Append playlist to queue")
			add(keymap.MoveUp, "Move playlist track up")
			add(keymap.MoveDown, "Move playlist track down")
			add(keymap.Remove, "Remove selected playlist track")
		}
	default:
		if m.onMediaList() {
			add(keymap.SortAlbums, m.nextMediaSort())
		}
		if selected {
			label := "Play selected track"
			if item.children != nil {
				label = "Open selection"
			}
			add(keymap.Open, label)
			add(keymap.Append, "Append selection to queue")
		}
	}
	if selected && len(m.selectionTracks(item)) > 0 {
		add(keymap.AddPlaylist, "Add selection to stored playlist")
		if m.tab != trayTab {
			add(keymap.AddTray, "Add selection to Tray")
		}
	}
	for i, action := range tabActions {
		add(action, "Switch to "+tabNames[i])
	}
	add(keymap.Help, "Show key bindings")
	add(keymap.Quit, "Quit")
	return rows
}
