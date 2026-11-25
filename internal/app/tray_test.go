package app

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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
	m.Update(keyRune('1'))
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

func TestTrayMarksMoveAndRemoveTogetherAcrossFilter(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.tab = trayTab
	for i, track := range []music.Track{fixtureTracks[0], fixtureTracks[1], fixtureTracks[0], fixtureTracks[2]} {
		m.addTrayEntry(music.Entry{ID: i, Track: track})
	}
	m.updateTrayRows()
	m.key(keyRune('v'))
	m.page().move(1)
	m.key(keyRune('v'))
	m.page().setFilter("First")
	m.key(keyRune('J'))
	for i, want := range []int{2, 0, 1, 3} {
		if m.tray[i].ID != want {
			t.Fatalf("move changed the order within the marked group: %+v", m.tray)
		}
	}
	m.key(keyRune('x'))
	if len(m.tray) != 2 || m.tray[0].ID != 2 || m.tray[1].ID != 3 || len(m.trayMarks) != 0 || len(server.actions) != 0 {
		t.Fatalf("remove should affect both marked entries, but not their duplicates or MPD: %+v", m.tray)
	}
	m.page().setFilter("")
	m.key(keyRune('v'))
	m.key(tea.KeyPressMsg{Code: tea.KeyEsc})
	if len(m.trayMarks) != 0 {
		t.Fatal("Escape did not clear the marks")
	}
}

func TestRestoredTrayAutosavesTrackEditsOnly(t *testing.T) {
	initial := []music.Track{fixtureTracks[0], fixtureTracks[1], fixtureTracks[0]}
	var saves [][]music.Track
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{
		InitialTray: initial,
		SaveTray: func(tracks []music.Track) error {
			saves = append(saves, tracks)
			return errors.New("disk full")
		},
	}).(*model)
	m.tab = trayTab
	if len(saves) != 0 || !slices.Equal(m.trayTracks(), initial) || m.tray[0].ID != -1 || len(m.trayMarks) != 0 {
		t.Fatal("restoring the Tray should only load tracks")
	}
	m.key(keyRune('v'))
	if len(saves) != 0 {
		t.Fatal("marks must not be saved")
	}
	m.key(keyRune('J'))
	m.key(keyRune('x'))
	want := []music.Track{fixtureTracks[1], fixtureTracks[0]}
	if len(saves) != 2 || !slices.Equal(saves[1], want) || !slices.Equal(m.trayTracks(), want) || len(server.actions) != 0 {
		t.Fatalf("edits did not save the right tracks or changed MPD: %v", saves)
	}
	if !strings.Contains(m.notice, "Tray not saved: disk full") {
		t.Fatalf("save failure was hidden: %q", m.notice)
	}
}

func TestRemoveTrayDuplicatesFromPalette(t *testing.T) {
	server := &fakeBackend{}
	first, other := fixtureTracks[0], fixtureTracks[1]
	var saved [][]music.Track
	m := New(context.Background(), server, Options{
		SaveTray: func(tracks []music.Track) error {
			saved = append(saved, tracks)
			return nil
		},
	}).(*model)
	for i, track := range []music.Track{first, other, first, fixtureTracks[2], other} {
		m.addTrayEntry(music.Entry{ID: i + 10, Track: track})
	}
	m.updateTrayRows()
	m.tab = trayTab
	keep, drop := m.tray[0].key, m.tray[2].key
	m.trayMarks = map[int]bool{keep: true, drop: true}
	m.page().setFilter("First")
	m.Update(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "remove duplicates"})
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	want := []music.Track{first, other, fixtureTracks[2]}
	if len(saved) != 1 || !slices.Equal(saved[0], want) || !slices.Equal(m.trayTracks(), want) ||
		m.tray[0].ID != 10 || m.tray[1].ID != 11 || m.tray[2].ID != 13 {
		t.Fatalf("expected to save only first occurrences in order: %v", saved)
	}
	if m.page().filter != "First" || len(m.trayMarks) != 1 || !m.trayMarks[keep] || len(server.actions) != 0 {
		t.Fatal("removing duplicates changed the filter, surviving marks, or live queue")
	}
	m.Update(keyRune('d'))
	if len(saved) != 1 || m.notice != "No duplicates in the Tray." {
		t.Fatal("the shortcut should report no duplicates without saving again")
	}
}

func TestShuffleKeyUsesCurrentList(t *testing.T) {
	server := &fakeBackend{}
	initial := []music.Track{fixtureTracks[0], fixtureTracks[1], fixtureTracks[0], fixtureTracks[2]}
	var saved []music.Track
	m := New(context.Background(), server, Options{
		InitialTray: initial,
		SaveTray: func(tracks []music.Track) error {
			saved = tracks
			return nil
		},
	}).(*model)
	m.tab = trayTab
	marked := m.tray[0].key
	m.key(keyRune('v'))
	m.page().setFilter("First")
	m.key(keyRune('s'))
	if m.notice != "Shuffled Tray." || !slices.Equal(saved, m.trayTracks()) || len(saved) != len(initial) {
		t.Fatalf("shuffle did not save the complete Tray: %v", saved)
	}
	counts := make(map[string]int)
	for _, track := range saved {
		counts[track.URI]++
	}
	if counts[initial[0].URI] != 2 || counts[initial[1].URI] != 1 || counts[initial[3].URI] != 1 ||
		!m.trayMarks[marked] || m.page().filter != "First" || len(server.actions) != 0 {
		t.Fatal("shuffle must preserve repeats, marks, filter, and live queue")
	}
	before := m.trayTracks()
	m.tab, m.connected, m.busy = queueTab, true, false
	m.queue = []music.Entry{{ID: 1, Track: initial[0]}, {ID: 2, Track: initial[1]}}
	cmd := m.key(keyRune('s'))
	if cmd == nil {
		t.Fatal("Queue did not handle the shared shuffle key")
	}
	cmd()
	if !slices.Equal(before, m.trayTracks()) || len(server.actions) != 1 ||
		server.actions[0].Operation != music.Shuffle {
		t.Fatal("Queue shuffle must not change the Tray")
	}
}

func TestTrayMenuShowsBothActionGroups(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{InitialTray: fixtureTracks}).(*model)
	m.tab = trayTab
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 36})
	m.key(keyRune('v'))
	m.page().setFilter("First")

	m.notice = "Copied queue to Tray: 3 tracks."
	view := ansi.Strip(m.View().Content)
	lines := strings.Split(view, "\n")
	notice := strings.Index(view, "Copied queue to Tray: 3 tracks.")
	playback := strings.Index(view, "╭ Playback")
	if notice < 0 || playback < notice || strings.Count(view, "Copied queue") != 1 {
		t.Fatalf("notice should be inset above Playback: %q", view)
	}
	if !strings.Contains(view, "Tray · 1 marked · 1 hidden") || !strings.Contains(view, "[x]") ||
		!strings.Contains(view, "Shuffle Tray") || !strings.Contains(view, "Copy queue") ||
		strings.Contains(view, "[ctrl+t]") ||
		!strings.HasPrefix(ansi.Cut(lines[len(lines)-1], m.sidebarWidth(), m.width), "╰") {
		t.Fatalf("expected both command groups and no mode switch or footer: %q", view)
	}

	m.Update(tea.WindowSizeMsg{Width: 50, Height: 18})
	lines = strings.Split(ansi.Strip(m.View().Content), "\n")
	if !strings.Contains(lines[0], "[:]") || strings.Contains(lines[0], "[ctrl+t]") {
		t.Fatal("compact navigation should advertise the palette, not Tray actions")
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
