package app

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/keymap"
)

func TestEmptyDirectionsCenteredWithCurrentKeys(t *testing.T) {
	keys, err := keymap.New(map[string][]string{"new_playlist": {"u"}})
	if err != nil {
		t.Fatal(err)
	}
	m := New(context.Background(), &fakeBackend{}, Options{Keys: keys}).(*model)
	m.tab = playlistsTab
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	lines := m.emptyRows()
	width := m.innerWidth(m.bodyWidth())
	first := (m.listHeight() - 2) / 2
	for i, want := range []string{"No stored playlists.", "[u] Create playlist"} {
		line := ansi.Strip(lines[first+i])
		wantLine := strings.Repeat(" ", (width-ansi.StringWidth(want))/2) + want
		if line != wantLine {
			t.Errorf("expected centered line %q, got %q", wantLine, line)
		}
	}

	m.Update(tea.WindowSizeMsg{Width: 12, Height: 5})
	if lines = m.emptyRows(); len(lines) != 1 || !strings.HasPrefix(ansi.Strip(lines[0]), "No stored") {
		t.Fatalf("short panel must keep the main message: %q", lines)
	}
}
