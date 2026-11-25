package app

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/music"
	"rampd/internal/theme"
)

func TestViewShowsStateWithoutColor(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m.Update(prepareLibrary(fixtureTracks, nil))
	m.connected = true
	m.status = music.Status{State: "play", SongID: 1, Elapsed: 15 * time.Second, Duration: time.Minute}
	m.queue = []music.Entry{{ID: 1, Track: fixtureTracks[0]}}
	m.storeArtwork(artworkMsg{uri: fixtureTracks[1].URI, revision: m.artworkRevision})

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

func TestTrackLengthsAndProgressShareThemeColor(t *testing.T) {
	styles, err := theme.FromConfig(map[string]string{"progress_played": "#3456CC"})
	if err != nil {
		t.Fatal(err)
	}
	tracks := []music.Track{
		{URI: "one", Title: "One", Duration: time.Minute}, {URI: "two", Title: "Two", Duration: 90 * time.Second},
	}
	m := New(context.Background(), &fakeBackend{}, Options{Styles: &styles}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m.Update(prepareLibrary(tracks, nil))
	m.connected = true
	m.status = music.Status{State: "play", Duration: time.Minute, Elapsed: 30 * time.Second}
	if !strings.Contains(m.viewRows()[1], styles.ProgressFilled.Render("    1:30")) {
		t.Fatal("unselected duration did not use the progress color")
	}
	if !strings.Contains(m.playbackWithProgress(), styles.ProgressFilled.Render("0:30 / 1:00")) {
		t.Fatal("playback times did not use the progress color")
	}
	if !strings.Contains(strings.Join(m.artworkRows(), "\n"), styles.ProgressFilled.Render("1:00")) {
		t.Fatal("selection duration did not use the progress color")
	}
}

func TestPlaybackKeepsTimeOnNarrowMainPanel(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m.connected = true
	m.status = music.Status{State: "play", SongID: 5, Elapsed: 5 * time.Second, Duration: time.Minute}
	m.queue = []music.Entry{{ID: 5, Track: music.Track{Title: strings.Repeat("Long title ", 8), Artist: "Artist"}}}
	line := ansi.Strip(m.playbackLine())
	if !strings.HasPrefix(line, "Playing · Long") || !strings.HasSuffix(line, "… · 0:05 / 1:00") ||
		ansi.StringWidth(line) > m.innerWidth(m.bodyWidth()) {
		t.Fatalf("playback did not keep time visible in the narrower panel: %q", line)
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
	cells := m.innerWidth(m.bodyWidth()) - ansi.StringWidth(playback) - 2
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
