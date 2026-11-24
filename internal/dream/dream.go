// Package dream generates listening sessions without changing playback.
package dream

import (
	"context"
	"errors"
	"math/rand/v2"
	"strings"

	"rampd/internal/music"
)

type Options struct {
	TrackCount int
	Seeds      []music.Track
}

type Strategy interface {
	Name() string
	Generate(context.Context, []music.Track, Options) ([]music.Track, error)
}

// Baseline selects unique tracks and prefers a different artist each time.
type Baseline struct{}

func (Baseline) Name() string { return "baseline" }

func (Baseline) Generate(ctx context.Context, library []music.Track, options Options) ([]music.Track, error) {
	if options.TrackCount <= 0 {
		return nil, errors.New("Dream track count must be positive")
	}

	candidates := make([]music.Track, 0, len(library))
	seedArtists, seedGenres := make(map[string]bool), make(map[string]bool)
	seen := make(map[string]bool, len(library))
	for _, seed := range options.Seeds {
		seen[seed.URI] = true
		for _, value := range []string{seed.Artist, seed.AlbumArtist} {
			if name := normalize(value); name != "" {
				seedArtists[name] = true
			}
		}
		if genre := normalize(seed.Genre); genre != "" {
			seedGenres[genre] = true
		}
	}
	for _, track := range library {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if strings.TrimSpace(track.URI) == "" || seen[track.URI] {
			continue
		}
		if len(options.Seeds) > 0 && !seedArtists[normalize(track.Artist)] &&
			!seedArtists[normalize(track.AlbumArtist)] && !seedGenres[normalize(track.Genre)] {
			continue
		}
		seen[track.URI] = true
		candidates = append(candidates, track)
	}
	if len(candidates) == 0 {
		if len(options.Seeds) > 0 {
			return nil, errors.New("no related tracks found outside the Tray")
		}
		return nil, errors.New("Dream needs at least one indexed track")
	}
	rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })

	// Group shuffled tracks by artist. Use the first group on a tie, so equal
	// groups start in a random order. Use the largest eligible group first to
	// avoid stranding the most common artist at the end of the session.
	var groups [][]music.Track
	index := make(map[string]int)
	for _, track := range candidates {
		key := normalize(track.Artist)
		if key == "" {
			key = normalize(track.AlbumArtist)
		}
		i, ok := index[key]
		if !ok {
			i = len(groups)
			groups = append(groups, nil)
			index[key] = i
		}
		groups[i] = append(groups[i], track)
	}

	limit := min(options.TrackCount, len(candidates))
	result := make([]music.Track, 0, limit)
	previous := -1
	for len(result) < limit {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		next := -1
		for i, group := range groups {
			if i != previous && len(group) > 0 && (next < 0 || len(group) > len(groups[next])) {
				next = i
			}
		}
		if next < 0 {
			next = previous
		}
		result = append(result, groups[next][0])
		groups[next] = groups[next][1:]
		previous = next
	}
	return result, nil
}

func normalize(tag string) string {
	return strings.ToLower(strings.TrimSpace(tag))
}
