// Package library groups indexed music for browsing.
package library

import (
	"cmp"
	"slices"
	"strings"

	"rampd/internal/music"
)

// Album groups tracks with the same album and album artist.
type Album struct {
	Artist string
	Name   string
	Tracks []music.Track
}

// Artist identifies an artist and its albums by index in Index.Albums.
type Artist struct {
	Name   string
	Albums []int
}

// Index holds the sorted album and artist views of a music collection.
type Index struct {
	Albums  []Album
	Artists []Artist
}

// Build groups and sorts tracks without modifying the supplied library.
func Build(tracks []music.Track) Index {
	type albumKey struct{ artist, title string }
	groups := make(map[albumKey][]music.Track)
	for _, t := range tracks {
		title := t.Album
		if title == "" {
			title = "Unknown album"
		}
		key := albumKey{albumArtist(t), title}
		groups[key] = append(groups[key], t)
	}

	type sortableAlbum struct {
		Album
		titleFold, artistFold string
	}
	sorted := make([]sortableAlbum, 0, len(groups))
	for key, songs := range groups {
		slices.SortFunc(songs, func(a, b music.Track) int {
			return cmp.Or(cmp.Compare(a.Disc, b.Disc), cmp.Compare(a.Number, b.Number), cmp.Compare(a.URI, b.URI))
		})
		sorted = append(sorted, sortableAlbum{
			Album:     Album{Artist: key.artist, Name: key.title, Tracks: songs},
			titleFold: strings.ToLower(key.title), artistFold: strings.ToLower(key.artist),
		})
	}
	slices.SortFunc(sorted, func(a, b sortableAlbum) int {
		return cmp.Or(
			cmp.Compare(a.titleFold, b.titleFold), cmp.Compare(a.artistFold, b.artistFold),
			cmp.Compare(a.Artist, b.Artist), cmp.Compare(a.Name, b.Name),
		)
	})

	index := Index{Albums: make([]Album, len(sorted))}
	byArtist := make(map[string][]int)
	for i, album := range sorted {
		index.Albums[i] = album.Album
		byArtist[album.Artist] = append(byArtist[album.Artist], i)
	}
	index.Artists = make([]Artist, 0, len(byArtist))
	for name, albums := range byArtist {
		index.Artists = append(index.Artists, Artist{Name: name, Albums: albums})
	}
	slices.SortFunc(index.Artists, func(a, b Artist) int {
		return cmp.Or(cmp.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)), cmp.Compare(a.Name, b.Name))
	})
	return index
}

func albumArtist(t music.Track) string {
	if t.AlbumArtist != "" {
		return t.AlbumArtist
	}
	if t.Artist != "" {
		return t.Artist
	}
	return "Unknown artist"
}
