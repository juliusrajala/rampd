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
	if m.confirmation != nil {
		t.Fatal("replace should be inactive in Edit tracks mode")
	}
	m.key(trayModeKey())
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
	if len(m.tray) != 2 {
		t.Fatal("remove should be inactive in Whole Tray mode")
	}
	m.key(trayModeKey())
	m.key(keyRune('x'))
	m.key(trayModeKey())
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
	m.key(trayModeKey())
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

func TestShuffleTrayFromPaletteSavesAllEntries(t *testing.T) {
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
	if saved != nil {
		t.Fatal("Tray shuffle must not use the queue shuffle key by default")
	}
	m.key(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "shuffle tray"})
	if len(m.page().rows) != 0 {
		t.Fatal("Edit tracks palette offered a whole-Tray action")
	}
	m.key(tea.KeyPressMsg{Code: tea.KeyEsc})
	m.key(trayModeKey())
	m.key(keyRune(':'))
	m.Update(tea.PasteMsg{Content: "shuffle tray"})
	m.key(tea.KeyPressMsg{Code: tea.KeyEnter})
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
}

func trayModeKey() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl}
}

func TestTrayFooterFlipsActionScopes(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{InitialTray: fixtureTracks}).(*model)
	m.tab = trayTab
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m.key(keyRune('v'))
	m.page().setFilter("First")

	m.notice = "Copied queue to Tray: 3 tracks."
	view := ansi.Strip(m.View().Content)
	lines := strings.Split(view, "\n")
	notice := strings.Index(view, "\n"+strings.Repeat(" ", m.sidebarWidth())+"Copied queue to Tray: 3 tracks.")
	playback := strings.Index(view, "╭ Playback")
	if notice < 0 || playback < notice || strings.Count(view, "Copied queue") != 1 {
		t.Fatalf("notice should be inset above Playback: %q", view)
	}
	edit := lines[len(lines)-1]
	if !strings.HasPrefix(edit, "  [ctrl+t] Mode  1 marked · 1 hidden") || !strings.Contains(edit, "[x] Remove") ||
		strings.Contains(view, "Whole Tray") {
		t.Fatalf("wrong edit hints: %q", edit)
	}
	m.Update(trayModeKey())
	view = ansi.Strip(m.View().Content)
	lines = strings.Split(view, "\n")
	whole := lines[len(lines)-1]
	if !strings.HasPrefix(whole, "  [ctrl+t] Mode  Whole Tray") || !strings.Contains(whole, "[P] Replace") ||
		strings.Contains(view, "[x] Remove") || !strings.HasPrefix(lines[len(lines)-2], "╰") {
		t.Fatalf("expected one row of whole-Tray hints: %q", whole)
	}
	if len(m.trayMarks) != 1 || m.page().filter != "First" || !slices.Equal(m.trayTracks(), fixtureTracks) {
		t.Fatal("flipping hints changed the Tray, marks, or filter")
	}

	m.Update(tea.WindowSizeMsg{Width: 50, Height: 18})
	lines = strings.Split(ansi.Strip(m.View().Content), "\n")
	if !strings.Contains(lines[len(lines)-1], "[:] Commands") || !strings.Contains(lines[len(lines)-1], "[ctrl+t] Mode") {
		t.Fatal("compact footer should keep the flip and palette hints")
	}
	m.Update(trayModeKey())
	if m.trayWholeMode {
		t.Fatal("flipping should return to edit hints")
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
