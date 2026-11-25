package app

import (
	"charm.land/lipgloss/v2"

	"rampd/internal/music"
)

func (m *model) canSwitchTrackColumn() bool {
	switch m.tab {
	case trayTab, dreamTab, queueTab:
		return true
	case playlistsTab:
		return m.browsePage().playlist != ""
	default:
		return false
	}
}

func (m *model) trackColumnName() string {
	if m.showAlbumColumn[m.tab] {
		return "Album"
	}
	return "Artist"
}

func (m *model) trackColumnCommand() string {
	if m.showAlbumColumn[m.tab] {
		return "Show artist"
	}
	return "Show album"
}

func (m *model) rowTrack(item row) (music.Track, bool) {
	if m.tab == queueTab && item.position < len(m.queue) && m.queue[item.position].ID == item.id {
		return m.queue[item.position].Track, true
	}
	if item.children == nil && len(item.tracks) == 1 {
		return item.tracks[0], true
	}
	return music.Track{}, false
}

func (m *model) rowNameStyle(item row) lipgloss.Style {
	switch {
	case item.children != nil && item.detail == "":
		return m.styles.ArtistName
	case item.children != nil:
		return m.styles.AlbumName
	default:
		if _, ok := m.rowTrack(item); ok {
			return m.styles.SongName
		}
		return m.styles.Muted
	}
}

func (m *model) mediaRowDetail(item row, selected bool) string {
	paint := func(style lipgloss.Style, name string) string {
		name = clean(name)
		if selected {
			return name
		}
		return style.Render(name)
	}
	artist := func(name string) string { return paint(m.styles.ArtistName, name) }
	album := func(name string) string { return paint(m.styles.AlbumName, name) }
	if item.children != nil {
		return artist(item.detail)
	}
	track, ok := m.rowTrack(item)
	if !ok {
		return clean(item.detail)
	}
	if m.canSwitchTrackColumn() && m.showAlbumColumn[m.tab] {
		return album(track.Album)
	}
	name := trackArtist(track)
	if m.tab == songsTab || m.canSwitchTrackColumn() || track.Album == "" {
		return artist(name)
	}
	if name == "" {
		return album(track.Album)
	}
	return artist(name) + " · " + album(track.Album)
}
