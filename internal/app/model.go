// Package app implements the terminal player.
package app

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"rampd/internal/dream"
	"rampd/internal/keymap"
	"rampd/internal/music"
	"rampd/internal/theme"
)

type backend interface {
	Snapshot(context.Context, string) (music.Snapshot, error)
	Library(context.Context) ([]music.Track, error)
	Playlists(context.Context) ([]string, error)
	Playlist(context.Context, string) ([]music.Track, error)
	Artwork(context.Context, string) ([]byte, error)
	Apply(context.Context, music.Action) error
	Watch(context.Context, func(music.Changes)) error
}

type model struct {
	ctx    context.Context
	server backend
	styles theme.Styles
	keys   keymap.Bindings

	width, height    int
	status           music.Status
	queue            []music.Entry
	connected        bool
	connectionFailed bool
	busy             bool
	pending          []music.Action
	retry            time.Duration
	refreshSeq       uint64
	notice           string

	events       chan watchMsg
	watching     bool
	watchBackoff time.Duration
	needSnapshot bool
	forceQueue   bool
	sampledAt    time.Time
	clockTime    time.Time

	tabs               [len(tabNames)][]page
	tab                int
	sortByArtist       [len(tabNames)]bool
	showAlbumColumn    [len(tabNames)]bool
	library            []music.Track
	libraryLoaded      bool
	libraryLoading     bool
	libraryError       error
	libraryInvalidated bool
	input              textinput.Model
	prompt             promptMode
	palettePage        page
	saveAction         music.Action
	confirmation       *confirmation
	showHelp           bool

	tray          []trayEntry
	trayNextKey   int
	trayMarks     map[int]bool
	trayWholeMode bool
	trayRevision  uint64
	trayCopying   bool
	saveTray      func([]music.Track) error
	traySaveErr   error

	dreamStrategy dream.Strategy
	dreamCount    int
	dreamLoading  bool
	dreamTracks   []music.Track

	playlistsLoading bool
	playlistsSeq     uint64
	playlistLoading  bool
	playlistSeq      uint64

	artworkEnabled  bool
	artworkLoading  string
	artworkCache    []artworkMsg
	artworkRevision uint64
}

type refreshTick uint64

type stateMsg struct {
	snapshot  music.Snapshot
	readError error
	actError  error
	action    *music.Action
}

type promptMode int

const (
	noPrompt promptMode = iota
	searchPrompt
	palettePrompt
	playlistNamePrompt
)

type libraryMsg struct {
	tracks                 []music.Track
	artists, albums, songs []row
	err                    error
}

type Options struct {
	DreamStrategy dream.Strategy
	DreamCount    int
	Artwork       bool
	Keys          keymap.Bindings
	Styles        *theme.Styles
	InitialTray   []music.Track
	SaveTray      func([]music.Track) error
}

type confirmation struct {
	message string
	run     func() tea.Cmd
}

func New(ctx context.Context, server backend, options Options) tea.Model {
	styles := theme.Default()
	if options.Styles != nil {
		styles = *options.Styles
	}
	input := textinput.New()
	input.SetStyles(styles.Input)
	input.SetVirtualCursor(false)
	if options.DreamStrategy == nil {
		options.DreamStrategy = dream.Baseline{}
	}
	if options.DreamCount <= 0 {
		options.DreamCount = 25
	}
	if options.Keys.IsZero() {
		options.Keys = keymap.Default()
	}
	m := &model{
		ctx: ctx, server: server, styles: styles, keys: options.Keys,
		input: input, retry: time.Second, busy: true,
		dreamStrategy: options.DreamStrategy, dreamCount: options.DreamCount,
		artworkEnabled: options.Artwork,
		events:         make(chan watchMsg), watchBackoff: time.Second,
	}
	for i, title := range tabNames {
		m.tabs[i] = []page{{title: title}}
	}
	m.saveTray = options.SaveTray
	for _, track := range options.InitialTray {
		m.addTrayEntry(music.Entry{ID: -1, Track: track})
	}
	m.updateTrayRows()
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(m.stateCommand(nil), m.runWatch(), m.awaitWatch(), nextProgressTick())
}

func (m *model) stateCommand(action *music.Action) tea.Cmd {
	ctx, server := m.ctx, m.server
	version := ""
	if m.connected && !m.forceQueue {
		version = m.status.QueueVersion
	}
	m.forceQueue, m.needSnapshot = false, false
	return func() tea.Msg {
		var actErr error
		if action != nil {
			actErr = server.Apply(ctx, *action)
		}
		snapshot, err := server.Snapshot(ctx, version)
		return stateMsg{snapshot, err, actErr, action}
	}
}

