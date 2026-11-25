package app

import (
	"fmt"
	"slices"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/keymap"
	"rampd/internal/music"
)

type playlistsMsg struct {
	names []string
	seq   uint64
	err   error
}

type playlistMsg struct {
	name   string
	seq    uint64
	tracks []music.Track
	err    error
}

func (m *model) selectTab(tab int) tea.Cmd {
	m.tab = tab
	if tab == playlistsTab && len(m.tabs[tab]) == 1 {
		return m.fetchPlaylists()
	}
	return nil
}

func (m *model) fetchPlaylists() tea.Cmd {
	if !m.connected {
		return nil
	}
	m.playlistsLoading = true
	m.playlistsSeq++
	seq, ctx, server := m.playlistsSeq, m.ctx, m.server
	return func() tea.Msg {
		names, err := server.Playlists(ctx)
		return playlistsMsg{names, seq, err}
	}
}

func (m *model) refreshStoredPlaylists() tea.Cmd {
	if !m.connected || m.tab != playlistsTab && m.playlistsSeq == 0 {
		return nil
	}
	var contents tea.Cmd
	if stack := m.tabs[playlistsTab]; len(stack) > 1 {
		contents = m.openPlaylist(stack[1].playlist)
	}
	return tea.Batch(m.fetchPlaylists(), contents)
}

func (m *model) openPlaylist(name string) tea.Cmd {
	stack := m.tabs[playlistsTab]
	if len(stack) == 1 || stack[1].playlist != name {
		m.tabs[playlistsTab] = append(stack[:1], page{title: "Playlists / " + name, playlist: name})
	}
	m.playlistSeq++
	seq, ctx, server := m.playlistSeq, m.ctx, m.server
	m.playlistLoading = true
	return func() tea.Msg {
		tracks, err := server.Playlist(ctx, name)
		return playlistMsg{name, seq, tracks, err}
	}
}

func (m *model) updatePlaylists(msg playlistsMsg) tea.Cmd {
	if msg.seq != m.playlistsSeq {
		return nil
	}
	m.playlistsLoading = false
	if msg.err != nil {
		m.notice = fmt.Sprintf("Playlists: %v", msg.err)
		return nil
	}
	rows := make([]row, 0, len(msg.names))
	for _, name := range msg.names {
		rows = append(rows, newRow(name, name, ""))
	}
	sortRows(rows)
	stack := m.tabs[playlistsTab]
	stack[0].setRows(rows)
	if len(stack) > 1 && !slices.Contains(msg.names, stack[1].playlist) {
		m.tabs[playlistsTab] = stack[:1]
		m.playlistLoading = false
	}
	if len(msg.names) == 0 {
		return nil
	}
	return m.fetchPlaylistSummaries(msg.names, msg.seq)
}

func (m *model) onPlaylistList() bool {
	return m.tab == playlistsTab && len(m.tabs[playlistsTab]) == 1
}

func (m *model) updatePlaylist(msg playlistMsg) {
	if msg.seq != m.playlistSeq || len(m.tabs[playlistsTab]) < 2 {
		return
	}
	m.playlistLoading = false
	if msg.err != nil {
		m.notice = fmt.Sprintf("Playlist %s: %v", msg.name, msg.err)
		return
	}
	p := &m.tabs[playlistsTab][1]
	p.setRows(playlistRows(msg.tracks))
	p.tracks = msg.tracks
}

func playlistRows(tracks []music.Track) []row {
	rows := trackRows(tracks)
	occurrences := make(map[string]int)
	for pos, track := range tracks {
		// The key preserves focus through reorder. Position addresses the
		// native MPD command. Equal URIs can appear more than once.
		rows[pos].key += "\x00" + strconv.Itoa(occurrences[track.URI])
		rows[pos].position = pos
		occurrences[track.URI]++
	}
	return rows
}

func (m *model) afterPlaylistAction(a music.Action, err error) tea.Cmd {
	var notice string
	switch a.Operation {
	case music.SaveQueue, music.SaveTracks:
		notice = fmt.Sprintf("Saved playlist %q.", a.Name)
	case music.CreatePlaylist:
		notice = fmt.Sprintf("Created playlist %q.", a.Name)
	case music.RenamePlaylist:
		notice = fmt.Sprintf("Renamed playlist to %q.", a.NewName)
	case music.DeletePlaylist:
		notice = fmt.Sprintf("Deleted playlist %q.", a.Name)
	case music.AddToPlaylist:
		notice = fmt.Sprintf("Added selection to %q.", a.Name)
	case music.MovePlaylistTrack, music.DeletePlaylistTrack:
		notice = fmt.Sprintf("Updated playlist %q.", a.Name)
	default:
		return nil
	}
	if err == nil {
		m.notice = notice
	}
	// Refresh even on failure. MPD can stop a command list partway through,
	// or reject a position because another client changed the playlist.
	var contents tea.Cmd
	stack := m.tabs[playlistsTab]
	if len(stack) > 1 && stack[1].playlist == a.Name {
		contents = m.openPlaylist(a.Name)
	}
	return tea.Batch(m.fetchPlaylists(), contents)
}

func (m *model) addSelectionToPlaylist() tea.Cmd {
	item, ok := m.selected()
	if !ok {
		return nil
	}
	tracks := m.selectionTracks(item)
	if len(tracks) == 0 {
		return nil
	}
	return m.namePlaylist(music.Action{Operation: music.AddToPlaylist, Tracks: tracks})
}

func (m *model) selectionTracks(item row) []music.Track {
	if m.tab != queueTab {
		return item.tracks
	}
	if track, ok := m.rowTrack(item); ok {
		return []music.Track{track}
	}
	return nil
}

func (m *model) editPlaylistTrack(key keymap.Action, item row) tea.Cmd {
	p := m.page()
	a := music.Action{
		Operation: music.MovePlaylistTrack, Name: p.playlist,
		Position: item.position, Expected: p.tracks, Delta: 1,
	}
	switch key {
	case keymap.Remove:
		a.Operation = music.DeletePlaylistTrack
		m.confirmAction("Remove this track from the stored playlist?", a)
		return nil
	case keymap.MoveUp:
		a.Delta = -1
	}
	if target := a.Position + a.Delta; target < 0 || target >= len(p.tracks) {
		return nil
	}
	return m.act(a)
}
