package app

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/music"
)

func TestViewShowsStateWithoutColor(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.connected = true
	m.status = music.Status{State: "play", SongID: 1, Elapsed: 15 * time.Second, Duration: time.Minute}
	m.queue = []music.Entry{{ID: 1, Track: fixtureTracks[0]}}
	m.storeArtwork(artworkMsg{uri: fixtureTracks[1].URI})

	view := ansi.Strip(m.View().Content)
	for _, want := range []string{"MPD: online", "▸ [1] Songs", "›   First", "Playing · Second", "0:15 / 1:00", "█", "▀ ▀"} {
		if !strings.Contains(view, want) {
			t.Errorf("missing %q in colorless view", want)
		}
	}
	for _, line := range strings.Split(view, "\n") {
		if width := ansi.StringWidth(line); width > m.width {
			t.Errorf("wide view exceeds terminal width: %d cells", width)
		}
	}
	m.Update(tea.WindowSizeMsg{Width: 38, Height: 12})
	view = ansi.Strip(m.View().Content)
	if strings.Contains(view, "SELECTION") || !strings.Contains(view, "Tab: Songs") {
		t.Errorf("narrow view did not use compact layout: %q", view)
	}
}

func TestBottomBarsUseThemeAccents(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m.connected = true
	m.status.State = "pause"
	view := m.View().Content
	for _, want := range []string{m.styles.AccentText.Render("Paused"), m.styles.Shortcut.Render("[/]")} {
		if !strings.Contains(view, want) {
			t.Errorf("missing styled bottom-bar text %q", want)
		}
	}
}

func TestPlaybackBarUsesAvailableWidth(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.connected = true
	m.status = music.Status{State: "play", Elapsed: 5 * time.Second, Duration: 10 * time.Second}

	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	if empty := m.progressBar(10, 0, time.Second); !strings.Contains(empty, "\x1b[48;") ||
		ansi.Strip(empty) != "▄"+strings.Repeat(" ", 8)+"▀" {
		t.Fatalf("the unplayed bar should use background color, not outlined glyphs: %q", empty)
	}
	const playback = "Playing · 0:05 / 0:10"
	cells := 86 - ansi.StringWidth(playback) - 2
	want := playback + "  ▄" + strings.Repeat("█", cells/2-1) + strings.Repeat(" ", cells-cells/2-1) + "▀"
	if got := ansi.Strip(m.playbackWithProgress()); got != want {
		t.Fatalf("progress should fill the space after playback text: %q", got)
	}

	m.Update(tea.WindowSizeMsg{Width: 26, Height: 12})
	if got := ansi.Strip(m.playbackWithProgress()); got != playback {
		t.Fatalf("compact playback should keep the time and omit the bar: %q", got)
	}

	if got := ansi.Strip(m.progressBar(86, -time.Second, m.status.Duration)); got != "▄"+strings.Repeat(" ", 84)+"▀" {
		t.Fatalf("negative progress: %q", got)
	}
	if got := ansi.Strip(m.progressBar(86, 11*time.Second, m.status.Duration)); got != "▄"+strings.Repeat("█", 84)+"▀" {
		t.Fatalf("progress beyond the end: %q", got)
	}
}