func (m *model) fetchLibrary() tea.Cmd {
	if m.libraryLoading {
		return nil
	}
	m.libraryLoading = true
	m.libraryError = nil
	ctx, server := m.ctx, m.server
	return func() tea.Msg {
		tracks, err := server.Library(ctx)
		return prepareLibrary(tracks, err)
	}
}

func (m *model) act(a music.Action) tea.Cmd {
	if !m.connected {
		m.notice = "MPD is not connected."
		return nil
	}
	if m.busy {
		if len(m.pending) < 16 {
			m.pending = append(m.pending, a)
		}
		return nil
	}
	m.notice = ""
	m.busy = true
	return m.stateCommand(&a)
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmd := m.update(msg)
	m.fitPage()
	return m, tea.Batch(cmd, m.requestArtwork())
}

func (m *model) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeInput()
	case refreshTick:
		if !m.busy && (!m.connected || !m.watching) && uint64(msg) == m.refreshSeq {
			return m.requestSnapshot()
		}
	case progressTick:
		m.clockTime = time.Time(msg)
		return nextProgressTick()
	case watchMsg:
		return m.handleWatch(msg)
	case watchRetry:
		return m.runWatch()
	case stateMsg:
		return m.updateState(msg)
	case artworkMsg:
		m.storeArtwork(msg)
	case libraryMsg:
		return m.updateLibrary(msg)
	case trayCopyMsg:
		m.applyTrayCopy(msg)
	case playlistsMsg:
		m.updatePlaylists(msg)
	case playlistMsg:
		m.updatePlaylist(msg)
	case dreamMsg:
		m.dreamLoading = false
		if msg.err == nil {
			m.dreamTracks = msg.tracks
			p := &m.tabs[dreamTab][0]
			p.title = "Dream"
			if msg.seeded {
				p.title = "Dream / From Tray"
			}
			p.setRows(trackRows(msg.tracks))
			p.cursor, p.top = 0, 0
			m.notice = fmt.Sprintf("Dream preview · Tracks: %d", len(msg.tracks))
		} else {
			m.notice = fmt.Sprintf("Dream: %v", msg.err)
		}
	case tea.KeyPressMsg:
		return m.key(msg)
	}
	if m.prompt != noPrompt {
		return m.updateInput(msg)
	}
	return nil
}

func (m *model) refreshAfter(delay time.Duration) tea.Cmd {
	// An action can finish before the previous timer fires. Ignore the old timer.
	m.refreshSeq++
	seq := refreshTick(m.refreshSeq)
	return tea.Tick(delay, func(time.Time) tea.Msg { return seq })
}

func (m *model) updateState(msg stateMsg) tea.Cmd {
	m.busy = false
	if msg.readError != nil {
		m.connected, m.connectionFailed = false, true
		m.pending = nil
		m.notice = fmt.Sprintf("MPD: %v", msg.readError)
		wait := m.retry
		m.retry = min(15*time.Second, m.retry*2)
		return m.refreshAfter(wait)
	}

	justConnected := !m.connected
	m.connected, m.connectionFailed, m.retry = true, false, time.Second
	m.status = msg.snapshot.Status
	m.sampledAt = time.Now()
	m.clockTime = m.sampledAt
	if msg.snapshot.QueueChanged {
		m.queue = msg.snapshot.Queue
		m.buildQueuePage()
	}
	switch {
	case msg.actError != nil:
		m.notice = fmt.Sprintf("MPD: %v", msg.actError)
	case justConnected:
		m.notice = ""
	case msg.action != nil && msg.action.Operation == music.ReplaceQueue:
		m.notice = "Replaced live queue with the Tray."
	}
	var library, playlists tea.Cmd
	if justConnected && !m.libraryLoaded {
		library = m.fetchLibrary()
	}
	if justConnected {
		m.invalidateArtwork()
		playlists = m.refreshStoredPlaylists()
	}
	if msg.action != nil {
		playlists = tea.Batch(playlists, m.afterPlaylistAction(*msg.action, msg.actError))
	}
	if len(m.pending) > 0 {
		a := m.pending[0]
		m.pending = m.pending[1:]
		return tea.Batch(library, playlists, m.act(a))
	}
	if m.needSnapshot {
		return tea.Batch(library, playlists, m.requestSnapshot())
	}
	if !m.watching {
		return tea.Batch(library, playlists, m.refreshAfter(time.Second))
	}
	return tea.Batch(library, playlists)
}
