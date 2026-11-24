package app

import (
	"cmp"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"rampd/internal/keymap"
	"rampd/internal/library"
	"rampd/internal/music"
)

const (
	artistsTab = iota
	albumsTab
	songsTab
	trayTab
	queueTab
	playlistsTab
	dreamTab
)

var tabNames = [...]string{"Artists", "Albums", "Songs", "Tray", "Queue", "Playlists", "Dream"}
var tabActions = [len(tabNames)]keymap.Action{
	keymap.Artists, keymap.Albums, keymap.Songs, keymap.Tray, keymap.Queue, keymap.Playlists, keymap.Dream,
}

type row struct {
	key, label, detail      string
	matchLabel, matchDetail string
	artURI                  string
	tracks                  []music.Track
	children                []row
	id, position            int
	duration                time.Duration
}

func newRow(key, label, detail string) row {
	return row{
		key: key, label: label, detail: detail,
		matchLabel: strings.ToLower(label), matchDetail: strings.ToLower(detail),
	}
}

type page struct {
	title     string
	parentKey string
	playlist  string
	tracks    []music.Track
	allRows   []row
	rows      []row
	filter    string
	cursor    int
	top       int
}

func (p *page) move(delta int) {
	p.cursor = max(0, min(len(p.rows)-1, p.cursor+delta))
}

func (p *page) setRows(rows []row) {
	p.allRows = rows
	p.updateRows()
}

func (p *page) setFilter(query string) {
	if p.filter != query {
		p.filter = query
		p.updateRows()
	}
}

func (p *page) updateRows() {
	rows := filterRows(p.allRows, p.filter)
	if p.cursor < len(p.rows) {
		key := p.rows[p.cursor].key
		for i, r := range rows {
			if r.key == key {
				p.cursor = i
				break
			}
		}
	}
	p.rows = rows
	p.move(0)
}

func (m *model) page() *page {
	if m.prompt == palettePrompt {
		return &m.palettePage
	}
	stack := m.tabs[m.tab]
	return &stack[len(stack)-1]
}

func (m *model) selected() (row, bool) {
	p := m.page()
	if p.cursor >= len(p.rows) {
		return row{}, false
	}
	return p.rows[p.cursor], true
}

func (m *model) onAlbumList() bool {
	return m.tab == albumsTab && len(m.tabs[albumsTab]) == 1 && m.prompt != palettePrompt
}

func (m *model) onMediaList() bool {
	return m.onAlbumList() || m.tab == songsTab && m.prompt != palettePrompt
}

func (m *model) mediaOrderLabel() string {
	if m.sortByArtist[m.tab] {
		return "artist"
	}
	return "title"
}

func (m *model) nextMediaSort() string {
	if m.sortByArtist[m.tab] {
		return "Sort by title"
	}
	return "Sort by artist"
}

func (m *model) toggleMediaSort() {
	m.sortByArtist[m.tab] = !m.sortByArtist[m.tab]
	root := &m.tabs[m.tab][0]
	root.setRows(orderMedia(root.allRows, m.sortByArtist[m.tab]))
}

func orderMedia(rows []row, byArtist bool) []row {
	// The prepared rows can also be used by other views. Do not reorder them.
	rows = slices.Clone(rows)
	if byArtist {
		artists := make(map[string]string)
		for _, r := range rows {
			if _, ok := artists[r.detail]; !ok {
				artists[r.detail] = strings.ToLower(r.detail)
			}
		}
		// The source is in title order. Keep that order within each artist.
		slices.SortStableFunc(rows, func(a, b row) int {
			return cmp.Or(cmp.Compare(artists[a.detail], artists[b.detail]), cmp.Compare(a.detail, b.detail))
		})
	} else {
		sortRows(rows)
	}
	return rows
}

func (m *model) buildQueuePage() {
	rows := make([]row, 0, len(m.queue))
	for i, entry := range m.queue {
		r := newRow(strconv.Itoa(entry.ID), trackTitle(entry.Track), trackDetail(entry.Track))
		r.id, r.position, r.artURI, r.duration = entry.ID, i, entry.URI, entry.Duration
		rows = append(rows, r)
	}
	m.tabs[queueTab][0].setRows(rows)
}

func prepareLibrary(tracks []music.Track, err error) libraryMsg {
	msg := libraryMsg{tracks: tracks, err: err}
	if err == nil {
		msg.artists, msg.albums = browseRows(library.Build(tracks))
		msg.songs = trackRows(tracks)
		for i := range msg.songs {
			// matchDetail still includes the album for searching.
			msg.songs[i].detail = trackArtist(tracks[i])
		}
		sortRows(msg.songs)
	}
	return msg
}

func (m *model) setLibraryRoot(tab int, rows []row) {
	// Rebuild open pages from the refreshed data. Keep a page only if its parent still exists.
	stack := m.tabs[tab]
	for i := range stack {
		stack[i].setRows(rows)
		if i+1 >= len(stack) {
			break
		}
		// The child belongs to its parent row, even when a filter hides that row.
		key := stack[i+1].parentKey
		parent := slices.IndexFunc(rows, func(r row) bool { return r.key == key })
		if parent < 0 || rows[parent].children == nil {
			stack = stack[:i+1]
			break
		}
		rows = rows[parent].children
	}
	m.tabs[tab] = stack
}

func browseRows(index library.Index) (artists, albums []row) {
	albums = make([]row, 0, len(index.Albums))
	for _, album := range index.Albums {
		r := newRow(album.Artist+"\x00"+album.Name, album.Name, album.Artist)
		r.tracks, r.children = album.Tracks, trackRows(album.Tracks)
		r.duration = albumDuration(album.Tracks)
		albums = append(albums, r)
	}
	artists = make([]row, 0, len(index.Artists))
	for _, artist := range index.Artists {
		r := newRow(artist.Name, artist.Name, "")
		r.children = make([]row, 0, len(artist.Albums))
		var count int
		for _, i := range artist.Albums {
			r.children = append(r.children, albums[i])
			count += len(albums[i].tracks)
		}
		r.tracks = make([]music.Track, 0, count)
		for _, album := range r.children {
			r.tracks = append(r.tracks, album.tracks...)
		}
		artists = append(artists, r)
	}
	return artists, albums
}

func albumDuration(tracks []music.Track) time.Duration {
	var total time.Duration
	for _, track := range tracks {
		if track.Duration <= 0 {
			return 0 // Do not show a partial total as the length of the album.
		}
		total += track.Duration
	}
	return total
}

func trackRows(tracks []music.Track) []row {
	rows := make([]row, 0, len(tracks))
	for i, t := range tracks {
		r := newRow(t.URI, trackTitle(t), trackDetail(t))
		r.tracks, r.duration = tracks[i:i+1], t.Duration
		rows = append(rows, r)
	}
	return rows
}

func trackTitle(t music.Track) string {
	if t.Title != "" {
		return t.Title
	}
	return path.Base(t.URI)
}

func trackArtist(t music.Track) string {
	if t.Artist != "" {
		return t.Artist
	}
	return t.AlbumArtist
}

func trackDetail(t music.Track) string {
	detail := trackArtist(t)
	if t.Album == "" {
		return detail
	}
	if detail == "" {
		return t.Album
	}
	return detail + " · " + t.Album
}

func sortRows(rows []row) {
	slices.SortFunc(rows, func(a, b row) int {
		return cmp.Or(
			cmp.Compare(a.matchLabel, b.matchLabel),
			cmp.Compare(a.matchDetail, b.matchDetail),
			cmp.Compare(a.key, b.key),
		)
	})
}
