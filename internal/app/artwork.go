package app

import (
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/artwork"
)

const (
	artRowCount   = 12
	artPanelWidth = 2 * artRowCount // Each terminal cell holds two vertical pixels.
	artCacheLimit = 16
)

type artworkMsg struct {
	uri      string
	lines    []string
	revision uint64
}

func (m *model) showArtworkPanel() bool {
	needed := artRowCount + 2 // Title and at least one metadata row.
	if !m.framed() {
		needed++ // The unframed panel uses one line for top padding.
	}
	width := m.width - m.sidebarWidth()
	minWidth := 76
	if m.sidebarWidth() > 0 {
		// Keep enough room for the main frame between the side panels.
		minWidth = minListWidth + sidePanelSpan
	}
	return m.artworkEnabled && width >= minWidth && m.bodyHeight() >= needed && !m.showHelp && m.prompt != palettePrompt
}

func (r row) artworkURI() string {
	if r.artURI != "" {
		return r.artURI
	}
	// Artist and album rows use their first track as a representative cover.
	if len(r.tracks) > 0 {
		return r.tracks[0].URI
	}
	return ""
}

func (m *model) artIndex(uri string) int {
	return slices.IndexFunc(m.artworkCache, func(a artworkMsg) bool { return a.uri == uri })
}

func (m *model) requestArtwork() tea.Cmd {
	if !m.connected || !m.showArtworkPanel() || m.artworkLoading != "" {
		return nil
	}
	selected, ok := m.selected()
	if !ok {
		return nil
	}
	uri := selected.artworkURI()
	if uri == "" || m.artIndex(uri) >= 0 {
		return nil
	}
	ctx, server, revision := m.ctx, m.server, m.artworkRevision
	m.artworkLoading = uri
	return func() tea.Msg {
		msg := artworkMsg{uri: uri, revision: revision}
		if data, err := server.Artwork(ctx, uri); err == nil && len(data) > 0 {
			if img, err := artwork.Decode(data); err == nil {
				msg.lines = artwork.SquareCover(img, artRowCount)
			}
		}
		return msg
	}
}

func (m *model) invalidateArtwork() {
	m.artworkCache = nil
	m.artworkRevision++
	// Keep the in-flight slot until its result arrives. Update can then
	// fetch the current selection without starting a second request in parallel.
}

func (m *model) storeArtwork(msg artworkMsg) {
	m.artworkLoading = ""
	if msg.revision != m.artworkRevision {
		return
	}
	m.artworkCache = append(m.artworkCache, msg)
	if len(m.artworkCache) > artCacheLimit {
		m.artworkCache = slices.Delete(m.artworkCache, 0, 1)
	}
}

func (m *model) artworkRows() []string {
	selected, ok := m.selected()
	var details []string
	if ok {
		details = append(details, m.rowNameStyle(selected).Render(clean(selected.label)))
		for _, field := range m.selectionFields(selected) {
			if field.value != "" {
				key := m.styles.Shortcut.Width(6).Render(field.label)
				value := m.styles.Muted
				switch field.label {
				case "Artist":
					value = m.styles.ArtistName
				case "Album":
					value = m.styles.AlbumName
				case "Length":
					if field.value != "—" {
						value = m.styles.ProgressFilled
					}
				}
				details = append(details, key+"  "+value.Render(clean(field.value)))
			}
		}
	} else {
		details = append(details, m.styles.Muted.Render("Nothing selected"))
	}

	// Leave room for the title and the first three fields before adding a gap.
	height := m.bodyHeight() - 1 // The side panel adds one line of top padding.
	if m.framed() {
		height += 2 // Use the height of the main frame, not just its contents.
	}
	panel := m.selectionCover(selected.artworkURI())
	if height > artRowCount+min(4, len(details)) {
		panel = append(panel, "")
	}
	return append(panel, details...)
}

type detailField struct {
	label, value string
}

