package app

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strconv"

	tea "charm.land/bubbletea/v2"

	"rampd/internal/keymap"
	"rampd/internal/music"
)

type trayEntry struct {
	key int
	// ID is the source MPD queue ID, or -1 for a library pick.
	music.Entry
}

type trayCopyMsg struct {
	queue    []music.Entry
	revision uint64
	err      error
}

func (m *model) hiddenTrayMarks() int {
	p := m.browsePage()
	if p.filter == "" {
		return 0
	}
	hidden := len(m.trayMarks)
	for _, row := range p.rows {
		if m.trayMarks[m.tray[row.position].key] {
			hidden--
		}
	}
	return hidden
}

func (m *model) trayTracks() []music.Track {
	tracks := make([]music.Track, 0, len(m.tray))
	for _, entry := range m.tray {
		tracks = append(tracks, entry.Track)
	}
	return tracks
}

func (m *model) addTraySelection() {
	item, ok := m.selected()
	if !ok || m.tab == trayTab {
		return
	}
	tracks := m.selectionTracks(item)
	if len(tracks) == 0 {
		return
	}
	seen := make(map[string]bool, len(m.tray)+len(tracks))
	for _, entry := range m.tray {
		seen[entry.URI] = true
	}
	var added int
	for _, track := range tracks {
		if track.URI == "" || seen[track.URI] {
			continue
		}
		seen[track.URI] = true
		m.addTrayEntry(music.Entry{ID: -1, Track: track})
		added++
	}
	if added == 0 {
		m.notice = "Selection already in Tray."
		return
	}
	m.notice = fmt.Sprintf("Added to Tray: %d · Total: %d", added, len(m.tray))
	m.trayChanged()
}

func (m *model) addTrayEntry(entry music.Entry) {
	m.tray = append(m.tray, trayEntry{key: m.trayNextKey, Entry: entry})
	m.trayNextKey++
}

func (m *model) updateTrayRows() {
	m.trayRevision++
	rows := make([]row, 0, len(m.tray))
	for i, entry := range m.tray {
		r := newRow(strconv.Itoa(entry.key), trackTitle(entry.Track), trackDetail(entry.Track))
		r.tracks = []music.Track{entry.Track}
		r.position, r.duration = i, entry.Duration
		rows = append(rows, r)
	}
	m.tabs[trayTab][0].setRows(rows)
}

func (m *model) markTray(item row) {
	key := m.tray[item.position].key
	if m.trayMarks[key] {
		delete(m.trayMarks, key)
		return
	}
	if m.trayMarks == nil {
		m.trayMarks = make(map[int]bool)
	}
	m.trayMarks[key] = true
}

func (m *model) editTray(action keymap.Action) {
	item, ok := m.selected()
	picked := m.trayMarks
	if len(picked) == 0 {
		if !ok {
			return
		}
		picked = map[int]bool{m.tray[item.position].key: true}
	}

	before := len(m.tray)
	switch action {
	case keymap.Remove:
		m.tray = slices.DeleteFunc(m.tray, func(entry trayEntry) bool { return picked[entry.key] })
		clear(m.trayMarks)
		m.notice = fmt.Sprintf("Removed from Tray: %d", before-len(m.tray))
	case keymap.MoveUp:
		// Walk in the direction of the move to shift each marked run once.
		moveTrayEntries(m.tray, picked, -1)
	case keymap.MoveDown:
		moveTrayEntries(m.tray, picked, 1)
	}
	m.trayChanged()
}

func (m *model) dedupeTray() {
	before := len(m.tray)
	seen := make(map[string]bool, before)
	m.tray = slices.DeleteFunc(m.tray, func(entry trayEntry) bool {
		if seen[entry.URI] {
			delete(m.trayMarks, entry.key)
			return true
		}
		seen[entry.URI] = true
		return false
	})
	removed := before - len(m.tray)
	if removed == 0 {
		m.notice = "No duplicates in the Tray."
		return
	}
	m.notice = fmt.Sprintf("Duplicates removed from Tray: %d", removed)
	m.trayChanged()
}

