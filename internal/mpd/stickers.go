package mpd

import (
	"context"
	"strconv"
	"strings"
	"time"

	"rampd/internal/music"
)

const (
	lastPlayedSticker = "rampd.last_played"
	playCountPrefix   = "rampd.playcount."
)

func (c Client) PlayHistory(ctx context.Context, uri string) (music.PlayHistory, error) {
	w, err := c.open(ctx)
	if err != nil {
		return music.PlayHistory{}, err
	}
	defer w.Close()
	fields, err := w.query("sticker", "list", "song", uri)
	if err != nil {
		return music.PlayHistory{}, err
	}
	return parsePlayHistory(fields), nil
}

func parsePlayHistory(fields []field) music.PlayHistory {
	var history music.PlayHistory
	for _, f := range fields {
		if f.key != "sticker" {
			continue
		}
		name, value, ok := strings.Cut(f.value, "=")
		if !ok {
			continue
		}
		if name == lastPlayedSticker {
			if at, err := time.Parse(time.RFC3339, value); err == nil {
				history.LastPlayed = at.UTC()
			}
		} else if year, ok := strings.CutPrefix(name, playCountPrefix); ok {
			y, yearErr := strconv.Atoi(year)
			n, countErr := strconv.ParseUint(value, 10, 64)
			if yearErr == nil && len(year) == 4 && y > 0 && countErr == nil {
				if history.Yearly == nil {
					history.Yearly = make(map[int]uint64)
				}
				history.Yearly[y] = n
			}
		}
	}
	return history
}

// RecordPlay stores the qualification time and increments its UTC year count.
func (c Client) RecordPlay(ctx context.Context, uri string, at time.Time) error {
	w, err := c.open(ctx)
	if err != nil {
		return err
	}
	defer w.Close()

	at = at.UTC()
	return w.runList([]command{
		{"sticker", []string{"inc", "song", uri, playCountPrefix + strconv.Itoa(at.Year()), "1"}},
		{"sticker", []string{"set", "song", uri, lastPlayedSticker, at.Format(time.RFC3339)}},
	})
}
