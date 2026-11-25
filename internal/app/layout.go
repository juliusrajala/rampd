package app

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const (
	panelInset     = 2 // One border cell and one padding cell on each side.
	panelGap       = 2
	sidePanelWidth = artPanelWidth + 2*panelInset
	sidePanelSpan  = sidePanelWidth + panelGap
	minListWidth   = 39
	navRows        = len(tabNames) + 5 // Three labels and two gaps.

	// The content frame, playback frame, and borderless footer.
	bodyChromeHeight = 2 + 3 + 2
	// Compact navigation adds one line above the content frame.
	frameChromeHeight = bodyChromeHeight + 1
)

func (m *model) footerHeight() int {
	if (m.prompt == playlistNamePrompt || m.confirmation != nil) && m.height >= 6 {
		return 3 // Keep prompt text and its hints below the status row.
	}
	return 2
}

func (m *model) framed() bool {
	minHeight := frameChromeHeight + (m.footerHeight() - 2) + 1
	// Do not ask where the palette is drawn before choosing a layout.
	if m.help == noHelp && (m.prompt == searchPrompt || m.prompt == palettePrompt || m.browsePage().filter != "") {
		minHeight++ // Leave one row for results below the filter.
	}
	return m.styles.OutlinePanels && m.width >= 16 && m.height >= minHeight
}

func (m *model) sidebarWidth() int {
	if m.framed() && m.width >= minListWidth+sidePanelSpan && m.height-bodyChromeHeight-(m.footerHeight()-2) >= navRows {
		return sidePanelSpan
	}
	return 0
}

func (m *model) textInset() int {
	if m.framed() {
		return panelInset
	}
	return 0
}

func (m *model) innerWidth(width int) int {
	return width - 2*m.textInset()
}

func (m *model) playbackGap() int {
	if m.framed() && m.height >= 12 && m.prompt != playlistNamePrompt && m.confirmation == nil {
		return 1
	}
	return 0
}

func (m *model) bodyHeight() int {
	chrome := 6
	switch {
	case m.sidebarWidth() > 0:
		chrome = bodyChromeHeight
	case m.framed():
		chrome = frameChromeHeight
	case m.height < 5:
		chrome = 2
	case m.height < 8:
		chrome = 4
	}
	return max(1, m.height-chrome-(m.footerHeight()-2)-m.playbackGap())
}

func (m *model) listHeight() int {
	if m.showFilter() {
		return m.bodyHeight() - 1
	}
	return m.bodyHeight()
}

func (m *model) filterPadding() int {
	if m.innerWidth(m.bodyWidth()) > 2*filterInset {
		return filterInset
	}
	return 0
}

func (m *model) resizeInput() {
	width := m.innerWidth(m.bodyWidth())
	if m.paletteInSidebar() {
		width = m.innerWidth(sidePanelWidth)
	} else if m.inlineInput() {
		width = m.innerWidth(m.bodyWidth()) - 2*m.filterPadding()
	}
	m.input.SetWidth(max(1, min(48, width-ansi.StringWidth(m.input.Prompt)-1)))
}

func (m *model) bodyWidth() int {
	width := m.width - m.sidebarWidth()
	if m.showArtworkPanel() {
		if m.framed() {
			return width - sidePanelSpan
		}
		return width - artPanelWidth - 3
	}
	return width
}

func (m *model) frame(title string, width int, lines []string) []string {
	border := m.styles.PanelBorder
	title = " " + ansi.Truncate(title, width-4, "…") + " "
	topRule := strings.Repeat("─", width-2-ansi.StringWidth(title))
	framed := []string{border.Render("╭" + title + topRule + "╮")}
	side := border.Render("│")
	for _, line := range lines {
		line = ansi.Truncate(line, m.innerWidth(width), "…")
		padding := strings.Repeat(" ", m.innerWidth(width)-ansi.StringWidth(line))
		framed = append(framed, side+" "+line+padding+" "+side)
	}
	return append(framed, border.Render("╰"+strings.Repeat("─", width-2)+"╯"))
}

// Start each side panel one row below the top of the main panel.
func (m *model) openPanel(width, height int, rows []string) []string {
	lines := fitRows(append([]string{""}, rows...), height)
	inset := m.textInset()
	pad := strings.Repeat(" ", inset)
	for i, line := range lines {
		line = ansi.Truncate(line, m.innerWidth(width), "…")
		lines[i] = pad + line + strings.Repeat(" ", width-inset-ansi.StringWidth(line))
	}
	return lines
}

func (m *model) navGroups(tabs []string) []string {
	var lines []string
	for i, tab := range tabs {
		switch i {
		case songsTab:
			lines = append(lines, m.styles.Muted.Render("LIBRARY"))
		case trayTab:
			lines = append(lines, "", m.styles.Muted.Render("SESSION"))
		case playlistsTab:
			lines = append(lines, "", m.styles.Muted.Render("MIXES"))
		}
		lines = append(lines, tab)
	}
	return lines
}

func fitRows(lines []string, height int) []string {
	lines = lines[:min(len(lines), height)]
	return append(lines, make([]string, height-len(lines))...)
}

func joinPanels(left, right []string) []string {
	gap := strings.Repeat(" ", panelGap)
	for i := range left {
		left[i] += gap + right[i]
	}
	return left
}

func (m *model) pixelRule(width int) string {
	width = min(width, 48)
	third := width / 3
	return m.styles.Motif[0].Render(strings.Repeat("━", third)) +
		m.styles.Divider.Render(strings.Repeat("━", third)) +
		m.styles.ProgressFilled.Render(strings.Repeat("━", width-2*third))
}
