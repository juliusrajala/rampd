package app

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/music"
)

func TestStoredPlaylistEditUsesPositionInFullList(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy = true, false
	m.tab = playlistsTab
	m.openPlaylist("Test")
	tracks := []music.Track{fixtureTracks[0], fixtureTracks[1], fixtureTracks[0]}
	m.updatePlaylist(playlistMsg{name: "Test", seq: m.playlistSeq, tracks: tracks})
	m.page().setFilter("Second")
	m.page().move(1)
	if cmd := m.key(keyRune('x')); cmd != nil || m.confirmation == nil {
		t.Fatal("removing a stored track must ask for confirmation")
	}
	if cmd := m.key(tea.KeyPressMsg{Code: tea.KeyEsc}); cmd != nil || len(server.actions) != 0 {
		t.Fatal("cancel should not edit the playlist")
	}
	m.key(keyRune('x'))
	cmd := m.key(keyRune('y'))
	if cmd == nil {
		t.Fatal("confirmation did not dispatch the edit")
	}
	cmd()
	if len(server.actions) != 1 {
		t.Fatalf("expected one edit, got %v", server.actions)
	}
	a := server.actions[0]
	if a.Operation != music.DeletePlaylistTrack || a.Name != "Test" || a.Position != 2 || len(a.Expected) != 3 {
		t.Fatalf("wrong playlist target: %+v", a)
	}
}

func TestAddAlbumToNamedPlaylist(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy = true, false
	m.tab = albumsTab
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.page().setFilter("Violet")
	m.key(keyRune('A'))
	if m.prompt != playlistNamePrompt {
		t.Fatal("no playlist prompt")
	}
	m.Update(tea.PasteMsg{Content: "Road Mix"})
	cmd := m.key(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("no add command")
	}
	cmd()
	if len(server.actions) != 1 || server.actions[0].Operation != music.AddToPlaylist ||
		server.actions[0].Name != "Road Mix" || len(server.actions[0].Tracks) != 2 {
		t.Fatalf("wrong playlist add: %+v", server.actions)
	}
}
