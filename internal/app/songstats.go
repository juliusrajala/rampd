package app

import (
	"context"
	"fmt"
	"strconv"
	"time"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/music"
)

type playHistoryReader interface {
	PlayHistory(context.Context, string) (music.PlayHistory, error)
}

type songStatsCache struct {
	result   songStatsMsg
	pending  string
	revision uint64
}

type songStatsMsg struct {
	uri      string
	revision uint64
	readAt   time.Time
	history  music.PlayHistory
	err      error
}

func (s *songStatsCache) invalidate() {
	s.result = songStatsMsg{}
	// Keep the in-flight slot until its result arrives.
	s.revision++
}

func (s *songStatsCache) finish(msg songStatsMsg) {
	s.pending = ""
	if msg.revision == s.revision {
		s.result = msg
	}
}

func (m *model) requestSongStats() tea.Cmd {
	reader, ok := m.server.(playHistoryReader)
	if !ok || !m.connected || !m.showArtworkPanel() || m.songStats.pending != "" {
		return nil
	}
	item, ok := m.selected()
	if !ok {
		return nil
	}
	track, ok := m.rowTrack(item)
	if !ok || track.URI == "" {
		return nil
	}
	if cached := m.songStats.result; cached.uri == track.URI && time.Since(cached.readAt) < time.Minute {
		return nil
	}
	uri, ctx, revision := track.URI, m.ctx, m.songStats.revision
	m.songStats.pending = uri
	return func() tea.Msg {
		history, err := reader.PlayHistory(ctx, uri)
		return songStatsMsg{uri, revision, time.Now(), history, err}
	}
}

func (m *model) songStatsFields(uri string, now time.Time) []detailField {
	cached := m.songStats.result
	if cached.uri != uri || uri == "" {
		return nil
	}
	if cached.err != nil {
		return []detailField{{"History", "Unavailable"}}
	}
	last := "Not recorded"
	if !cached.history.LastPlayed.IsZero() {
		last = cached.history.LastPlayed.UTC().Format("2006-01-02")
	}
	year := now.UTC().Year()
	return []detailField{
		{"Last played", last},
		{fmt.Sprintf("%d plays", year), strconv.FormatUint(cached.history.Yearly[year], 10)},
		{fmt.Sprintf("%d plays", year-1), strconv.FormatUint(cached.history.Yearly[year-1], 10)},
	}
}
