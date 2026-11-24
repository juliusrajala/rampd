package app

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/music"
)

type artworkBackend struct {
	fakeBackend
	requests []string
}

func (b *artworkBackend) Artwork(_ context.Context, uri string) ([]byte, error) {
	b.requests = append(b.requests, uri)
	return nil, nil
}

func TestArtworkStatesUseTheSameTile(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.connected = true
	m.page().setRows(trackRows(fixtureTracks))
	uri := fixtureTracks[0].URI
	assertTile := func(want string, artRow int) {
		t.Helper()
		rows := m.artworkRows()
		if len(rows) <= artRowCount || ansi.Strip(rows[artRowCount]) != fixtureTracks[0].Title {
			t.Fatalf("expected a %d-row tile above the title, got %q", artRowCount, rows)
		}
		tile := rows[:artRowCount]
		for _, row := range tile {
			if got := ansi.StringWidth(row); got != artPanelWidth {
				t.Fatalf("wrong tile width: %d cells", got)
			}
		}
		if !strings.Contains(ansi.Strip(tile[artRow]), want) {
			t.Fatalf("expected %q on tile row %d: %q", want, artRow, tile[artRow])
		}
	}
	assertRoundedMatte := func() {
		t.Helper()
		rows := m.artworkRows()
		for i, glyph := range map[int]string{0: "▄", artRowCount - 1: "▀"} {
			line := ansi.Strip(rows[i])
			if !strings.HasPrefix(line, glyph) || !strings.HasSuffix(line, glyph) {
				t.Errorf("expected rounded tile edge: %q", line)
			}
		}
	}
	assertTile("Loading artwork…", artRowCount-1)
	assertRoundedMatte()
	m.artworkCache = []artworkMsg{{uri: uri}}
	assertTile("▀ ▀", (artRowCount-4)/2)
	assertRoundedMatte()
	if strings.Contains(ansi.Strip(strings.Join(m.artworkRows(), "\n")), "No artwork") {
		t.Fatal("placeholder should not have a missing artwork caption")
	}
	m.artworkCache[0].lines = []string{"COVER", "COVER"}
	assertTile("COVER", (artRowCount-2)/2)
}

func TestSelectionRowsShowDetailsForEachKind(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(prepareLibrary(fixtureTracks, nil))
	check := func(want ...string) {
		t.Helper()
		rows := m.artworkRows()
		if len(rows) < artRowCount+len(want) {
			t.Fatalf("not enough detail rows: %q", rows)
		}
		for i, text := range want {
			if got := strings.Join(strings.Fields(ansi.Strip(rows[artRowCount+i])), " "); got != text {
				t.Errorf("detail row %d = %q, want %q", i, got, text)
			}
		}
	}
	m.Update(keyRune('3'))
	check("Plum", "Albums 1", "Songs 1")
	m.Update(keyRune('2'))
	check("Bright", "Artist Plum", "Length —", "Songs 1")
	m.Update(keyRune('1'))
	check("First", "Artist Violet", "Album Bright", "Length —", "Genre Jazz")
}

func TestSelectionSpacingKeepsPrimaryDetails(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('1'))

	m.Update(tea.WindowSizeMsg{Width: 90, Height: 25})
	rows := m.artworkRows()
	if rows[0] != "" || rows[artRowCount+1] != "" || ansi.Strip(rows[artRowCount+2]) != "First" {
		t.Fatal("roomy panel should pad above the cover and its text")
	}

	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	rows = m.artworkRows()
	if rows[0] != "" || ansi.Strip(rows[artRowCount+1]) != "First" ||
		!strings.Contains(ansi.Strip(rows[m.bodyHeight()-1]), "Length") {
		t.Fatal("shorter panel should give primary details priority over extra padding")
	}
}

func TestSelectionArtworkDoesNotFollowPlayback(t *testing.T) {
	backend := &artworkBackend{}
	m := New(context.Background(), backend, Options{Artwork: true}).(*model)
	m.connected = true
	m.status = music.Status{State: "play", SongID: 41}
	m.queue = []music.Entry{{ID: 41, Track: fixtureTracks[2]}}
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.Update(keyRune('3'))
	_, first := m.Update(prepareLibrary(fixtureTracks, nil))
	if first == nil {
		t.Fatal("missing artwork request when the list first loads")
	}
	m.Update(keyRune('j'))
	old := first().(artworkMsg)
	if old.uri != fixtureTracks[2].URI {
		t.Fatalf("first artwork request: %q", old.uri)
	}
	old.lines = []string{"STALE COVER"}
	_, latest := m.Update(old)
	if latest == nil || latest().(artworkMsg).uri != fixtureTracks[1].URI {
		t.Fatal("artwork did not follow the selection")
	}
	view := ansi.Strip(m.View().Content)
	playbackStart := strings.Index(view, "╭ Playback")
	if playbackStart < 0 {
		t.Fatal("no playback frame")
	}
	panel, playback := view[:playbackStart], view[playbackStart:]
	if !strings.Contains(panel, "│ Violet") || strings.Contains(panel, "STALE COVER") {
		t.Fatalf("selection panel did not follow the browsing cursor:\n%s", panel)
	}
	if !strings.Contains(playback, "Playing · Other — Plum") || strings.Contains(playback, "Violet") {
		t.Fatalf("playback did not retain its current track:\n%s", playback)
	}
	if len(backend.actions) != 0 {
		t.Fatal("browsing sent a playback action")
	}
}
