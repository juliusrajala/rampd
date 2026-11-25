package app

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/music"
)

func TestPlaylistSummaryRanksGenresAndDeduplicatesArtists(t *testing.T) {
	tracks := []music.Track{
		{Artist: "Zeta", Genre: "Rock", Duration: time.Minute},
		{Artist: "Zeta", Genre: "rock", Duration: time.Minute},
		{AlbumArtist: "Bravo", Genre: "Pop", Duration: time.Minute},
		{Artist: "Alpha", Genre: "Jazz", Duration: time.Minute},
		{Artist: "Alpha", Genre: "Soul"},
	}
	summary := summarizePlaylist(tracks)
	if !slices.Equal(summary.genres, []string{"Rock", "Jazz", "Pop"}) ||
		!slices.Equal(summary.artists, []string{"Alpha", "Bravo", "Zeta"}) ||
		summary.count != 5 || summary.duration != 4*time.Minute || summary.timed != 4 {
		t.Fatalf("incorrect summary: %+v", summary)
	}
}

func TestPlaylistOverviewAndSelectionPreview(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 132, Height: 24})
	m.tab = playlistsTab
	m.Update(playlistsMsg{names: []string{"Live Set", "Morning"}})
	m.browsePage().move(1)
	m.Update(playlistSummariesMsg{items: map[string]*playlistSummary{
		"Morning": {genres: []string{"Rock", "Jazz", "Pop"}, artists: []string{"Alpha", "Zeta"}, count: 3, timed: 3, duration: 270 * time.Second},
	}})
	selected, ok := m.selected()
	if !ok || selected.key != "Morning" {
		t.Fatal("loading summaries changed the selection")
	}
	row := ansi.Strip(m.viewRows()[1])
	if !strings.Contains(row, "Rock, Jazz, Pop") || !strings.Contains(row, "3 tracks") || !strings.HasSuffix(row, "4:30") {
		t.Fatalf("missing playlist columns: %q", row)
	}
	var preview []string
	for _, line := range strings.Split(m.View().Content, "\n") {
		right := ansi.Cut(line, m.sidebarWidth()+m.bodyWidth()+panelGap, m.width)
		if right = strings.TrimSpace(ansi.Strip(right)); right != "" {
			preview = append(preview, right)
		}
	}
	if len(preview) != artRowCount+3 || !strings.Contains(preview[0], "▀") ||
		!slices.Equal(preview[artRowCount:], []string{"Morning", "Alpha", "Zeta"}) {
		t.Fatalf("expected a cover above the name and artists: %q", preview)
	}
	m.Update(tea.WindowSizeMsg{Width: 132, Height: 15})
	preview = m.playlistPreview(selected)
	for i := range preview {
		preview[i] = ansi.Strip(preview[i])
	}
	if !slices.Equal(preview, []string{"Morning", "Alpha", "Zeta"}) {
		t.Fatalf("short previews should keep the artists instead of the cover: %q", preview)
	}
	m.Update(tea.WindowSizeMsg{Width: 42, Height: 24})
	row = ansi.Strip(m.viewRows()[1])
	if !strings.Contains(row, "Morning") || !strings.Contains(row, "3 tracks") || !strings.HasSuffix(row, "4:30") || strings.Contains(row, "Rock") {
		t.Fatalf("narrow layout did not give name and totals priority: %q", row)
	}
}
