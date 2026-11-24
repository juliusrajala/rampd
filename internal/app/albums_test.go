package app

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/music"
)

func TestAlbumsAlignArtistsOnRight(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('2'))
	for i, want := range []string{"Plum", "Violet"} {
		line := ansi.Strip(m.viewRows()[i])
		if !strings.HasPrefix(strings.TrimLeft(line, " ›"), "Bright") || !strings.HasSuffix(strings.TrimSpace(strings.TrimSuffix(line, "—")), want) || ansi.StringWidth(line) != m.innerWidth(m.bodyWidth()) {
			t.Errorf("expected album on left and artist on right: %q", line)
		}
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if line := ansi.Strip(m.viewRows()[0]); !strings.Contains(line, "Other  ·  Plum · Bright") {
		t.Errorf("track rows should keep their original layout: %q", line)
	}
}

func TestAlbumSortKeepsSelectionAndFilterOnRefresh(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	tracks := []music.Track{
		{URI: "1", Album: "Able", Artist: "Zulu Band"},
		{URI: "2", Album: "Zebra", Artist: "Alpha Band"},
		{URI: "3", Album: "Baker", Artist: "Alpha Band"},
		{URI: "4", Album: "Hidden", Artist: "Solo"},
	}
	msg := prepareLibrary(tracks, nil)
	m.Update(msg)
	m.Update(keyRune('2'))
	m.page().setFilter("band")
	m.Update(keyRune('j'))

	assertOrder := func(want string) {
		t.Helper()
		p := m.page()
		if len(p.rows) != 3 || p.rows[0].label != want || p.filter != "band" {
			t.Fatalf("wrong sort order or filter: %+v", p)
		}
		if got, ok := m.selected(); !ok || got.label != "Baker" {
			t.Fatalf("sorting changed the selection: %+v", got)
		}
	}
	assertOrder("Able")
	m.Update(keyRune('o'))
	assertOrder("Baker")
	m.Update(msg)
	assertOrder("Baker")

	m.Update(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "sort by title"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	assertOrder("Able")
}

func TestSongsSortAndColumnsMatchAlbums(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	msg := prepareLibrary([]music.Track{
		{URI: "1", Title: "A Song", Artist: "Zulu", Album: "Mix One"},
		{URI: "2", Title: "Z Song", Artist: "Alpha", Album: "Mix Two"},
		{URI: "3", Title: "B Song", Artist: "Alpha", Album: "Mix Three"},
	}, nil)
	m.Update(msg)
	m.Update(keyRune('1'))
	m.page().setFilter("mix") // Albums still match, even though the rows hide them.
	line := ansi.Strip(m.viewRows()[0])
	if !strings.Contains(line, "A Song") || !strings.Contains(line, "  Zulu") || strings.Contains(line, "Mix One") || strings.Contains(line, "·") {
		t.Fatalf("Songs should use separate columns like Albums: %q", line)
	}
	m.Update(keyRune('o'))
	checkSongs := func() {
		t.Helper()
		p := m.page()
		if len(p.rows) != 3 || p.rows[0].label != "B Song" || p.rows[1].label != "Z Song" || p.filter != "mix" {
			t.Fatalf("wrong artist order or filter: %+v", p)
		}
		if r, ok := m.selected(); !ok || r.label != "A Song" {
			t.Fatalf("sort changed the selected song: %+v", r)
		}
	}
	checkSongs()
	m.Update(msg)
	checkSongs()
	m.Update(keyRune('2'))
	if m.mediaOrderLabel() != "title" {
		t.Fatal("song sort changed album sort")
	}
	m.Update(keyRune('1'))
	m.Update(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "sort by title"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.page().rows[0].label != "A Song" || m.mediaOrderLabel() != "title" {
		t.Fatal("palette did not restore title order")
	}
}

func TestAlbumColumnsFitWideCharacters(t *testing.T) {
	line := alignMediaRow("›   とても長いアルバム名", "長い歌手の名前", 16)
	if ansi.StringWidth(line) != 16 || !strings.HasPrefix(line, "›   と") || !strings.HasSuffix(line, "…") {
		t.Fatalf("columns should truncate to fit: %q", line)
	}
}
