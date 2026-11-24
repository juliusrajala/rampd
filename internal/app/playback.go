package app

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func (m *model) playbackLine() string {
	if !m.connected {
		return m.styles.AccentText.Render("Not connected")
	}
	status := m.status
	mode := "Stopped"
	switch status.State {
	case "play":
		mode = "Playing"
	case "pause":
		mode = "Paused"
	}
	mode = m.styles.AccentText.Render(mode)
	var artist, suffix string
	if status.State == "play" || status.State == "pause" {
		for _, entry := range m.queue {
			if entry.ID == status.SongID {
				mode += " · " + clean(trackTitle(entry.Track))
				name := entry.Artist
				if name == "" {
					name = entry.AlbumArtist
				}
				if name != "" {
					artist = " — " + clean(name)
				}
				break
			}
		}
		if status.Duration > 0 {
			suffix += " · " + clock(m.playbackElapsed()) + " / " + clock(status.Duration)
		}
	}
	if status.Random {
		suffix += " · Random"
	}
	if status.Repeat {
		suffix += " · Repeat"
	}
	if ansi.StringWidth(mode+artist+suffix) <= m.innerWidth(m.width) {
		mode += artist
	}
	return mode + suffix
}

func durationLabel(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	return clock(d)
}

func clock(d time.Duration) string {
	seconds := int(d / time.Second)
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, seconds/60%60, seconds%60)
	}
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func (m *model) playbackWithProgress() string {
	text := m.playbackLine()
	width := m.innerWidth(m.width)
	elapsed, total := m.playbackElapsed(), m.status.Duration
	hasProgress := m.connected && total > 0 && (m.status.State == "play" || m.status.State == "pause")
	// Do not hide playback text to fit an inline bar.
	cells := width - ansi.StringWidth(text) - 2
	if hasProgress && cells >= 8 {
		text += "  " + m.progressBar(cells, elapsed, total)
	}
	return text
}

func (m *model) progressBar(cells int, elapsed, total time.Duration) string {
	elapsed = max(0, min(elapsed, total))
	filled := int(float64(cells) * (float64(elapsed) / float64(total)))
	played, unplayed := m.styles.ProgressFilled, m.styles.ProgressEmpty
	if cells < 2 {
		return played.Render(strings.Repeat("█", filled)) +
			unplayed.Render(strings.Repeat(" ", cells-filled))
	}

	// Half-block end caps expose the background at opposite corners.
	emptyCap := unplayed.UnsetBackground().Foreground(unplayed.GetBackground())
	first, last := emptyCap, emptyCap
	if filled > 0 {
		first = played.UnsetBackground()
	}
	if filled == cells {
		last = played.UnsetBackground()
	}
	middle := min(cells-2, max(0, filled-1))
	return first.Render("▄") + played.Render(strings.Repeat("█", middle)) +
		unplayed.Render(strings.Repeat(" ", cells-2-middle)) + last.Render("▀")
}
