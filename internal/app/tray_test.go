package app

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/music"
)

type trayBackend struct {
	fakeBackend
	snapshot music.Snapshot
}

func (s *trayBackend) Snapshot(context.Context, string) (music.Snapshot, error) {
	return s.snapshot, nil
}

func TestSongsFilterAndCollect(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('3'))
	m.Update(keyRune('/'))
	m.Update(tea.PasteMsg{Content: "violet first"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m.Update(keyRune('t'))
	m.Update(keyRune('t'))
	if m.tab != songsTab || len(m.tray) != 1 || m.tray[0].Title != "First" || m.page().filter != "violet first" {
		t.Fatalf("collect should keep browsing state and skip duplicates: %+v", m.tray)
	}
	if len(server.actions) != 0 {
		t.Fatal("collecting touched MPD")
	}
}

func TestQueueCopyUsesFreshSnapshotAndKeepsRepeatedTracks(t *testing.T) {
	repeated := fixtureTracks[0]
	server := &trayBackend{snapshot: music.Snapshot{Queue: []music.Entry{
		{ID: 10, Track: repeated}, {ID: 11, Track: fixtureTracks[1]}, {ID: 12, Track: repeated},
	}}}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.tab = true, queueTab
	m.addTrayEntry(music.Entry{ID: -1, Track: fixtureTracks[2]})
	m.updateTrayRows()
	m.key(keyRune('C'))
	if m.confirmation == nil {
		t.Fatal("replacing a populated Tray did not ask for confirmation")
	}
	m.key(tea.KeyPressMsg{Code: tea.KeyEsc})
	if len(m.tray) != 1 || m.tray[0].URI != fixtureTracks[2].URI {
		t.Fatal("cancelling changed the Tray")
	}
	m.key(keyRune('C'))
	cmd := m.key(keyRune('y'))
	if cmd == nil {
		t.Fatal("confirming did not start the copy")
	}
	m.Update(cmd())
	if m.tab != trayTab || len(m.tray) != 3 || m.tray[0].ID != 10 || m.tray[2].ID != 12 {
		t.Fatalf("copy did not preserve order and repeats: %+v", m.tray)
	}

	m.page().setFilter("second")
	m.page().move(1)
	m.key(keyRune('K'))
	if entry, ok := m.selected(); !ok || entry.position != 1 || m.tray[1].ID != 12 {
		t.Fatalf("move should address the selected occurrence in the full list: %+v", m.tray)
	}
	m.key(keyRune('x'))
	if len(m.tray) != 2 || m.tray[0].ID != 10 || m.tray[1].ID != 11 || len(server.actions) != 0 {
		t.Fatal("remove should change only the selected occurrence in the Tray")
	}
}

func TestTrayReplacementKeepsTheOriginalPlayingOccurrence(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy, m.tab = true, false, trayTab
	track := fixtureTracks[0]
	m.status = music.Status{State: "play", SongID: 11, QueueVersion: "9"}
	m.queue = []music.Entry{{ID: 11, Track: track}, {ID: 12, Track: track}}
	m.addTrayEntry(m.queue[1])
	m.addTrayEntry(m.queue[0])
	m.updateTrayRows()
	m.key(keyRune('P'))
	if m.confirmation == nil || strings.Contains(m.confirmation.message, "stop") {
		t.Fatal("retained playing entry should not stop")
	}
	cmd := m.key(keyRune('y'))
	if cmd == nil {
		t.Fatal("confirmed replacement did not dispatch")
	}
	cmd()
	if len(server.actions) != 1 || server.actions[0].Operation != music.ReplaceQueue || server.actions[0].Replacement.KeepAt != 1 || len(m.tray) != 2 {
		t.Fatalf("wrong replacement action or Tray changed: %+v", server.actions)
	}
	m.busy = false
	m.page().move(1)
	m.key(keyRune('x'))
	m.key(keyRune('P'))
	if m.confirmation == nil || !strings.Contains(m.confirmation.message, "stop") {
		t.Fatal("removing the playing occurrence should warn even when another repeat remains")
	}
}

func TestAppendUsesFullTrayEvenWhenFiltered(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy, m.tab = true, false, trayTab
	m.addTrayEntry(music.Entry{ID: -1, Track: fixtureTracks[0]})
	m.addTrayEntry(music.Entry{ID: -1, Track: fixtureTracks[1]})
	m.updateTrayRows()
	m.page().setFilter("first")
	cmd := m.key(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd == nil {
		t.Fatal("no append command")
	}
	cmd()
	if len(server.actions) != 1 || server.actions[0].Operation != music.Append ||
		len(server.actions[0].Tracks) != 2 || len(m.tray) != 2 {
		t.Fatalf("append ignored hidden tracks or changed the Tray: %+v", server.actions)
	}
}

func TestTrayCopyDoesNotOverwriteNewerEdits(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.connected = true
	command := m.copyQueueToTray()
	if command == nil {
		t.Fatal("no copy request")
	}
	m.addTrayEntry(music.Entry{ID: -1, Track: fixtureTracks[0]})
	m.updateTrayRows()
	m.Update(command())
	if len(m.tray) != 1 || m.tray[0].Track != fixtureTracks[0] {
		t.Fatal("a stale copy replaced the edited Tray")
	}
}
