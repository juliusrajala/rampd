// Package theme defines the terminal styles used by the player.
package theme

import (
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

// Development outlines the synthwave panels for layout work.
func Development() Styles {
	styles := Synthwave()
	styles.OutlinePanels = true
	styles.PanelBorder = lipgloss.NewStyle().Foreground(lipgloss.Color("#D8B4FE"))
	return styles
}

// Synthwave returns the built-in player theme.
func Synthwave() Styles {
	ink := lipgloss.Color("#180E2A")
	violet := lipgloss.Color("#40225E")
	pink := lipgloss.Color("#FF8CC9")
	cyan := lipgloss.Color("#77F5E5")
	pale := lipgloss.Color("#FFE0F7")
	slate := lipgloss.Color("#B29AC9")

	heading := lipgloss.NewStyle().Foreground(pale).Background(violet).Bold(true)
	active := lipgloss.NewStyle().Foreground(ink).Background(cyan).Bold(true)
	muted := lipgloss.NewStyle().Foreground(slate)
	progress := lipgloss.NewStyle().Background(lipgloss.Color("#241539"))
	input := textinput.DefaultDarkStyles()
	input.Focused.Prompt = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	input.Focused.Placeholder = muted
	input.Cursor.Color = cyan

	return Styles{
		Heading:           heading,
		Muted:             muted,
		Selection:         lipgloss.NewStyle().Foreground(ink).Background(pink).Bold(true),
		InactiveSelection: heading,
		ActiveTab:         active,
		AccentText:        lipgloss.NewStyle().Foreground(pink).Bold(true),
		Shortcut:          lipgloss.NewStyle().Foreground(cyan).Bold(true),
		SongName:          lipgloss.NewStyle().Foreground(lipgloss.Color("#F5EFFB")),
		ArtistName:        lipgloss.NewStyle().Foreground(lipgloss.Color("#CEB5F5")),
		AlbumName:         lipgloss.NewStyle().Foreground(lipgloss.Color("#EFB69C")),
		Online:            lipgloss.NewStyle().Foreground(cyan),
		Offline:           lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA82")),
		Divider:           lipgloss.NewStyle().Foreground(lipgloss.Color("#8D61AE")),
		ProgressFilled:    progress.Foreground(cyan),
		ProgressEmpty:     lipgloss.NewStyle().Background(lipgloss.Color("#3A3743")),
		ArtworkMatte:      lipgloss.NewStyle().Background(ink),
		Motif: [3]lipgloss.Style{
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F95DB5")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7FB2")),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#FFAA82")),
		},
		Input: input,
	}
}
