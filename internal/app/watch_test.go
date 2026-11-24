package app

import (
	"context"
	"testing"
	"time"

	"rampd/internal/music"
)

func TestIdleChangesRefreshAffectedData(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.connected, m.busy, m.libraryLoaded = true, false, true
	m.handleWatch(watchMsg{changes: music.DatabaseChanged})
	if !m.libraryLoading || m.busy || m.playlistsLoading {
		t.Fatal("database change should only refresh the library")
	}
	m.handleWatch(watchMsg{changes: music.DatabaseChanged})
	if !m.libraryInvalidated {
		t.Fatal("change during a library read should schedule another read")
	}
	m.tab = playlistsTab
	m.handleWatch(watchMsg{changes: music.StoredPlaylistsChanged})
	if !m.playlistsLoading || m.busy {
		t.Fatal("stored playlist change should not refresh playback")
	}
	m.handleWatch(watchMsg{changes: music.PlaybackChanged})
	if !m.busy {
		t.Fatal("playback change should refresh status")
	}
	m.handleWatch(watchMsg{changes: music.PlaybackChanged})
	if !m.needSnapshot {
		t.Fatal("a change during a status read should schedule another read")
	}
}

func TestIdleStopsPeriodicStatusReads(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.connected, m.watching, m.busy, m.libraryLoaded = true, true, false, true
	m.updateState(stateMsg{})
	if m.update(refreshTick(m.refreshSeq)) != nil {
		t.Fatal("poll should be ignored while idle is active")
	}
	m.handleWatch(watchMsg{stopped: true})
	if m.watching || !m.busy || m.watchBackoff != 2*time.Second {
		t.Fatal("disconnect should resume status reads and schedule a retry")
	}
	m.handleWatch(watchMsg{changes: music.SessionReady})
	if !m.watching || !m.forceQueue || !m.needSnapshot || m.watchBackoff != time.Second {
		t.Fatal("new watch connection should force a full status and queue refresh")
	}
}

func TestPlaybackElapsedUsesLocalClock(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.connected = true
	m.status = music.Status{State: "play", Elapsed: 3 * time.Second, Duration: 8 * time.Second}
	m.sampledAt = time.Unix(100, 0)
	m.update(progressTick(m.sampledAt.Add(2 * time.Second)))
	if got := m.playbackElapsed(); got != 5*time.Second {
		t.Fatalf("elapsed playback: %s", got)
	}
	m.update(progressTick(m.sampledAt.Add(20 * time.Second)))
	if got := m.playbackElapsed(); got != m.status.Duration {
		t.Fatalf("elapsed playback should not overrun the track: %s", got)
	}
	m.status.State = "pause"
	if got := m.playbackElapsed(); got != m.status.Elapsed {
		t.Fatalf("paused playback advanced: %s", got)
	}
}
