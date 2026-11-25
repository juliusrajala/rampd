package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

type observedPlay struct {
	id       int
	uri      string
	at       time.Time // Last observation while playing. Zero while paused or offline.
	listened time.Duration
	counted  bool
}

type playRecorded struct {
	uri string
	err error
}

// Sample wall time, not the MPD position. Seeking cannot add listening time.
// A pause or a long gap in observations does not contribute an interval.
func (p *observedPlay) sample(id int, uri string, playing bool, duration time.Duration, now time.Time) bool {
	if id < 0 || uri == "" {
		*p = observedPlay{}
		return false
	}
	if p.id != id || p.uri != uri {
		*p = observedPlay{id: id, uri: uri}
	}
	if !playing {
		p.at = time.Time{}
		return false
	}
	if !p.at.IsZero() {
		if delta := now.Sub(p.at); delta > 0 && delta <= 3*time.Second {
			p.listened += delta
		}
	}
	p.at = now
	threshold := 30 * time.Second
	if duration > 0 {
		threshold = min(threshold, duration/2)
	}
	if p.counted || p.listened < threshold {
		return false
	}
	p.counted = true
	return true
}

func (m *model) observePlay(now time.Time) tea.Cmd {
	if !m.connected {
		return nil
	}
	id := m.status.PlayingID()
	track, ok := m.playingSong()
	if id >= 0 && !ok {
		// A snapshot may contain a new playing ID before its queue entry arrives.
		m.play.at = time.Time{}
		return nil
	}
	duration := m.status.Duration
	if duration <= 0 {
		duration = track.Duration
	}
	if !m.play.sample(id, track.URI, m.status.State == "play", duration, now) {
		return nil
	}
	ctx, server, uri := m.ctx, m.server, track.URI
	return func() tea.Msg {
		// Do not retry: MPD may have applied the increment before a lost reply.
		return playRecorded{uri, server.RecordPlay(ctx, uri, now)}
	}
}
