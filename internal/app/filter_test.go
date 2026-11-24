package app

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/music"
)

func TestEscapeClearsFocusedFilter(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('/'))
	m.Update(tea.PasteMsg{Content: "missing"})
	if len(m.page().rows) != 0 || !m.showFilter() {
		t.Fatal("pasting did not filter the list")
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.showFilter() || m.prompt != noPrompt || len(m.page().rows) != 2 {
		t.Fatal("Escape did not restore the list and hide the focused filter")
	}
}

func TestFilteredQueueActionsUseSongID(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy = true, false
	m.tab = queueTab
	m.queue = []music.Entry{
		{ID: 42, Track: fixtureTracks[0]},
		{ID: 81, Track: fixtureTracks[1]},
	}
	m.buildQueuePage()
	m.page().setFilter("first")
	cmd := keyCommand(m, keyRune('x'))
	if cmd == nil {
		t.Fatal("no remove command for filtered entry")
	}
	cmd()
	if len(server.actions) != 1 || server.actions[0].Operation != music.Remove || server.actions[0].ID != 81 {
		t.Fatalf("remove did not use the filtered entry's ID: %+v", server.actions)
	}
	m.queue = append([]music.Entry{{ID: 99, Track: fixtureTracks[2]}}, m.queue...)
	m.buildQueuePage()
	if item, ok := m.selected(); !ok || item.id != 81 || len(m.page().rows) != 1 || len(m.queue) != 3 {
		t.Fatal("refresh did not retain the filter and selected song ID")
	}
}

func TestFilteredBrowseSurvivesLibraryRefresh(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.page().setFilter("violet")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.page().setFilter("second")
	m.Update(prepareLibrary(fixtureTracks, nil))

	if len(m.tabs[artistsTab]) != 3 {
		t.Fatal("refresh closed an open page under a filtered parent")
	}
	row, ok := m.selected()
	if !ok || row.label != "Second" || len(m.page().rows) != 1 || m.page().filter != "second" {
		t.Fatalf("refresh lost the filter: %+v", m.page())
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if len(m.tabs[artistsTab]) != 3 || m.page().filter != "" {
		t.Fatal("Escape should clear the filter before navigating back")
	}
}

func TestDreamAndPlaylistFiltersDoNotChangeTheirContents(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(dreamMsg{tracks: fixtureTracks})
	m.Update(keyRune('7'))
	m.page().setFilter("violet first")
	if len(m.page().rows) != 1 || len(m.dreamTracks) != 3 {
		t.Fatal("Dream filter changed the session")
	}
	m.Update(keyRune('6'))
	m.Update(playlistsMsg{names: []string{"Day", "Night"}})
	m.page().setFilter("night")
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Update(playlistMsg{name: "Night", seq: m.playlistSeq, tracks: fixtureTracks})
	m.page().setFilter("second")
	if len(m.page().rows) != 1 || len(m.page().tracks) != 3 {
		t.Fatal("playlist filter changed the stored track list")
	}
}
