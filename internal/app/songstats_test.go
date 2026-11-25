package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/music"
)

type songStatsBackend struct {
	fakeBackend
	reads   []string
	history music.PlayHistory
}

func (s *songStatsBackend) PlayHistory(_ context.Context, uri string) (music.PlayHistory, error) {
	s.reads = append(s.reads, uri)
	return s.history, nil
}

func TestSongStatsRefreshAfterRecording(t *testing.T) {
	now := time.Now().UTC()
	server := &songStatsBackend{history: music.PlayHistory{
		LastPlayed: now, Yearly: map[int]uint64{now.Year() - 1: 2, now.Year() - 2: 8},
	}}
	m := New(context.Background(), server, Options{Artwork: true}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 112, Height: 22})
	track := fixtureTracks[1]
	track.Added, track.Modified = now, now
	m.Update(prepareLibrary([]music.Track{track}, nil))
	m.connected = true
	read := m.requestSongStats()
	if read == nil || m.requestSongStats() != nil {
		t.Fatal("expected only one read in flight")
	}
	m.songStats.finish(read().(songStatsMsg))
	if m.requestSongStats() != nil {
		t.Fatal("a fresh result should be reused")
	}
	view := strings.Join(strings.Fields(ansi.Strip(m.View().Content)), " ")
	for _, want := range []string{"Last played " + now.Format("2006-01-02"),
		fmt.Sprintf("%d plays 0", now.Year()), fmt.Sprintf("%d plays 2", now.Year()-1)} {
		if !strings.Contains(view, want) {
			t.Errorf("missing %q from short selection preview", want)
		}
	}
	if strings.Contains(view, fmt.Sprintf("%d plays", now.Year()-2)) {
		t.Fatal("only the last two calendar years should be shown")
	}

	server.history.Yearly[now.Year()] = 1
	m.update(playRecorded{uri: track.URI})
	read = m.requestSongStats()
	if read == nil {
		t.Fatal("recording a play should invalidate the selected song statistics")
	}
	m.songStats.finish(read().(songStatsMsg))
	if !slices.Equal(server.reads, []string{track.URI, track.URI}) || m.songStats.result.history.Yearly[now.Year()] != 1 {
		t.Fatal("expected the recorded song's statistics to refresh")
	}
}

func TestSongStatsDistinguishMissingAndUnavailable(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m.songStats.result = songStatsMsg{uri: "one"}
	want := []detailField{{"Last played", "Not recorded"}, {"2026 plays", "0"}, {"2025 plays", "0"}}
	if got := m.songStatsFields("one", at); !slices.Equal(got, want) {
		t.Fatalf("unexpected empty history: %v", got)
	}
	if got := m.songStatsFields("another", at); len(got) != 0 {
		t.Fatalf("song got another track's history: %v", got)
	}
	m.songStats.result.err = errors.New("stickers disabled")
	if got := m.songStatsFields("one", at); !slices.Equal(got, []detailField{{"History", "Unavailable"}}) {
		t.Fatalf("unexpected error display: %v", got)
	}
}
