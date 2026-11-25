package app

import (
	"context"
	"testing"
	"time"

	"rampd/internal/music"
)

func TestObservedPlayCountsOncePerEntry(t *testing.T) {
	var p observedPlay
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, s := range []struct {
		second, id int
		playing    bool
		want       bool
	}{
		{0, 1, true, false}, {1, 1, true, false},
		{2, 1, false, false}, {90, 1, false, false},
		{91, 1, true, false}, {92, 1, true, true}, {93, 1, true, false},
		{94, 2, true, false}, {95, 2, true, false}, {96, 2, true, true},
		{97, -1, false, false}, {98, 2, true, false}, {99, 2, true, false}, {100, 2, true, true},
	} {
		if got := p.sample(s.id, "song.flac", s.playing, 4*time.Second, start.Add(time.Duration(s.second)*time.Second)); got != s.want {
			t.Errorf("observation at %ds: got %t, want %t", s.second, got, s.want)
		}
	}
}

type historyBackend struct {
	fakeBackend
	uri string
	at  time.Time
}

func (f *historyBackend) RecordPlay(_ context.Context, uri string, at time.Time) error {
	f.uri, f.at = uri, at
	return nil
}

func TestObservePlayIgnoresSeekPositionAndLongGaps(t *testing.T) {
	server := &historyBackend{}
	m := New(context.Background(), server, Options{}).(*model)
	m.connected = true
	m.status = music.Status{State: "play", SongID: 4, Duration: 2 * time.Minute}
	m.queue = []music.Entry{{ID: 4, Track: fixtureTracks[0]}}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if cmd := m.observePlay(start); cmd != nil {
		t.Fatal("first observation counted a play")
	}
	m.status.Elapsed = 80 * time.Second
	if cmd := m.observePlay(start.Add(time.Second)); cmd != nil {
		t.Fatal("seek position counted towards a play")
	}
	start = start.Add(time.Minute)
	for i := range 29 {
		if cmd := m.observePlay(start.Add(time.Duration(i) * time.Second)); cmd != nil {
			t.Fatalf("play qualified early at sample %d", i)
		}
	}
	qualified := start.Add(29 * time.Second)
	cmd := m.observePlay(qualified)
	if cmd == nil {
		t.Fatal("expected a sticker command after 30 seconds of observed play")
	}
	cmd()
	if server.uri != fixtureTracks[0].URI || !server.at.Equal(qualified) || len(server.actions) != 0 {
		t.Fatalf("wrong history write: %q at %s", server.uri, server.at)
	}
}
