// Package keymap assigns terminal keys to named player actions.
package keymap

import (
	"fmt"
	"strings"
	"unicode"
)

type Action string

const (
	Quit        Action = "quit"
	Help        Action = "help"
	Artists     Action = "artists"
	Albums      Action = "albums"
	Songs       Action = "songs"
	Tray        Action = "tray"
	Queue       Action = "queue"
	Dream       Action = "dream"
	Playlists   Action = "playlists"
	NextTab     Action = "next_tab"
	PreviousTab Action = "previous_tab"
	Down        Action = "down"
	Up          Action = "up"
	PageDown    Action = "page_down"
	PageUp      Action = "page_up"
	First       Action = "first"
	Home        Action = "home"
	Last        Action = "last"
	Back        Action = "back"
	Cancel      Action = "cancel"
	Forward     Action = "forward"
	Open        Action = "open"
	Append      Action = "append"
	Filter      Action = "filter"
	Refresh     Action = "refresh"
	NewPlaylist Action = "new_playlist"
	Rename      Action = "rename_playlist"
	AddPlaylist Action = "add_to_playlist"
	AppendAll   Action = "append_playlist"
	Save        Action = "save_playlist"
	PlayPause   Action = "play_pause"
	Next        Action = "next"
	Previous    Action = "previous"
	SeekBack    Action = "seek_back"
	SeekForward Action = "seek_forward"
	Random      Action = "random"
	Repeat      Action = "repeat"
	Remove      Action = "remove"
	MoveDown    Action = "move_down"
	MoveUp      Action = "move_up"
	Shuffle     Action = "shuffle"
	ShuffleTray Action = "shuffle_tray"
	SortAlbums  Action = "sort_albums"
	Clear       Action = "clear_or_delete"
	Replace     Action = "replace_queue"
	Commands    Action = "commands"
	AddTray     Action = "add_to_tray"
	MarkTray    Action = "mark_tray"
	TrayHints   Action = "tray_hints"
	CopyTray    Action = "copy_queue_to_tray"
	SeedDream   Action = "dream_from_tray"
)

var defaults = map[Action][]string{
	Quit: {"q"}, Help: {"?"}, Songs: {"1"}, Albums: {"2"}, Artists: {"3"}, Tray: {"4"}, Queue: {"5"}, Playlists: {"6"}, Dream: {"7"},
	NextTab: {"tab"}, PreviousTab: {"shift+tab"}, Down: {"j", "down"}, Up: {"k", "up"},
	PageDown: {"pgdown"}, PageUp: {"pgup"}, First: {"g"}, Home: {"home"}, Last: {"G", "end"},
	Back: {"h", "left", "backspace"}, Cancel: {"esc"}, Forward: {"l", "right"}, Open: {"enter"},
	Append: {"space"}, Filter: {"/"}, Refresh: {"ctrl+r"}, NewPlaylist: {"c"}, Rename: {"e"},
	AddPlaylist: {"A"}, AppendAll: {"a"}, Save: {"w"}, PlayPause: {"p"}, Next: {"n"}, Previous: {"b"},
	SeekBack: {"["}, SeekForward: {"]"}, Random: {"r"}, Repeat: {"R"}, Remove: {"x"}, MoveDown: {"J"},
	MoveUp: {"K"}, Shuffle: {"s"}, SortAlbums: {"o"}, ShuffleTray: nil, Clear: {"X"}, Replace: {"P"}, Commands: {":"},
	AddTray: {"t"}, MarkTray: {"v"}, TrayHints: {"ctrl+t"}, CopyTray: {"C"}, SeedDream: {"D"},
}

type Bindings struct {
	keys    map[string]Action
	primary map[Action]string
}

func Default() Bindings {
	b, _ := New(nil)
	return b
}

func New(overrides map[string][]string) (Bindings, error) {
	for name := range overrides {
		if _, ok := defaults[Action(name)]; !ok {
			return Bindings{}, fmt.Errorf("unknown keybinding action %q", name)
		}
	}
	b := Bindings{keys: make(map[string]Action), primary: make(map[Action]string)}
	for action, keys := range defaults {
		if override, ok := overrides[string(action)]; ok {
			keys = override
		}
		for i, key := range keys {
			if key == "" || strings.ContainsFunc(key, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) }) {
				return Bindings{}, fmt.Errorf("invalid key for %q; use \"space\" for the space bar", action)
			}
			if key == "ctrl+c" {
				return Bindings{}, fmt.Errorf("ctrl+c is reserved for quitting")
			}
			if other, exists := b.keys[key]; exists {
				return Bindings{}, fmt.Errorf("key %q is assigned to %q and %q", key, other, action)
			}
			b.keys[key] = action
			if i == 0 {
				b.primary[action] = key
			}
		}
	}
	return b, nil
}

func (b Bindings) IsZero() bool { return b.keys == nil }

func (b Bindings) Action(key string) Action { return b.keys[key] }

// Key returns the primary key for an action, or an empty string.
func (b Bindings) Key(action Action) string { return b.primary[action] }

// Button formats a key label without changing the binding.
func Button(key string) string {
	if key == "" {
		return ""
	}
	return "[" + key + "]"
}

// Hint shows the primary key for an action. Unbound actions have no hint.
func (b Bindings) Hint(action Action, label string) string {
	if key := b.primary[action]; key != "" {
		return Button(key) + " " + label
	}
	return ""
}
