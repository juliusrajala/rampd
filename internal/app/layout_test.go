package app

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/keymap"
	"rampd/internal/music"
	"rampd/internal/theme"
)

func TestConnectionStatusInBottomRight(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	for _, size := range []tea.WindowSizeMsg{{Width: 90, Height: 24}, {Width: 38, Height: 12}, {Width: 18, Height: 5}} {
		m.Update(size)
		for _, state := range []string{"connecting", "online", "offline"} {
			m.connected = state == "online"
			m.connectionFailed = state == "offline"
			m.notice = "A notice is not a connection error."
			view := m.View()
			lines := strings.Split(ansi.Strip(view.Content), "\n")
			status := "MPD: " + state
			last := lines[len(lines)-1]
			if !strings.HasSuffix(last, status) || ansi.StringWidth(last) != size.Width {
				t.Errorf("status not at the right edge at %dx%d: %q", size.Width, size.Height, last)
			}
			if strings.Contains(strings.Join(lines[:len(lines)-1], "\n"), status) {
				t.Errorf("duplicate status outside footer: %q", view.Content)
			}
		}
	}
	if got := m.styles.Online.Render("MPD: online"); strings.Contains(got, "[48;") {
		t.Errorf("online status has a background color: %q", got)
	}
}

func TestNavigationHighlightsConfiguredKeys(t *testing.T) {
	keys, err := keymap.New(map[string][]string{"artists": {"f"}, "albums": {"u"}})
	if err != nil {
		t.Fatal(err)
	}
	m := New(context.Background(), &fakeBackend{}, Options{Keys: keys}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	view := m.View().Content
	for _, want := range []string{
		m.styles.Shortcut.Render("[f]") + " " + m.styles.ActiveTab.Render("Artists"),
		m.styles.Shortcut.Render("[u]") + " Albums",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("missing highlighted navigation key: %q", want)
		}
	}
}

func TestDevelopmentPanelOutlines(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	for _, tc := range []struct {
		name          string
		width, height int
		wantPanels    int
	}{
		{"wide", 90, 24, 4},
		{"narrow", 38, 24, 3},
		{"short", 90, 12, 3},
		{"too short", 90, 10, 0},
		{"too narrow", 15, 24, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m.Update(tea.WindowSizeMsg{Width: tc.width, Height: tc.height})
			view := ansi.Strip(m.View().Content)
			if top, bottom := strings.Count(view, "╭"), strings.Count(view, "╯"); top != tc.wantPanels || bottom != tc.wantPanels {
				t.Fatalf("expected %d closed panels, got %d top and %d bottom corners:\n%s", tc.wantPanels, top, bottom, view)
			}
			if tc.wantPanels == 0 {
				return
			}
			lines := strings.Split(view, "\n")
			if len(lines) != tc.height {
				t.Fatalf("expected %d rows, got %d", tc.height, len(lines))
			}
			for _, line := range lines[:len(lines)-2] {
				if width := ansi.StringWidth(line); width != tc.width {
					t.Errorf("expected %d columns, got %d: %q", tc.width, width, line)
				}
			}
			if hint := lines[len(lines)-1]; !strings.HasPrefix(hint, "[/] Filter") {
				t.Fatalf("key hints should be below the playback border: %q", hint)
			}
			if tc.wantPanels == 4 {
				if !strings.HasPrefix(view, "╭ Navigation") || !strings.Contains(lines[0], "╮  ╭ Artists") ||
					!strings.Contains(lines[0], "╮  ╭ SELECTION") {
					t.Fatal("navigation, content, and artwork should share the same top row")
				}
				for i, label := range []string{"LIBRARY", "▸ [1] Artists", "[2] Albums", "[3] Songs", "",
					"SESSION", "[4] Tray", "[5] Queue", "", "MIXES", "[6] Playlists", "[7] Dream"} {
					if !strings.Contains(strings.SplitN(lines[i+1], "│  │", 2)[0], label) {
						t.Errorf("missing sidebar entry %q on row %d: %q", label, i+1, lines[i+1])
					}
				}
				if strings.Contains(view, "rämpd player") {
					t.Fatal("standalone title should not be rendered")
				}
			}
		})
	}

	m.styles = theme.Synthwave()
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	view := ansi.Strip(m.View().Content)
	if strings.Contains(view, "╭") || !strings.Contains(view, "SELECTION") {
		t.Fatal("the normal synthwave theme should show artwork without outlines")
	}
}

func TestSavePromptStaysBelowPlaybackFrame(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.namePlaylist(music.Action{Operation: music.SaveQueue})
	view := m.View()
	cursor := view.Cursor
	if cursor == nil || cursor.X != len("Save as: ") || cursor.Y != 22 {
		t.Fatalf("wrong save cursor position: %+v", cursor)
	}
	lines := strings.Split(ansi.Strip(view.Content), "\n")
	if !strings.HasPrefix(lines[cursor.Y-1], "╰") || !strings.HasPrefix(lines[cursor.Y], "Save as: ") {
		t.Fatal("save prompt should be outside the playback frame")
	}
}

func TestPromptCursorStaysInsidePanel(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.Update(keyRune('/'))
	m.Update(keyRune('x'))

	for _, tc := range []struct{ width, height, left, row int }{
		{90, 24, navWidth + panelGap, 1},
		{38, 13, 0, 4},
	} {
		m.Update(tea.WindowSizeMsg{Width: tc.width, Height: tc.height})
		view := m.View()
		cursor := view.Cursor
		if cursor == nil || cursor.X != tc.left+panelInset+filterInset+3 || cursor.Y != tc.row {
			t.Fatalf("wrong search cursor at %dx%d: %+v", tc.width, tc.height, cursor)
		}
		line := strings.Split(ansi.Strip(view.Content), "\n")[cursor.Y]
		if !strings.HasPrefix(string([]rune(line)[tc.left:]), "│ [ / x") || !strings.HasSuffix(line, "│") {
			t.Fatalf("cursor is not on the framed search line: %q", line)
		}
	}
}
