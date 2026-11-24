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

func TestLibraryMetadataColumns(t *testing.T) {
	tracks := []music.Track{
		{URI: "1", Artist: "Artist A", Album: "Short", Title: "Song One", Duration: 3*time.Minute + 17*time.Second},
		{URI: "2", Artist: "Artist A", Album: "Short", Title: "Song Two", Duration: 2*time.Minute + 30*time.Second},
		{URI: "3", Artist: "Artist A", Album: "Unknown", Title: "Song Three"},
		{URI: "4", Artist: "Artist B", Album: "Long", Title: "Song Four", Duration: time.Hour + time.Second},
	}
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.Update(prepareLibrary(tracks, nil))
	check := func(i int, parts ...string) string {
		t.Helper()
		line := ansi.Strip(m.viewRows()[i])
		if width := m.innerWidth(m.bodyWidth()); ansi.StringWidth(line) != width {
			t.Errorf("row does not fit %d columns: %q", width, line)
		}
		for _, part := range parts {
			if !strings.Contains(line, part) {
				t.Errorf("missing %q in %q", part, line)
			}
		}
		return line
	}

	first := check(0, "Artist A", "Albums: 2 - Songs: 3")
	second := check(1, "Artist B", "Albums: 1 - Songs: 1")
	firstTitle, _, _ := strings.Cut(first, "Albums:")
	secondTitle, _, _ := strings.Cut(second, "Albums:")
	if ansi.StringWidth(firstTitle) != ansi.StringWidth(secondTitle) {
		t.Fatal("combined count columns should share a left edge")
	}

	m.Update(keyRune('2'))
	if line := check(0, "Long", "Artist B"); !strings.HasSuffix(line, "1:00:01") {
		t.Errorf("missing hour-length album duration: %q", line)
	}
	if line := check(1, "Short", "Artist A"); !strings.HasSuffix(line, "5:47") {
		t.Errorf("missing total album duration: %q", line)
	}
	if line := check(2, "Unknown"); !strings.HasSuffix(line, "—") {
		t.Errorf("missing unknown duration marker: %q", line)
	}
	m.Update(keyRune('j'))
	m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if line := check(0, "Song One"); !strings.HasSuffix(line, "3:17") {
		t.Errorf("missing duration inside album: %q", line)
	}
	m.Update(keyRune('3'))
	if line := check(0, "Song Four", "Artist B"); !strings.HasSuffix(line, "1:00:01") || strings.Contains(line, "Long") {
		t.Errorf("Songs should show title, artist and duration, but not the album: %q", line)
	}
	m.Update(tea.WindowSizeMsg{Width: 18, Height: 12})
	if line := ansi.Strip(m.viewRows()[0]); !strings.Contains(line, "Song Four") || strings.Contains(line, "1:00:01") {
		t.Errorf("narrow row should prioritize title: %q", line)
	}
}
