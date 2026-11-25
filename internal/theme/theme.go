// Package theme defines the terminal styles used by the player.
package theme

import (
	"encoding/hex"
	"fmt"
	"image/color"
	"maps"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
)

// Styles names the visual roles used by the player.
type Styles struct {
	Heading, Muted, Selection, InactiveSelection lipgloss.Style
	ActiveTab, Divider                           lipgloss.Style
	AccentText, Shortcut                         lipgloss.Style
	SongName, ArtistName, AlbumName              lipgloss.Style
	Online, Offline                              lipgloss.Style
	ProgressFilled, ProgressEmpty                lipgloss.Style
	ArtworkMatte                                 lipgloss.Style
	Motif                                        [3]lipgloss.Style
	Input                                        textinput.Styles
	OutlinePanels                                bool
	PanelBorder                                  lipgloss.Style
}

var synthwave = map[string]string{
	"heading_text": "#FFE0F7", "heading_background": "#40225E",
	"muted": "#B29AC9", "accent": "#FF8CC9", "shortcut": "#77F5E5",
	"selection_text": "#180E2A", "selection_background": "#FF8CC9",
	"active_tab_text": "#180E2A", "active_tab_background": "#77F5E5",
	"song": "#F5EFFB", "artist": "#CEB5F5", "album": "#EFB69C",
	"panel_border": "#D8B4FE", "divider": "#8D61AE",
	"online": "#77F5E5", "offline": "#FFAA82",
	"progress_played": "#77F5E5", "progress_unplayed": "#3A3743",
	"artwork_background": "#180E2A",
	"motif_top":          "#F95DB5", "motif_middle": "#FF7FB2", "motif_bottom": "#FFAA82",
}

// Default returns the built-in Synthwave styles.
func Default() Styles {
	return makeStyles(synthwave)
}

// FromConfig applies validated color overrides without changing the defaults.
func FromConfig(overrides map[string]string) (Styles, error) {
	palette := maps.Clone(synthwave)
	for role, value := range overrides {
		if _, ok := palette[role]; !ok {
			return Styles{}, fmt.Errorf("unknown theme setting %q", role)
		}
		if len(value) != 7 || value[0] != '#' {
			return Styles{}, fmt.Errorf("theme.%s must be a #RRGGBB color", role)
		}
		if _, err := hex.DecodeString(value[1:]); err != nil {
			return Styles{}, fmt.Errorf("theme.%s must be a #RRGGBB color", role)
		}
		palette[role] = value
	}
	return makeStyles(palette), nil
}

func makeStyles(palette map[string]string) Styles {
	value := func(role string) color.Color { return lipgloss.Color(palette[role]) }
	foreground := func(role string) lipgloss.Style { return lipgloss.NewStyle().Foreground(value(role)) }
	background := func(role string) lipgloss.Style { return lipgloss.NewStyle().Background(value(role)) }

	heading := foreground("heading_text").Background(value("heading_background")).Bold(true)
	muted := foreground("muted")
	input := textinput.DefaultDarkStyles()
	input.Focused.Prompt = foreground("shortcut").Bold(true)
	input.Focused.Placeholder = muted
	input.Cursor.Color = value("shortcut")

	return Styles{
		Heading:           heading,
		Muted:             muted,
		Selection:         foreground("selection_text").Background(value("selection_background")).Bold(true),
		InactiveSelection: heading,
		ActiveTab:         foreground("active_tab_text").Background(value("active_tab_background")).Bold(true),
		AccentText:        foreground("accent").Bold(true),
		Shortcut:          foreground("shortcut").Bold(true),
		SongName:          foreground("song"),
		ArtistName:        foreground("artist"),
		AlbumName:         foreground("album"),
		Online:            foreground("online"),
		Offline:           foreground("offline"),
		Divider:           foreground("divider"),
		ProgressFilled:    foreground("progress_played"),
		ProgressEmpty:     background("progress_unplayed"),
		ArtworkMatte:      background("artwork_background"),
		Motif:             [3]lipgloss.Style{foreground("motif_top"), foreground("motif_middle"), foreground("motif_bottom")},
		Input:             input,
		OutlinePanels:     true,
		PanelBorder:       foreground("panel_border"),
	}
}
