package app

import (
	"context"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/dream"
	"rampd/internal/music"
)

type firstTrack struct{}

func (firstTrack) Name() string { return "first-track" }

func (firstTrack) Generate(_ context.Context, tracks []music.Track, opts dream.Options) ([]music.Track, error) {
	return slices.Clone(tracks[:opts.TrackCount]), nil
}

func TestDreamPreviewDoesNotChangeQueue(t *testing.T) {
	server := &fakeBackend{}
	m := New(context.Background(), server, Options{DreamStrategy: firstTrack{}, DreamCount: 1}).(*model)
	m.connected, m.busy = true, false
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.Update(keyRune('7'))
	cmd := keyCommand(m, keyRune('g'))
	if cmd == nil {
		t.Fatal("Dream generation did not start")
	}
	m.Update(cmd())
	if len(server.actions) != 0 || len(m.dreamTracks) != 1 || m.dreamTracks[0] != fixtureTracks[0] {
		t.Fatalf("Dream did not use the supplied strategy or changed playback: %+v", m.dreamTracks)
	}
	m.Update(keyRune('P'))
	m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if len(server.actions) != 0 || m.busy {
		t.Fatal("cancelling replace changed the queue")
	}
	cmd = keyCommand(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("playing the Dream did not start")
	}
	cmd()
	if len(server.actions) != 1 || server.actions[0].Operation != music.AppendAndPlay {
		t.Fatalf("playing the Dream: %+v", server.actions)
	}
}
