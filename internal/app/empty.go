package app

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"rampd/internal/keymap"
)

func (m *model) emptyRows() []string {
	title, hints := m.emptyMessage()
	lines := append([]string{m.styles.AccentText.Render(title)}, hints...)
	height := m.listHeight()
	if height <= 0 {
		return nil
	}
	lines = lines[:min(len(lines), height)]
	width := m.innerWidth(m.bodyWidth())
	// Short panels keep the message first. Taller panels center the whole group.
	rows := make([]string, (height-len(lines))/2)
	for _, line := range lines {
		line = ansi.Truncate(line, width, "…")
		rows = append(rows, strings.Repeat(" ", (width-ansi.StringWidth(line))/2)+line)
	}
	return rows
}

func (m *model) emptyMessage() (string, []string) {
	p := m.page()
	hint := m.actionHint
	directions := func(title string, lines ...string) (string, []string) {
		return title, nonemptyHints(lines)
	}
	switch {
	case m.prompt == palettePrompt:
		return "No matching commands.", nil
	case p.filter != "":
		if m.prompt == searchPrompt {
			return "No matching entries.", nil
		}
		return directions("No matching entries.", hint(keymap.Cancel, "Clear filter"))
	case m.tab == trayTab:
		copyHint := hint(keymap.TrayHints, "Switch to Whole Tray to copy")
		if m.trayWholeMode {
			copyHint = hint(keymap.CopyTray, "Copy live queue")
		}
		return directions("Tray is empty.", hint(keymap.AddTray, "Collect tracks while browsing"), copyHint)
	case m.tab == queueTab:
		return directions("Queue is empty.", hint(keymap.Append, "Append tracks while browsing"))
	case m.tab == dreamTab:
		return directions("Rediscover your library.", hint(keymap.First, "Generate a preview"))
	case m.tab == playlistsTab:
		if m.playlistsLoading || p.playlist != "" && m.playlistLoading {
			return "Reading MPD playlists…", nil
		}
		if p.playlist != "" {
			return directions("No tracks in this playlist.", hint(keymap.Refresh, "Reload playlist"))
		}
		return directions("No stored playlists.", hint(keymap.NewPlaylist, "Create playlist"))
	case m.libraryLoading:
		return "Reading the MPD library…", nil
	case m.libraryError != nil && !m.libraryLoaded:
		return directions(clean("Library: "+m.libraryError.Error()), hint(keymap.Refresh, "Retry"))
	default:
		return directions("No indexed tracks.", hint(keymap.Refresh, "Reload library"))
	}
}