func (m *model) shuffleTray() {
	if len(m.tray) < 2 {
		return
	}
	rand.Shuffle(len(m.tray), func(i, j int) {
		m.tray[i], m.tray[j] = m.tray[j], m.tray[i]
	})
	m.notice = "Shuffled Tray."
	m.trayChanged()
}

func (m *model) trayChanged() {
	m.updateTrayRows()
	if m.saveTray != nil {
		// Finish the small local write before accepting another edit or quit.
		m.traySaveErr = m.saveTray(m.trayTracks())
		if m.traySaveErr != nil {
			m.notice = fmt.Sprintf("Tray not saved: %v", m.traySaveErr)
		}
	}
}

func moveTrayEntries(entries []trayEntry, picked map[int]bool, delta int) {
	for step := 1; step < len(entries); step++ {
		from := step
		if delta > 0 {
			from = len(entries) - 1 - step
		}
		to := from + delta
		if picked[entries[from].key] && !picked[entries[to].key] {
			entries[from], entries[to] = entries[to], entries[from]
		}
	}
}

func (m *model) copyQueueToTray() tea.Cmd {
	if m.trayCopying {
		return nil
	}
	if len(m.tray) > 0 {
		m.confirmation = &confirmation{"Replace the Tray with a copy of the live queue?", m.fetchTrayCopy}
		return nil
	}
	return m.fetchTrayCopy()
}

func (m *model) fetchTrayCopy() tea.Cmd {
	if !m.connected {
		m.notice = "MPD is not connected."
		return nil
	}
	m.trayCopying = true
	ctx, server, revision := m.ctx, m.server, m.trayRevision
	return func() tea.Msg {
		snapshot, err := server.Snapshot(ctx, "")
		return trayCopyMsg{snapshot.Queue, revision, err}
	}
}

func (m *model) applyTrayCopy(msg trayCopyMsg) {
	m.trayCopying = false
	if msg.err != nil {
		m.notice = fmt.Sprintf("Copy queue: %v", msg.err)
		return
	}
	if msg.revision != m.trayRevision {
		m.notice = "Tray changed while copying. Copy the queue again to replace it."
		return
	}
	m.tray = nil
	clear(m.trayMarks)
	for _, entry := range msg.queue {
		m.addTrayEntry(entry)
	}
	m.tabs[trayTab][0].setFilter("")
	if (m.tab == queueTab || m.tab == trayTab) && m.prompt == noPrompt && m.confirmation == nil && m.help == noHelp {
		m.tab = trayTab
	}
	m.notice = fmt.Sprintf("Copied queue to Tray: %d tracks.", len(msg.queue))
	m.trayChanged()
}

func (m *model) confirmTrayReplacement() {
	if !m.connected || m.busy {
		m.notice = "Wait for MPD to finish updating. Then try again."
		return
	}
	plan := music.QueueReplacement{
		Version:   m.status.QueueVersion,
		CurrentID: m.status.PlayingID(),
		Before:    m.queue,
		KeepAt:    -1,
	}
	if plan.CurrentID >= 0 {
		for _, entry := range m.queue {
			if entry.ID != plan.CurrentID {
				continue
			}
			// Prefer the original occurrence in a copied queue. A newly
			// chosen track with the same URI can also keep playback going.
			for i, next := range m.tray {
				if next.URI != entry.URI {
					continue
				}
				if next.ID == entry.ID {
					plan.KeepAt = i
					break
				}
				if next.ID < 0 && plan.KeepAt < 0 {
					plan.KeepAt = i
				}
			}
			break
		}
	}
	message := "Replace the live queue with the Tray?"
	if plan.CurrentID >= 0 && plan.KeepAt < 0 {
		message = "Replace queue? Playback will stop."
	}
	m.confirmAction(message, music.Action{Operation: music.ReplaceQueue, Tracks: m.trayTracks(), Replacement: &plan})
}
