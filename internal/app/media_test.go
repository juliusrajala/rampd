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

func TestTrayAndDreamSwitchColumnWithoutReordering(t *testing.T) {
	tracks := []music.Track{
		{URI: "2", Title: "Second", Artist: "Violet", Album: "Bright", Duration: 65 * time.Second},
		{URI: "1", Title: "First", Artist: "Plum", Album: "Bright", Duration: 90 * time.Second},
	}
	m := New(context.Background(), &fakeBackend{}, Options{InitialTray: tracks}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 25})
	m.Update(dreamMsg{tracks: tracks})
	m.Update(keyRune('4'))
	m.page().setFilter("bright")
	m.Update(keyRune('j'))
	m.Update(keyRune('v'))

	assertColumn := func(want, hidden string) {
		t.Helper()
		if got := ansi.Strip(m.viewRows()[0]); !strings.Contains(got, want) || strings.Contains(got, hidden) || !strings.HasSuffix(got, "1:05") {
			t.Fatalf("wrong visible column: %q", got)
		}
	}
	assertColumn("Violet", "Bright")
	m.Update(keyRune('o'))
	assertColumn("Bright", "Violet")
	if r, ok := m.selected(); !ok || r.label != "First" || m.page().filter != "bright" || len(m.trayMarks) != 1 {
		t.Fatal("column switch changed Tray selection, filter, or marks")
	}

	m.Update(keyRune('7'))
	assertColumn("Violet", "Bright")
	m.Update(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "show album"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assertColumn("Bright", "Violet")
	m.Update(keyRune('o'))
	assertColumn("Violet", "Bright")
	m.Update(keyRune('4'))
	assertColumn("Bright", "Violet")
	if !slices.Equal(m.trayTracks(), tracks) || !slices.Equal(m.dreamTracks, tracks) {
		t.Fatal("column switch changed track order")
	}
}

func TestQueueAndPlaylistTracksUseSessionColumns(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 25})
	track := music.Track{URI: "track", Title: "Song", Artist: "Artist", Album: "Album", Duration: time.Minute}
	m.queue = []music.Entry{{ID: 8, Track: track}}
	m.buildQueuePage()
	m.Update(keyRune('5'))
	check := func(want, hidden string) {
		t.Helper()
		line := ansi.Strip(m.viewRows()[0])
		if !strings.Contains(line, "Song") || !strings.Contains(line, "  "+want) ||
			strings.Contains(line, hidden) || strings.Contains(line, "·") || !strings.HasSuffix(line, "1:00") {
			t.Fatalf("wrong track layout: %q", line)
		}
	}
	check("Artist", "Album")
	m.Update(keyRune('o'))
	check("Album", "Artist")

	m.Update(keyRune('6'))
	m.openPlaylist("Test")
	m.updatePlaylist(playlistMsg{seq: m.playlistSeq, tracks: []music.Track{track}})
	check("Artist", "Album")
	m.Update(keyRune('o'))
	check("Album", "Artist")
}

func TestMediaColorsMatchBetweenRowsAndSelection(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 25})
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('1'))
	row, ok := m.selected()
	if !ok {
		t.Fatal("no selected track")
	}
	lines := m.viewRows()
	width := m.innerWidth(m.bodyWidth())
	if line := lines[0]; line != m.styles.Selection.Width(width).Render(ansi.Strip(line)) {
		t.Fatalf("focused row should have one background across its full width: %q", line)
	}
	next := m.page().rows[1]
	if !strings.Contains(lines[1], m.styles.SongName.Render(next.label)) ||
		!strings.Contains(lines[1], m.styles.ArtistName.Render(next.detail)) {
		t.Fatalf("unfocused row lost its media colors: %q", lines[1])
	}
	selection := strings.Join(m.artworkRows(), "\n")
	for _, want := range []string{m.styles.SongName.Render(row.label), m.styles.ArtistName.Render(row.detail), m.styles.AlbumName.Render("Bright")} {
		if !strings.Contains(selection, want) {
			t.Errorf("missing media color in Selection: %q", want)
		}
	}
}
