package app

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"rampd/internal/artwork"
	"rampd/internal/music"
)

type playlistSummary struct {
	genres, artists []string
	count, timed    int
	duration        time.Duration
}

type playlistSummariesMsg struct {
	seq   uint64
	items map[string]*playlistSummary
}

func (m *model) fetchPlaylistSummaries(names []string, seq uint64) tea.Cmd {
	ctx, server := m.ctx, m.server
	return func() tea.Msg {
		items := make(map[string]*playlistSummary, len(names))
		for _, name := range names {
			if ctx.Err() != nil {
				break
			}
			if tracks, err := server.Playlist(ctx, name); err == nil {
				summary := summarizePlaylist(tracks)
				items[name] = &summary
			}
		}
		return playlistSummariesMsg{seq, items}
	}
}

func (m *model) updatePlaylistSummaries(msg playlistSummariesMsg) {
	if msg.seq != m.playlistsSeq {
		return
	}
	p := &m.tabs[playlistsTab][0]
	rows := slices.Clone(p.allRows)
	for i := range rows {
		rows[i].playlist = msg.items[rows[i].key]
	}
	p.setRows(rows)
}

func summarizePlaylist(tracks []music.Track) playlistSummary {
	type genre struct {
		label string
		count int
	}
	genres := make(map[string]genre)
	artists := make(map[string]string)
	summary := playlistSummary{count: len(tracks)}
	for _, track := range tracks {
		if track.Duration > 0 {
			summary.duration += track.Duration
			summary.timed++
		}
		if label := strings.TrimSpace(track.Genre); label != "" {
			key := strings.ToLower(label)
			g := genres[key]
			if g.count == 0 {
				g.label = label
			}
			g.count++
			genres[key] = g
		}
		if label := strings.TrimSpace(trackArtist(track)); label != "" {
			key := strings.ToLower(label)
			if _, ok := artists[key]; !ok {
				artists[key] = label
			}
		}
	}
	keys := make([]string, 0, len(genres))
	for key := range genres {
		keys = append(keys, key)
	}
	slices.SortFunc(keys, func(a, b string) int {
		return cmp.Or(cmp.Compare(genres[b].count, genres[a].count), cmp.Compare(a, b))
	})
	for _, key := range keys[:min(3, len(keys))] {
		summary.genres = append(summary.genres, genres[key].label)
	}
	keys = keys[:0]
	for key := range artists {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		summary.artists = append(summary.artists, artists[key])
	}
	return summary
}

func (m *model) playlistOverviewRow(title string, summary *playlistSummary, width int, selected bool) string {
	if width < 32 {
		return ansi.Truncate(title, width, "…")
	}
	const countWidth, lengthWidth, gap = 9, 8, 1
	genres, count, length := "—", "—", "—"
	if summary != nil {
		if len(summary.genres) > 0 {
			genres = clean(strings.Join(summary.genres, ", "))
		}
		count = fmt.Sprintf("%d tracks", summary.count)
		if len(count) > countWidth {
			count = fmt.Sprint(summary.count)
		}
		switch {
		case summary.count == summary.timed:
			length = clock(summary.duration)
		case summary.timed > 0:
			length = "≥" + clock(summary.duration)
		}
	}
	nameWidth := width - countWidth - lengthWidth - 2*gap
	genreCell := ""
	if width >= 44 {
		genreWidth := min(nameWidth-13-gap, width/3)
		nameWidth -= genreWidth + gap
		genreCell = playlistCell(genres, genreWidth, false) + strings.Repeat(" ", gap)
	}
	totals := playlistCell(count, countWidth, true) + strings.Repeat(" ", gap) + playlistCell(length, lengthWidth, true)
	if !selected {
		genreCell = m.styles.Muted.Render(genreCell)
		totals = m.styles.ProgressFilled.Render(totals)
	}
	return playlistCell(title, nameWidth, false) + strings.Repeat(" ", gap) + genreCell + totals
}

func playlistCell(text string, width int, right bool) string {
	text = ansi.Truncate(text, width, "…")
	padding := strings.Repeat(" ", width-ansi.StringWidth(text))
	if right {
		return padding + text
	}
	return text + padding
}

func (m *model) playlistPreview(item row) []string {
	room := m.bodyHeight() - 1 // The panel has one row of top padding.
	if m.framed() {
		room = m.height - 1
	}
	var artists []string
	if item.playlist != nil {
		artists = item.playlist.artists
	}
	var lines []string
	// Keep room for the name and up to five artist rows before drawing a cover.
	if room >= artRowCount+2+max(1, min(5, len(artists))) {
		lines = append(m.artworkTile(artwork.SynthwaveCover(item.key, artRowCount), ""), "")
	}
	lines = append(lines, m.styles.SongName.Bold(true).Render(clean(item.label)))
	room -= len(lines)
	if item.playlist == nil {
		return append(lines, m.styles.Muted.Render("Details unavailable"))
	}
	if len(artists) == 0 {
		return append(lines, m.styles.Muted.Render("No artist tags"))
	}
	more := len(artists) > room
	if more {
		room-- // Reserve a row for the number of artists that are not shown.
	}
	visible := min(len(artists), max(0, room))
	for _, artist := range artists[:visible] {
		lines = append(lines, m.styles.ArtistName.Render(clean(artist)))
	}
	if more {
		lines = append(lines, m.styles.Muted.Render(fmt.Sprintf("+%d more", len(artists)-visible)))
	}
	return lines
}
