package app

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/keymap"
	"rampd/internal/music"
)

func TestConnectionStatusAlignsWithMainPanel(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	for _, size := range []tea.WindowSizeMsg{{Width: 100, Height: 24}, {Width: 90, Height: 24}, {Width: 38, Height: 12}, {Width: 18, Height: 5}} {
		m.Update(size)
		for _, state := range []string{"connecting", "online", "offline"} {
			m.connected = state == "online"
			m.connectionFailed = state == "offline"
			m.notice = "A notice is not a connection error."
			view := m.View()
			lines := strings.Split(ansi.Strip(view.Content), "\n")
			status := "MPD: " + state
			_, statusLine := findStatusLine(lines, status)
			if !strings.HasSuffix(statusLine, status) || ansi.StringWidth(statusLine) != m.sidebarWidth()+m.bodyWidth() {
				t.Errorf("status not at the main panel right edge at %dx%d: %q", size.Width, size.Height, statusLine)
			}
			notice := ansi.Cut(statusLine, m.sidebarWidth(), m.width)
			if !strings.HasPrefix(notice, "A") && !strings.HasPrefix(notice, "…") {
				t.Errorf("notice not at the main panel left edge: %q", statusLine)
			}
			if strings.Count(view.Content, status) != 1 || strings.Contains(lines[len(lines)-1], status) {
				t.Errorf("status should appear once, above the controls: %q", view.Content)
			}
		}
	}
	if got := m.styles.Online.Render("MPD: online"); strings.Contains(got, "[48;") {
		t.Errorf("online status has a background color: %q", got)
	}
}

func findStatusLine(lines []string, status string) (int, string) {
	for i, line := range lines {
		if strings.HasSuffix(line, status) {
			return i, line
		}
	}
	return -1, ""
}

func TestNavigationHighlightsConfiguredKeys(t *testing.T) {
	keys, err := keymap.New(map[string][]string{"songs": {"f"}, "albums": {"u"}})
	if err != nil {
		t.Fatal(err)
	}
	m := New(context.Background(), &fakeBackend{}, Options{Keys: keys}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	view := m.View().Content
	for _, want := range []string{
		m.styles.Shortcut.Render("[f]") + " " + m.styles.ActiveTab.Render("Songs"),
		m.styles.Shortcut.Render("[u]") + " Albums",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("missing highlighted navigation key: %q", want)
		}
	}
}

