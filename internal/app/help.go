package app

import (
	"fmt"
	"strings"

	"rampd/internal/keymap"
)

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
	h := m.keys.Hint
	return []string{
		joinHints(h(keymap.NextTab, "Next view"), h(keymap.PreviousTab, "Previous view")),
		joinHints(h(keymap.Down, "Down"), h(keymap.Up, "Up"), h(keymap.Back, "Back"), h(keymap.Forward, "Open")),
		joinHints(h(keymap.Open, "Open or play"), h(keymap.Append, "Append to queue")),
		joinHints(h(keymap.Filter, "Filter this list"), h(keymap.Cancel, "Clear filter"), h(keymap.Refresh, "Refresh")),
		joinHints(h(keymap.PlayPause, "Play/pause"), h(keymap.Next, "Next"), h(keymap.Previous, "Previous")),
		joinHints(h(keymap.SeekBack, "Seek back"), h(keymap.SeekForward, "Seek forward"), h(keymap.Random, "Random"), h(keymap.Repeat, "Repeat")),
		"Queue: " + joinHints(h(keymap.Remove, "Remove"), h(keymap.MoveDown, "Move down"), h(keymap.MoveUp, "Move up"), h(keymap.Shuffle, "Shuffle")),
		"Queue: " + joinHints(h(keymap.Clear, "Clear"), h(keymap.Save, "Save as playlist")),
		"Albums/Songs: " + h(keymap.SortAlbums, "Change sort order"),
		"Track lists: " + h(keymap.SortAlbums, "Switch artist or album column"),
		joinHints(h(keymap.AddTray, "Collect in Tray"), h(keymap.CopyTray, "Copy live queue to Tray")),
		"Tray edit: " + joinHints(h(keymap.TrayHints, "Switch mode"), h(keymap.MarkTray, "Mark"), h(keymap.MoveDown, "Down"), h(keymap.MoveUp, "Up"), h(keymap.Remove, "Remove")),
		"Tray: " + joinHints(h(keymap.Append, "Queue all"), h(keymap.Replace, "Replace queue"), h(keymap.Save, "Save"), h(keymap.SeedDream, "Dream from Tray")),
		"Dream: " + joinHints(h(keymap.First, "Generate"), h(keymap.Open, "Play"), h(keymap.Replace, "Replace queue"), h(keymap.Save, "Save")),
		"Playlists: " + joinHints(h(keymap.NewPlaylist, "New"), h(keymap.Rename, "Rename"), h(keymap.Clear, "Delete")),
		"Open playlist: " + joinHints(h(keymap.MoveDown, "Move down"), h(keymap.MoveUp, "Move up"), h(keymap.Remove, "Remove"), h(keymap.AppendAll, "Queue all")),
		joinHints(h(keymap.AddPlaylist, "Add to playlist"), h(keymap.Commands, "Commands"), h(keymap.Quit, "Quit")),
	}
}

func (m *model) trayHints() (label string, actions []string) {
	h := m.actionHint
	if m.trayWholeMode {
		actions = append(actions, h(keymap.CopyTray, "Copy Queue"))
		if len(m.tray) > 0 {
			actions = append(actions, h(keymap.Append, "Append"), h(keymap.Replace, "Replace"),
				h(keymap.Save, "Save"), h(keymap.SeedDream, "Dream"))
		}
		return "Whole Tray", actions
	}

	label = "Edit tracks"
	if len(m.trayMarks) > 0 {
		label = fmt.Sprintf("%d marked", len(m.trayMarks))
		if p := m.page(); p.filter != "" {
			hidden := len(m.trayMarks)
			for _, row := range p.rows {
				if m.trayMarks[m.tray[row.position].key] {
					hidden--
				}
			}
			if hidden > 0 {
				label += fmt.Sprintf(" · %d hidden", hidden)
			}
		}
	}
	if len(m.tray) > 0 {
		actions = append(actions, h(keymap.MarkTray, "Mark"), h(keymap.MoveDown, "Down"),
			h(keymap.MoveUp, "Up"), h(keymap.Remove, "Remove"), h(keymap.SortAlbums, m.trackColumnCommand()))
	}
	if m.page().filter != "" {
		actions = append(actions, h(keymap.Cancel, "Clear filter"))
	} else {
		actions = append(actions, h(keymap.Filter, "Filter"))
	}
	return label, actions
}

func (m *model) trayFooter() string {
	label, actions := m.trayHints()
	h := m.actionHint
	switchMode := h(keymap.TrayHints, "Mode")
	parts := []string{switchMode, m.styles.AccentText.Render(label)}
	if m.width < 90 || switchMode == "" {
		parts = append(parts, h(keymap.Commands, "Commands"))
	}
	return strings.Join(nonemptyHints(append(parts, actions...)), "  ")
}

func (m *model) actionHints() string {
	h := m.actionHint
	var actions []string
	if m.showFilter() {
		actions = append(actions, h(keymap.Cancel, "Clear filter"))
	}
	actions = append(actions, h(keymap.Filter, "Filter"))
	if m.onMediaList() {
		actions = append(actions, h(keymap.SortAlbums, m.nextMediaSort()))
	} else if m.canSwitchTrackColumn() {
		actions = append(actions, h(keymap.SortAlbums, m.trackColumnCommand()))
	}
	switch m.tab {
	case dreamTab:
		actions = append(actions, h(keymap.First, "Generate"), h(keymap.Open, "Play session"), h(keymap.Save, "Save"))
	case queueTab:
		actions = append(actions, h(keymap.CopyTray, "Copy to Tray"), h(keymap.Open, "Play"), h(keymap.Remove, "Remove"))
	case playlistsTab:
		if m.page().playlist == "" {
			actions = append(actions, h(keymap.NewPlaylist, "New"), h(keymap.Rename, "Rename"), h(keymap.Clear, "Delete"))
		} else {
			actions = append(actions, h(keymap.MoveDown, "Move down"), h(keymap.Remove, "Remove"), h(keymap.AppendAll, "Queue all"))
		}
	default:
		actions = append(actions, h(keymap.AddTray, "Collect in Tray"), h(keymap.Open, "Open/play"), h(keymap.Append, "Append"))
	}
	return joinHints(append(actions, h(keymap.Commands, "Commands"), h(keymap.Help, "Keys"))...)
}
