package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/keymap"
	"rampd/internal/music"
)

func (m *model) key(msg tea.KeyPressMsg) tea.Cmd {
	raw := msg.String()
	if raw == "ctrl+c" {
		return tea.Quit
	}
	if m.prompt != noPrompt {
		return m.promptKey(msg)
	}
	if m.confirmation != nil {
		run := m.confirmation.run
		m.confirmation = nil
		if raw == "y" && run != nil {
			return run()
		}
		return nil
	}
	key := m.keys.Action(raw)
	if m.showHelp {
		m.showHelp = false
		if key != keymap.Quit {
			return nil
		}
	}
	return m.dispatch(key)
}

func (m *model) dispatch(key keymap.Action) tea.Cmd {
	for tab, action := range tabActions {
		if key == action {
			return m.selectTab(tab)
		}
	}
	switch key {
	case keymap.Quit:
		return tea.Quit
	case keymap.Help:
		m.showHelp = true
	case keymap.NextTab:
		return m.selectTab((m.tab + 1) % len(tabNames))
	case keymap.PreviousTab:
		return m.selectTab((m.tab + len(tabNames) - 1) % len(tabNames))
	case keymap.Down:
		m.page().move(1)
	case keymap.Up:
		m.page().move(-1)
	case keymap.PageDown:
		m.page().move(m.listHeight())
	case keymap.PageUp:
		m.page().move(-m.listHeight())
	case keymap.First:
		if m.tab == dreamTab {
			return m.generateDream()
		}
		m.page().cursor = 0
	case keymap.Home:
		m.page().cursor = 0
	case keymap.Last:
		m.page().move(len(m.page().rows))
	case keymap.Back, keymap.Cancel:
		if key == keymap.Cancel && m.page().filter != "" {
			m.page().setFilter("")
			return nil
		}
		if stack := m.tabs[m.tab]; len(stack) > 1 {
			m.tabs[m.tab] = stack[:len(stack)-1]
			if m.tab == playlistsTab && len(m.tabs[m.tab]) == 1 {
				return m.fetchPlaylists()
			}
		}
	case keymap.Forward, keymap.Open:
		return m.openSelected(key == keymap.Open)
	case keymap.Append:
		if m.tab == trayTab && len(m.tray) > 0 {
			return m.act(music.Action{Operation: music.Append, Tracks: m.trayTracks()})
		}
		if m.tab == dreamTab && len(m.dreamTracks) > 0 {
			return m.act(music.Action{Operation: music.Append, Tracks: m.dreamTracks})
		}
		if row, ok := m.selected(); ok && len(row.tracks) > 0 {
			return m.act(music.Action{Operation: music.Append, Tracks: row.tracks})
		}
	case keymap.Filter:
		return m.focusFilter()
	case keymap.Commands:
		return m.openPalette()
	case keymap.Refresh:
		if m.tab == playlistsTab {
			if name := m.page().playlist; name != "" {
				return m.openPlaylist(name)
			}
			return m.fetchPlaylists()
		}
		return m.fetchLibrary()
	case keymap.NewPlaylist:
		if m.tab == playlistsTab && m.page().playlist == "" {
			return m.namePlaylist(music.Action{Operation: music.CreatePlaylist})
		}
	case keymap.Rename:
		if item, ok := m.selected(); m.tab == playlistsTab && m.page().playlist == "" && ok {
			return m.namePlaylist(music.Action{Operation: music.RenamePlaylist, Name: item.key})
		}
	case keymap.AddPlaylist:
		return m.addSelectionToPlaylist()
	case keymap.AddTray:
		m.addTraySelection()
	case keymap.CopyTray:
		if m.tab == queueTab || m.tab == trayTab {
			return m.copyQueueToTray()
		}
	case keymap.SeedDream:
		if m.tab == trayTab {
			return m.startDreamFromTray()
		}
	case keymap.AppendAll:
		if m.tab == playlistsTab && len(m.page().tracks) > 0 {
			return m.act(music.Action{Operation: music.Append, Tracks: m.page().tracks})
		}
	case keymap.Save:
		switch {
		case m.tab == trayTab && len(m.tray) > 0:
			return m.namePlaylist(music.Action{Operation: music.SaveTracks, Tracks: m.trayTracks()})
		case m.tab == dreamTab && len(m.dreamTracks) > 0:
			return m.namePlaylist(music.Action{Operation: music.SaveTracks, Tracks: m.dreamTracks})
		case m.tab == queueTab && len(m.queue) > 0:
			return m.namePlaylist(music.Action{Operation: music.SaveQueue})
		}
	case keymap.PlayPause:
		return m.act(music.Action{Operation: music.Toggle})
	case keymap.Next:
		return m.act(music.Action{Operation: music.Next})
	case keymap.Previous:
		return m.act(music.Action{Operation: music.Previous})
	case keymap.SeekBack:
		return m.act(music.Action{Operation: music.Seek, Delta: -5})
	case keymap.SeekForward:
		return m.act(music.Action{Operation: music.Seek, Delta: 5})
	case keymap.Random:
		return m.act(music.Action{Operation: music.Random})
	case keymap.Repeat:
		return m.act(music.Action{Operation: music.Repeat})
	case keymap.Remove, keymap.MoveDown, keymap.MoveUp:
		item, ok := m.selected()
		if m.tab == trayTab && ok {
			m.editTray(key, item)
		}
		if m.tab == playlistsTab && m.page().playlist != "" && ok {
			return m.editPlaylistTrack(key, item)
		}
		if m.tab == queueTab && ok {
			a := music.Action{Operation: music.Move, ID: item.id, Delta: 1}
			switch key {
			case keymap.Remove:
				a.Operation = music.Remove
			case keymap.MoveUp:
				a.Delta = -1
			}
			return m.act(a)
		}
	case keymap.SortAlbums:
		if m.onMediaList() {
			m.toggleMediaSort()
		}
	case keymap.Shuffle:
		if m.tab == queueTab && len(m.queue) > 1 {
			return m.act(music.Action{Operation: music.Shuffle})
		}
	case keymap.Clear:
		if m.tab == queueTab && len(m.queue) > 0 {
			m.confirmAction("Clear the active queue?", music.Action{Operation: music.Clear})
		} else if item, ok := m.selected(); m.tab == playlistsTab && m.page().playlist == "" && ok {
			m.confirmAction("Delete playlist "+clean(item.label)+"?",
				music.Action{Operation: music.DeletePlaylist, Name: item.key})
		}
	case keymap.Replace:
		if m.tab == trayTab && len(m.tray) > 0 {
			m.confirmTrayReplacement()
		} else if m.tab == dreamTab && len(m.dreamTracks) > 0 {
			m.confirmAction("Replace the active queue with this Dream?",
				music.Action{Operation: music.ReplaceAndPlay, Tracks: m.dreamTracks})
		}
	}
	return nil
}

