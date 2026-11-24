package app

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

const (
	panelInset = 2 // One border cell and one padding cell on each side.
	panelGap   = 2
	navWidth   = 19
	navRows    = len(tabNames) + 5 // Three labels and two gaps.

	// The content frame, playback frame, and borderless footer.
	bodyChromeHeight = 2 + 3 + 2
	// Compact navigation adds a frame with one line.
	frameChromeHeight = bodyChromeHeight + 3
)

func (m *model) framed() bool {
	minHeight := frameChromeHeight + 1
	if m.showFilter() {
		minHeight++ // Leave one row for results below the filter.
	}
	return m.styles.OutlinePanels && m.width >= 16 && m.height >= minHeight
}

func (m *model) sidebarWidth() int {
	if m.framed() && m.width >= 64 && m.height-bodyChromeHeight >= navRows {
		return navWidth + panelGap
	}
	return 0
}

func (m *model) innerWidth(width int) int {
	if m.framed() {
		return width - 2*panelInset
	}
	return width
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
	return max(1, m.height-chrome)
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
	width := m.width
	if m.inlineInput() {
		width = m.innerWidth(m.bodyWidth()) - 2*m.filterPadding()
	}
	m.input.SetWidth(max(1, min(48, width-ansi.StringWidth(m.input.Prompt)-1)))
}

func (m *model) bodyWidth() int {
	width := m.width - m.sidebarWidth()
	if m.showArtworkPanel() {
		if m.framed() {
			return width - (artPanelWidth + 2*panelInset) - panelGap
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

func (m *model) framedBody(title string, rows, tabs []string) []string {
	body := m.frame(title, m.bodyWidth(), rows)
	if m.sidebarWidth() > 0 {
		nav := m.frame("Navigation", navWidth, fitRows(m.navGroups(tabs), len(rows)))
		body = joinPanels(nav, body)
	}
	if m.showArtworkPanel() {
		art := m.frame("SELECTION", artPanelWidth+2*panelInset, fitRows(m.artworkRows(), len(rows)))
		body = joinPanels(body, art)
	}
	return body
}

func (m *model) navGroups(tabs []string) []string {
	var lines []string
	for i, tab := range tabs {
		switch i {
		case artistsTab:
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
