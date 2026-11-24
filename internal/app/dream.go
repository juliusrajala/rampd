package app

import (
	tea "charm.land/bubbletea/v2"

	"rampd/internal/dream"
	"rampd/internal/music"
)

type dreamMsg struct {
	tracks []music.Track
	err    error
	seeded bool
}

func (m *model) generateDream() tea.Cmd {
	return m.dreamCommand(nil)
}

func (m *model) startDreamFromTray() tea.Cmd {
	if len(m.tray) == 0 {
		m.notice = "Add tracks to the Tray first."
		return nil
	}
	cmd := m.dreamCommand(m.trayTracks())
	if cmd != nil {
		m.tab = dreamTab
	}
	return cmd
}

func (m *model) dreamCommand(seeds []music.Track) tea.Cmd {
	if m.dreamLoading {
		return nil
	}
	if !m.libraryLoaded {
		m.notice = "Load the MPD library before starting Dream."
		return nil
	}
	m.dreamLoading = true
	m.notice = ""
	ctx, strategy, tracks, count := m.ctx, m.dreamStrategy, m.library, m.dreamCount
	return func() tea.Msg {
		generated, err := strategy.Generate(ctx, tracks, dream.Options{TrackCount: count, Seeds: seeds})
		return dreamMsg{tracks: generated, err: err, seeded: len(seeds) > 0}
	}
}