func TestOnlyListAndPlaybackHaveBorders(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	for _, tc := range []struct {
		name          string
		width, height int
		wantPanels    int
	}{
		{"wide", 100, 24, 2},
		{"narrow", 38, 24, 2},
		{"short", 90, 12, 2},
		{"too short", 90, 8, 0},
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
			statusRow, _ := findStatusLine(lines, "MPD: connecting")
			main := func(line string) string { return ansi.Cut(line, m.sidebarWidth(), m.width) }
			if statusRow < 0 || !strings.HasPrefix(main(lines[statusRow+1]), "╭ Playback") {
				t.Fatal("status should sit just above Playback")
			}
			right := m.sidebarWidth() + m.bodyWidth()
			for _, line := range lines[statusRow+1:] {
				if ansi.StringWidth(line) != right {
					t.Errorf("Playback should align with the main panel: %q", line)
				}
			}
			gap, ok := strings.CutPrefix(main(lines[statusRow+3]), "│")
			gap, end := strings.CutSuffix(gap, "│")
			if !ok || !end || strings.TrimSpace(gap) != "" {
				t.Fatalf("expected a blank line before Playback shortcuts: %q", gap)
			}
			hint := strings.Trim(main(lines[statusRow+4]), "│")
			if !strings.HasPrefix(strings.TrimSpace(hint), "[p] Play/Pause") || strings.Contains(hint, "Filter") {
				t.Fatalf("only playback keys should be inside Playback: %q", hint)
			}
			leftPad := len(hint) - len(strings.TrimLeft(hint, " "))
			rightPad := len(hint) - len(strings.TrimRight(hint, " "))
			if rightPad < leftPad || rightPad > leftPad+1 {
				t.Fatalf("Playback shortcuts not centered: %q", hint)
			}
			if !strings.HasPrefix(main(lines[len(lines)-1]), "╰") {
				t.Fatalf("unexpected footer below Playback: %q", lines[len(lines)-1])
			}
			if m.sidebarWidth() > 0 {
				leftWidth := m.sidebarWidth()
				if rightWidth := m.width - leftWidth - m.bodyWidth(); rightWidth != leftWidth {
					t.Fatalf("side columns should center the main panel: left %d, right %d", leftWidth, rightWidth)
				}
				navTop, rightTop, _ := strings.Cut(lines[0], "╭ Songs")
				_, artTop, _ := strings.Cut(rightTop, "╮")
				if strings.TrimSpace(navTop) != "" || strings.TrimSpace(artTop) != "" {
					t.Fatal("side panels should leave the first row empty")
				}
				if !strings.HasPrefix(lines[1], "  LIBRARY") || !strings.Contains(lines[1], "│    ▄") {
					t.Fatal("navigation and cover should start on the same row")
				}
				for i, label := range []string{"LIBRARY", "▸ [1] Songs", "[2] Albums", "[3] Artists", "",
					"SESSION", "[4] Tray", "[5] Queue", "", "MIXES", "[6] Playlists", "[7] Dream"} {
					nav, _, _ := strings.Cut(lines[i+1], "│")
					if !strings.Contains(nav, label) {
						t.Errorf("missing sidebar entry %q on row %d: %q", label, i+1, lines[i+1])
					}
				}
			} else if !strings.HasPrefix(lines[1], "╭ Songs") {
				t.Fatal("compact navigation should use one line above the list")
			}
		})
	}

	m.styles.OutlinePanels = false
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	view := ansi.Strip(m.View().Content)
	if strings.Contains(view, "╭") || !strings.Contains(view, "▀ ▀") {
		t.Fatal("disabling panel outlines should keep the artwork visible")
	}
}

func TestSidebarAllowsRoomForTheList(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	for _, width := range []int{64, 90, 99, 120} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		if m.bodyWidth() < minListWidth {
			t.Errorf("list is too narrow at %d columns", width)
		}
		if m.showArtworkPanel() != (width >= 2*sidePanelSpan+minListWidth) {
			t.Errorf("wrong artwork visibility at %d columns", width)
		}
	}
}

func TestSavePromptStaysBelowPlaybackFrame(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 90, Height: 24})
	m.namePlaylist(music.Action{Operation: music.SaveQueue})
	view := m.View()
	cursor := view.Cursor
	if cursor == nil || cursor.X != m.sidebarWidth()+panelInset+len("Save as: ") || cursor.Y != 22 {
		t.Fatalf("wrong save cursor position: %+v", cursor)
	}
	lines := strings.Split(ansi.Strip(view.Content), "\n")
	main := func(line string) string { return ansi.Cut(line, m.sidebarWidth(), m.width) }
	if !strings.HasPrefix(main(lines[cursor.Y-1]), "╰") || !strings.HasPrefix(main(lines[cursor.Y]), "  Save as: ") {
		t.Fatal("save prompt should be outside the playback frame")
	}
}

func TestPromptCursorStaysInsidePanel(t *testing.T) {
	m := New(context.Background(), &fakeBackend{}, Options{Artwork: true}).(*model)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	m.Update(keyRune('/'))
	m.Update(keyRune('x'))

	for _, tc := range []struct{ width, height, left, row int }{
		{100, 24, sidePanelSpan, 1},
		{38, 13, 0, 2},
	} {
		m.Update(tea.WindowSizeMsg{Width: tc.width, Height: tc.height})
		view := m.View()
		cursor := view.Cursor
		if cursor == nil || cursor.X != tc.left+panelInset+filterInset+3 || cursor.Y != tc.row {
			t.Fatalf("wrong search cursor at %dx%d: %+v", tc.width, tc.height, cursor)
		}
		line := strings.Split(view.Content, "\n")[cursor.Y]
		main := ansi.Strip(ansi.Cut(line, tc.left, tc.left+m.bodyWidth()))
		if !strings.HasPrefix(main, "│ [ / x") || !strings.HasSuffix(main, "│") {
			t.Fatalf("cursor is not on the framed search line: %q", line)
		}
	}
}
