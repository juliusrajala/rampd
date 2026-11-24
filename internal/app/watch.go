package app

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/music"
)

type watchMsg struct {
	changes music.Changes
	stopped bool
}

type watchRetry struct{}
type progressTick time.Time

func (m *model) runWatch() tea.Cmd {
	ctx, server, events := m.ctx, m.server, m.events
	return func() tea.Msg {
		emit := func(msg watchMsg) {
			select {
			case events <- msg:
			case <-ctx.Done():
			}
		}
		// A failed or unsupported idle request uses the same polling fallback.
		_ = server.Watch(ctx, func(changes music.Changes) { emit(watchMsg{changes: changes}) })
		emit(watchMsg{stopped: true})
		return nil
	}
}

func (m *model) awaitWatch() tea.Cmd {
	ctx, events := m.ctx, m.events
	return func() tea.Msg {
		select {
		case msg := <-events:
			return msg
		case <-ctx.Done():
			return nil
		}
	}
}

func (m *model) handleWatch(msg watchMsg) tea.Cmd {
	next := m.awaitWatch()
	if msg.stopped {
		m.watching = false
		wait := m.watchBackoff
		m.watchBackoff = min(15*time.Second, wait*2)
		retry := tea.Tick(wait, func(time.Time) tea.Msg { return watchRetry{} })
		if m.connected {
			return tea.Batch(next, retry, m.requestSnapshot())
		}
		return tea.Batch(next, retry)
	}

	changes := msg.changes
	if changes&music.SessionReady != 0 {
		m.watching = true
		m.watchBackoff = time.Second
		m.forceQueue = true // The server may have restarted with the same version number.
		changes |= music.PlaybackChanged | music.DatabaseChanged | music.StoredPlaylistsChanged
	}
	var snapshot, library, playlists tea.Cmd
	if changes&music.PlaybackChanged != 0 {
		snapshot = m.requestSnapshot()
	}
	if changes&music.DatabaseChanged != 0 {
		library = m.invalidateLibrary()
	}
	if changes&music.StoredPlaylistsChanged != 0 {
		playlists = m.refreshStoredPlaylists()
	}
	return tea.Batch(next, snapshot, library, playlists)
}

func (m *model) requestSnapshot() tea.Cmd {
	if m.busy {
		m.needSnapshot = true
		return nil
	}
	m.busy = true
	return m.stateCommand(nil)
}

func (m *model) invalidateLibrary() tea.Cmd {
	if m.libraryLoading {
		m.libraryInvalidated = true
		return nil
	}
	return m.fetchLibrary()
}

func nextProgressTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg { return progressTick(t) })
}

func (m *model) playbackElapsed() time.Duration {
	elapsed := m.status.Elapsed
	if m.connected && m.status.State == "play" && !m.sampledAt.IsZero() {
		elapsed += max(0, m.clockTime.Sub(m.sampledAt))
	}
	if m.status.Duration > 0 {
		elapsed = min(elapsed, m.status.Duration)
	}
	return max(0, elapsed)
}