func (m *model) selectionFields(item row) []detailField {
	switch {
	case item.children != nil && item.detail == "":
		return []detailField{
			{"Albums", strconv.Itoa(len(item.children))},
			{"Songs", strconv.Itoa(len(item.tracks))},
		}
	case item.children != nil:
		return []detailField{
			{"Artist", item.detail},
			{"Length", durationLabel(item.duration)},
			{"Songs", strconv.Itoa(len(item.tracks))},
		}
	}

	t, ok := m.rowTrack(item)
	if !ok {
		return nil
	}
	fields := []detailField{
		{"Artist", trackArtist(t)},
		{"Album", t.Album},
		{"Length", durationLabel(t.Duration)},
		{"Genre", t.Genre},
	}
	if t.Number > 0 {
		fields = append(fields, detailField{"Track", strconv.Itoa(t.Number)})
	}
	if t.Disc > 0 {
		fields = append(fields, detailField{"Disc", strconv.Itoa(t.Disc)})
	}
	return fields
}

func (m *model) selectionCover(uri string) []string {
	if i := m.artIndex(uri); i >= 0 {
		if art := m.artworkCache[i].lines; len(art) > 0 {
			return m.artworkTile(art, "")
		}
	} else if uri != "" && m.connected {
		return m.artworkTile(nil, "Loading artwork…")
	}
	return m.artworkTile(m.pixelWordmark(), "")
}

func (m *model) artworkTile(lines []string, caption string) []string {
	mat := m.styles.ArtworkMatte
	blank := mat.Render(strings.Repeat(" ", artPanelWidth))
	tile := make([]string, artRowCount)
	for i := range tile {
		tile[i] = blank
	}
	// Match the cover's half-block corners on the placeholder and loading tiles.
	edge := mat.UnsetBackground().Foreground(mat.GetBackground())
	matteEdge := func(corner, label string) string {
		const width = artPanelWidth - 2
		label = ansi.Truncate(label, width, "…")
		space := width - ansi.StringWidth(label)
		return edge.Render(corner) + mat.Render(strings.Repeat(" ", space/2)) +
			m.styles.Muted.Inherit(mat).Render(label) +
			mat.Render(strings.Repeat(" ", space-space/2)) + edge.Render(corner)
	}
	tile[0] = matteEdge("▄", "")
	tile[len(tile)-1] = matteEdge("▀", caption)
	available := len(tile)
	if caption != "" {
		available--
	}
	// Each rendered line already has its horizontal offset. Fill the unused
	// cells without changing the cover's colors or stretching the image.
	lines = lines[:min(len(lines), available)]
	top := (available - len(lines)) / 2
	for i, line := range lines {
		line = ansi.Truncate(line, artPanelWidth, "…")
		content := strings.TrimLeft(line, " ")
		tile[top+i] = mat.Render(line[:len(line)-len(content)]) + content +
			mat.Render(strings.Repeat(" ", artPanelWidth-ansi.StringWidth(line)))
	}
	return tile
}

func (m *model) joinArtwork(rows []string) {
	if !m.showArtworkPanel() {
		return
	}
	panel := m.openPanel(artPanelWidth, len(rows), m.artworkRows())
	leftWidth := m.bodyWidth()
	for i, line := range rows {
		left := ansi.Truncate(line, leftWidth, "…")
		rows[i] = left + strings.Repeat(" ", max(0, leftWidth-ansi.StringWidth(left))) + "   " + panel[i]
	}
}

func (m *model) pixelWordmark() []string {
	// Draw RÄMPD in a 3×5 pixel font, with two extra rows for the umlaut.
	const pixels = "    # #            " +
		"                   " +
		"##   #  # # ##  ## " +
		"# # # # ### # # # #" +
		"##  ### # # ##  # #" +
		"# # # # # # #   # #" +
		"# # # # # # #   ## " +
		"                   "
	const width = 19
	blocks := [4]rune{' ', '▀', '▄', '█'}
	left := strings.Repeat(" ", (artPanelWidth-width)/2)
	lines := make([]string, 0, len(pixels)/(2*width))
	for y := 0; y < len(pixels); y += 2 * width {
		var line strings.Builder
		for x := 0; x < width; x++ {
			mask := 0
			if pixels[y+x] == '#' {
				mask |= 1
			}
			if pixels[y+width+x] == '#' {
				mask |= 2
			}
			line.WriteRune(blocks[mask])
		}
		style := m.styles.Motif[min(2, len(lines))].Inherit(m.styles.ArtworkMatte)
		lines = append(lines, left+style.Render(line.String()))
	}
	return lines
}
