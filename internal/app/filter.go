package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

const filterInset = 2 // "[ " before the text and " ]" after it.

func (m *model) focusFilter() tea.Cmd {
	m.prompt = searchPrompt
	m.input.Prompt = "/ "
	m.input.Placeholder = "Filter this list"
	m.input.SetValue(m.page().filter)
	m.input.CursorEnd()
	m.resizeInput()
	return m.input.Focus()
}

func (m *model) updateInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if m.inlineInput() {
		m.page().setFilter(strings.TrimSpace(m.input.Value()))
	}
	return cmd
}

func (m *model) inlineInput() bool {
	return m.prompt == searchPrompt || m.prompt == palettePrompt
}

func (m *model) showFilter() bool {
	return !m.showHelp && (m.inlineInput() || m.page().filter != "")
}

func (m *model) filterLine() string {
	value := "/ " + clean(m.page().filter)
	if m.inlineInput() {
		value = m.input.View()
	}
	width := m.innerWidth(m.bodyWidth())
	padding := m.filterPadding()
	if padding == 0 {
		return ansi.Truncate(value, width, "…")
	}
	border := m.styles.Divider
	return border.Render("[ ") + ansi.Truncate(value, width-2*padding, "…") + border.Render(" ]")
}

func filterRows(rows []row, query string) []row {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return rows
	}
	var matches []row
	for _, row := range rows {
		match := true
		for _, word := range words {
			if !strings.Contains(row.matchLabel, word) && !strings.Contains(row.matchDetail, word) {
				match = false
				break
			}
		}
		if match {
			matches = append(matches, row)
		}
	}
	return matches
}
