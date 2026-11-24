package app

import (
	"context"
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/dream"
	"rampd/internal/music"
)

func TestNavigationWhileLibraryArtworkAndDreamArePending(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	m.connected = true
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.Update(prepareLibrary(fixtureTracks, nil))
	if m.artworkLoading == "" {
		t.Fatal("expected an outstanding artwork request")
	}
	if m.fetchLibrary() == nil || m.generateDream() == nil {
		t.Fatal("expected outstanding library and Dream commands")
	}

	// Do not deliver the command results. The user must still be able to browse
	// the previous library while the commands run.
	m.Update(keyRune('j'))
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if page := m.page(); page.title != "Artists / Violet" || len(page.rows) != 1 {
		t.Fatalf("pending work blocked navigation: %+v", page)
	}
	m.Update(keyRune('/'))
	m.Update(tea.PasteMsg{Content: "bright"})
	if got, ok := m.selected(); !ok || got.label != "Bright" {
		t.Fatal("pending work blocked filtering")
	}
}

func largeLibrary() []music.Track {
	const count = 50_000
	tracks := make([]music.Track, count)
	for i := range tracks {
		artist, album, track := i/100, (i/10)%10, i%10
		tracks[i] = music.Track{
			URI:    fmt.Sprintf("%d/%d/%d.flac", artist, album, track),
			Artist: fmt.Sprintf("Artist %04d", artist), Album: fmt.Sprintf("Album %02d", album),
			Title: fmt.Sprintf("Track %02d", track), Number: track + 1,
		}
	}
	return tracks
}

func BenchmarkLibraryOpen50k(b *testing.B) {
	tracks := largeLibrary()
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	b.ReportAllocs()
	for b.Loop() {
		m.Update(prepareLibrary(tracks, nil))
	}
}

func BenchmarkApplyPreparedLibrary50k(b *testing.B) {
	msg := prepareLibrary(largeLibrary(), nil)
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	b.ReportAllocs()
	for b.Loop() {
		m.Update(msg)
	}
}

func BenchmarkFilterQueue50k(b *testing.B) {
	tracks := largeLibrary()
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.tab = queueTab
	for i, track := range tracks {
		m.queue = append(m.queue, music.Entry{ID: i, Track: track})
	}
	m.buildQueuePage()
	b.ReportAllocs()
	for b.Loop() {
		m.page().setFilter("artist 004")
		m.page().setFilter("artist 0042")
	}
}

func BenchmarkNavigateQueue50k(b *testing.B) {
	tracks := largeLibrary()
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.tab = queueTab
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	for i, track := range tracks {
		m.queue = append(m.queue, music.Entry{ID: i, Track: track})
	}
	m.buildQueuePage()
	b.ReportAllocs()
	for b.Loop() {
		m.Update(keyRune('j'))
		m.View()
	}
}

func BenchmarkDream50k(b *testing.B) {
	tracks := largeLibrary()
	strategy := dream.Baseline{}
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		if _, err := strategy.Generate(ctx, tracks, dream.Options{TrackCount: 25}); err != nil {
			b.Fatal(err)
		}
	}
}
