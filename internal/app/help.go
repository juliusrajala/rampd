package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/keymap"
)

type helpView uint8

const (
	noHelp helpView = iota
	keysHelp
	guideHelp
)

func (m *model) guideRows() []string {
	width := max(1, m.innerWidth(m.bodyWidth()))
	sections := [...]struct{ title, body string }{
		{"Library", "Browse your MPD music library by song, album, or artist. Play a selection now, add it to the Queue, or set it aside in the Tray."},
		{"Queue", "MPD plays from this list. Changes to the Queue are live. Add tracks to play them later, or select a track to play it now."},
		{"Tray", "Collect and arrange tracks here without changing the Queue. You can copy the Queue into the Tray, add the Tray to the Queue, replace the Queue, or save the Tray as a playlist."},
		{"Dream", "Make a preview from your library, or use the Tray as a starting point. Generating a preview does not change the Tray or the Queue. You can play, queue, or save the result."},
		{"Playlists", "These are named track lists saved in MPD. Open one to play or add tracks to the Queue. You can also save tracks from the Queue, Tray, or Dream for later."},
	}
	var lines []string
	for i, section := range sections {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, m.styles.Heading.Render(section.title))
		lines = append(lines, strings.Split(ansi.Wrap(section.body, width, ""), "\n")...)
	}
	return lines
}

func (m *model) helpKey(raw string, action keymap.Action) tea.Cmd {
	if raw == "esc" {
		m.help = noHelp
		return nil
	}
	switch action {
	case keymap.Quit:
		return tea.Quit
	case keymap.Up:
		m.scrollHelp(-1)
	case keymap.Down:
		m.scrollHelp(1)
	case keymap.PageUp:
		m.scrollHelp(-m.listHeight())
	case keymap.PageDown:
		m.scrollHelp(m.listHeight())
	case keymap.First, keymap.Home:
		m.helpTop = 0
	case keymap.Last:
		m.scrollHelp(len(m.helpPageRows()))
	}
	return nil
}

func (m *model) helpPageRows() []string {
	if m.help == guideHelp {
		return m.guideRows()
	}
	return m.helpRows()
}

func (m *model) scrollHelp(delta int) {
	last := max(0, len(m.helpPageRows())-m.listHeight())
	m.helpTop = max(0, min(m.helpTop+delta, last))
}

func joinHints(hints ...string) string {
	return strings.Join(nonemptyHints(hints), "   ")
}

func nonemptyHints(hints []string) []string {
	visible := hints[:0]
	for _, hint := range hints {
		if hint != "" {
			visible = append(visible, hint)
		}
	}
	return visible
}

func (m *model) shortcut(key, label string) string {
	if key == "" {
		return ""
	}
	return m.styles.Shortcut.Render(keymap.Button(key)) + " " + m.styles.Muted.Render(label)
}

func (m *model) actionHint(action keymap.Action, label string) string {
	return m.shortcut(m.keys.Key(action), label)
}

func (m *model) helpRows() []string {
	h := m.actionHint
	heading := m.styles.AccentText.Render
	columns := [][]string{
		nonemptyHints([]string{
			heading("General"),
			h(keymap.Up, "Up"), h(keymap.Down, "Down"), h(keymap.PageUp, "Page up"), h(keymap.PageDown, "Page down"),
			h(keymap.Back, "Back"), h(keymap.Forward, "Open"), h(keymap.Open, "Open or play"),
			h(keymap.NextTab, "Next view"), h(keymap.PreviousTab, "Previous view"),
			h(keymap.Filter, "Filter"), h(keymap.Cancel, "Clear filter"), h(keymap.Refresh, "Refresh"),
			h(keymap.Commands, "Commands"), h(keymap.Help, "Keybindings"), h(keymap.Guide, "Help"), h(keymap.Quit, "Quit"),
		}),
		nonemptyHints([]string{
			heading("Playback"),
			h(keymap.PlayPause, "Play/Pause"), h(keymap.Previous, "Prev"), h(keymap.Next, "Next"),
			h(keymap.FocusSong, "Focus song"),
			h(keymap.SeekBack, "Seek back"), h(keymap.SeekForward, "Seek forward"), h(keymap.Random, "Random"), h(keymap.Repeat, "Repeat"),
		}),
		nonemptyHints([]string{
			heading("By view"),
			"Library", h(keymap.SortAlbums, "Sort"), h(keymap.AddTray, "Add to Tray"), h(keymap.Append, "Add to queue"), h(keymap.AddPlaylist, "Add to playlist"),
			"Track lists", h(keymap.SortAlbums, "Switch column"), h(keymap.MoveUp, "Move up"), h(keymap.MoveDown, "Move down"), h(keymap.Remove, "Remove track"),
			"Tray and Queue", h(keymap.CopyTray, "Copy queue"), h(keymap.MarkTray, "Mark (Tray)"), h(keymap.Shuffle, "Shuffle"), h(keymap.DedupeTray, "Dedupe Tray"),
			h(keymap.Replace, "Replace queue"), h(keymap.Clear, "Clear queue"), h(keymap.Save, "Save playlist"),
			"Dream", h(keymap.SeedDream, "Seed from Tray"), h(keymap.First, "New preview"),
			"Playlists", h(keymap.NewPlaylist, "New playlist"), h(keymap.Rename, "Rename"), h(keymap.Clear, "Delete list"), h(keymap.AppendAll, "Queue all"),
		}),
	}
	width := m.innerWidth(m.bodyWidth())
	const gap = 2
	if width < 3*21+2*gap {
		return append(append(append(columns[0], ""), append(columns[1], "")...), columns[2]...)
	}
	colWidth := (width - 2*gap) / 3
	lines := make([]string, max(len(columns[0]), len(columns[1]), len(columns[2])))
	for col, entries := range columns {
		for i := range lines {
			line := ""
			if i < len(entries) {
				line = ansi.Truncate(entries[i], colWidth, "…")
			}
			lines[i] += line
			if col < len(columns)-1 {
				lines[i] += strings.Repeat(" ", colWidth+gap-ansi.StringWidth(line))
			}
		}
	}
	return lines
}

func (m *model) playbackHints() string {
	h := m.actionHint
	width := m.innerWidth(m.bodyWidth())
	play, prev, next := "Play/Pause", "Prev", "Next"
	focus := ""
	back, forward := h(keymap.SeekBack, "−5s"), h(keymap.SeekForward, "+5s")
	if m.connected && m.status.PlayingID() >= 0 {
		focus = h(keymap.FocusSong, "Focus")
		core := strings.Join(nonemptyHints([]string{h(keymap.PlayPause, play), focus,
			h(keymap.Previous, prev), h(keymap.Next, next), back, forward}), " ")
		if focus != "" && ansi.StringWidth(core) > width {
			play, prev, next = "Play", "←", "→"
			if m.status.State == "play" {
				play = "Pause"
			}
		}
	}
	var line string
	for _, hint := range []string{
		h(keymap.PlayPause, play), focus, h(keymap.Previous, prev), h(keymap.Next, next),
		back, forward, h(keymap.Random, "Random"), h(keymap.Repeat, "Repeat"),
	} {
		if hint == "" {
			continue
		}
		candidate := hint
		if line != "" {
			candidate = line + " " + hint
		}
		if ansi.StringWidth(candidate) > width {
			break
		}
		line = candidate
	}
	if line == "" {
		return ""
	}
	return strings.Repeat(" ", (width-ansi.StringWidth(line))/2) + line
}