func (m *model) confirmAction(message string, action music.Action) {
	m.confirmation = &confirmation{message, func() tea.Cmd { return m.act(action) }}
}

func (m *model) openSelected(play bool) tea.Cmd {
	if m.tab == trayTab {
		return nil
	}
	if m.tab == dreamTab {
		if !play {
			return nil
		}
		if len(m.dreamTracks) == 0 {
			return m.generateDream()
		}
		return m.act(music.Action{Operation: music.AppendAndPlay, Tracks: m.dreamTracks})
	}

	row, ok := m.selected()
	if !ok {
		return nil
	}
	switch {
	case m.tab == playlistsTab && len(m.tabs[playlistsTab]) == 1:
		return m.openPlaylist(row.key)
	case row.children != nil:
		child := page{title: m.page().title + " / " + row.label, parentKey: row.key}
		child.setRows(row.children)
		m.tabs[m.tab] = append(m.tabs[m.tab], child)
	case m.tab == queueTab && play:
		return m.act(music.Action{Operation: music.Play, ID: row.id})
	case play && len(row.tracks) > 0:
		return m.act(music.Action{Operation: music.AppendAndPlay, Tracks: row.tracks})
	}
	return nil
}

func (m *model) namePlaylist(action music.Action) tea.Cmd {
	m.prompt = playlistNamePrompt
	m.saveAction = action
	m.input.Prompt = "Save as: "
	m.input.Placeholder = "New playlist name"
	value := ""
	switch action.Operation {
	case music.CreatePlaylist:
		m.input.Prompt = "Create: "
	case music.RenamePlaylist:
		m.input.Prompt = "Rename to: "
		value = action.Name
	case music.AddToPlaylist:
		m.input.Prompt = "Add to: "
		m.input.Placeholder = "Existing playlist name"
	}
	m.input.SetValue(value)
	m.input.CursorEnd()
	m.resizeInput()
	return m.input.Focus()
}

func (m *model) promptKey(msg tea.KeyPressMsg) tea.Cmd {
	mode := m.prompt
	if mode == palettePrompt {
		return m.paletteKey(msg)
	}
	switch msg.String() {
	case "esc", "enter":
		m.prompt = noPrompt
		m.input.Blur()
	default:
		return m.updateInput(msg)
	}

	if mode == searchPrompt {
		if msg.String() == "esc" {
			m.page().setFilter("")
		}
		return nil
	}
	if msg.String() == "esc" {
		return nil
	}
	text := strings.TrimSpace(m.input.Value())
	if text == "" {
		m.notice = "Enter a playlist name."
		return nil
	}
	action := m.saveAction
	if action.Operation == music.RenamePlaylist {
		if action.Name == text {
			return nil
		}
		action.NewName = text
	} else {
		action.Name = text
	}
	return m.act(action)
}
