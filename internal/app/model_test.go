package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/music"
)

type fakeBackend struct {
	actions []music.Action
}

func (*fakeBackend) Snapshot(context.Context, string) (music.Snapshot, error) {
	return music.Snapshot{}, nil
}

func (*fakeBackend) Library(context.Context) ([]music.Track, error) {
	return nil, nil
}

func (*fakeBackend) Playlists(context.Context) ([]string, error) { return nil, nil }
func (*fakeBackend) Playlist(context.Context, string) ([]music.Track, error) {
	return nil, nil
}

func (*fakeBackend) Artwork(context.Context, string) ([]byte, error) { return nil, nil }

func (*fakeBackend) Watch(ctx context.Context, _ func(music.Changes)) error {
	<-ctx.Done()
	return ctx.Err()
}

func (f *fakeBackend) Apply(_ context.Context, a music.Action) error {
	f.actions = append(f.actions, a)
	return nil
}

var fixtureTracks = []music.Track{
	{URI: "2.flac", Title: "Second", Artist: "Violet", Album: "Bright", Genre: "Jazz", Number: 2},
	{URI: "1.flac", Title: "First", Artist: "Violet", Album: "Bright", Genre: "Jazz", Number: 1},
	{URI: "else.flac", Title: "Other", Artist: "Plum", Album: "Bright", Number: 1},
}

func TestBrowseAndAppendAlbum(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy = true, false
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('j'))
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if !strings.HasSuffix(m.page().title, "/ Violet") {
		t.Fatalf("did not open artist: %s", m.page().title)
	}
	cmd := keyCommand(m, tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	if cmd == nil {
		t.Fatal("album append did not start")
	}
	cmd()
	got := server.actions
	if len(got) != 1 || got[0].Operation != music.Append || len(got[0].Tracks) != 2 || got[0].Tracks[0].Title != "First" {
		t.Fatalf("unexpected album action: %+v", got)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	gotRow, _ := m.selected()
	if gotRow.label != "First" {
		t.Fatalf("first track: %+v", gotRow)
	}
	m.Update(keyRune('h'))
	m.Update(keyRune('h'))
	gotRow, _ = m.selected()
	if gotRow.label != "Violet" {
		t.Fatalf("artist selection was not preserved: %+v", gotRow)
	}
}

func TestSearchInputDoesNotDispatchPlayback(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy = true, false
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('2'))
	m.Update(keyRune('/'))
	for _, ch := range "plum bright" {
		m.Update(keyRune(ch))
	}
	p := m.page()
	if len(p.rows) != 1 || p.rows[0].detail != "Plum" {
		t.Fatalf("typing did not filter the current list: %+v", p.rows)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(server.actions) > 0 || m.busy || len(m.pending) != 0 || m.tab != albumsTab || m.prompt != noPrompt || p.filter != "plum bright" {
		t.Fatalf("search dispatched an action or did not keep its result: %+v", p)
	}
	m.Update(keyRune('1'))
	if m.page().filter != "" || len(m.page().rows) != 2 {
		t.Fatal("filter leaked to another tab")
	}
	m.Update(keyRune('2'))
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if p.filter != "" || len(p.rows) != 2 {
		t.Fatalf("Escape did not clear the filter: %+v", p)
	}
}

func TestQueueSelectionAndClearConfirmation(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected, m.busy, m.tab = true, false, queueTab
	a := music.Entry{ID: 0, Track: fixtureTracks[0]}
	b := music.Entry{ID: 1, Track: fixtureTracks[1]}
	m.queue = []music.Entry{a, b}
	m.buildQueuePage()
	m.Update(keyRune('j'))
	m.queue = []music.Entry{b, a}
	m.buildQueuePage()
	selected, ok := m.selected()
	if !ok || selected.id != b.ID {
		t.Fatalf("selected entry changed across reorder: %+v", selected)
	}
	m.Update(keyRune('X'))
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if len(server.actions) > 0 || m.busy {
		t.Fatal("cancelling clear changed the queue")
	}
	m.Update(keyRune('X'))
	cmd := keyCommand(m, keyRune('y'))
	if cmd == nil {
		t.Fatal("confirming clear did not start a command")
	}
	cmd()
	if len(server.actions) != 1 || server.actions[0].Operation != music.Clear {
		t.Fatalf("clear: %+v", server.actions)
	}
}

func TestRetryResetsOnReconnect(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.connected, m.libraryLoaded = true, true
	m.pending = []music.Action{{Operation: music.Next}}
	m.updateState(stateMsg{readError: errors.New("offline")})
	if m.connected || len(m.pending) > 0 || m.retry != 2*time.Second {
		t.Fatalf("disconnect did not reset transient state: connected=%v retry=%v", m.connected, m.retry)
	}
	m.updateState(stateMsg{})
	if !m.connected || m.notice != "" || m.retry != time.Second {
		t.Fatalf("did not recover: connected=%v retry=%v notice=%q", m.connected, m.retry, m.notice)
	}
}

func TestViewFitsTerminal(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(prepareLibrary(fixtureTracks, nil))
	for _, size := range []tea.WindowSizeMsg{{Width: 1, Height: 1}, {Width: 12, Height: 5}, {Width: 80, Height: 25}} {
		m.Update(size)
		for _, mode := range []string{"browse", "search", "help", "confirm"} {
			m.prompt, m.showHelp = noPrompt, mode == "help"
			if mode == "search" {
				m.prompt = searchPrompt
			}
			m.confirmation = nil
			if mode == "confirm" {
				m.confirmation = &confirmation{message: "Confirm the action?"}
			}
			m.input.Focus()
			view := m.View()
			lines := strings.Split(view.Content, "\n")
			if len(lines) > size.Height {
				t.Fatalf("%s: %d lines at height %d", mode, len(lines), size.Height)
			}
			for _, line := range lines {
				if w := ansi.StringWidth(line); w > size.Width {
					t.Fatalf("%s: %d cells at width %d: %q", mode, w, size.Width, line)
				}
			}
			if view.Cursor != nil && (view.Cursor.Y >= size.Height || view.Cursor.X >= size.Width) {
				t.Fatalf("%s: cursor outside terminal: %+v", mode, view.Cursor)
			}
		}
	}
}

func keyRune(ch rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: ch, Text: string(ch)}
}

func keyCommand(m *model, key tea.KeyPressMsg) tea.Cmd {
	_, cmd := m.Update(key)
	return cmd
}
