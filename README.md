# rämpd player

A keyboard-first terminal music player for [Music Player Daemon](https://www.musicpd.org/). MPD plays your music and owns the queue. rämpd player does not start MPD or modify music files.

## Run

Start and index your library with MPD before opening the player. Use the Go version in `go.mod` or later.

```sh
go run ./cmd/rampd
```

The default address is `127.0.0.1:6600`. To choose another server or a Unix socket:

```sh
go run ./cmd/rampd -host jukebox.local -port 6600
go run ./cmd/rampd -host /path/to/mpd/socket
```

You can also set `MPD_HOST` and `MPD_PORT`. Use `MPD_PASSWORD` if needed. The player also accepts the standard `password@host` form of `MPD_HOST`. MPD's TCP protocol is not encrypted. Use a trusted network or an SSH tunnel for a remote server.

## Configuration

The optional settings file is `$XDG_CONFIG_HOME/rampd/config.toml`, or `~/.config/rampd/config.toml` if `XDG_CONFIG_HOME` is unset. Use `-config PATH` to read a different file. Command-line flags override environment variables, which override the file.

```toml
[mpd]
host = "127.0.0.1"
port = 6600
timeout = "5s"

[ui]
artwork = true

[dream]
track_count = 25

[keybindings]
play_pause = ["ctrl+p"]
filter = ["/"]
```

Key names are case-sensitive. Use names such as `space`, `enter`, `up`, or `ctrl+r` for special keys. The first key in a list appears in hints. Use an empty list to unbind an action. See [default actions](internal/keymap/keymap.go) for their configuration names. Text prompts keep their standard Enter and Esc controls. Ctrl+C always quits.

Use `MPD_PASSWORD` or set `mpd.password` in a private config file if your server requires a password.

## Keys

| Key | Action |
| --- | --- |
| `1`–`7`, `Tab` | Switch between Artists, Albums, Songs, Tray, Queue, Playlists, and Dream |
| `j` / `k`, arrows | Move the selection |
| `h` / `l` | Go back or open the selection |
| `Enter` | Open an artist or album, or play a track |
| `Space` | Append the selected artist, album, or track |
| `/` | Filter the current list as you type. Press `Enter` to return to the list or `Esc` to clear the filter |
| `Ctrl+R` | Reload the indexed library or current playlist view |
| `p`, `n`, `b` | Play/pause, next, and previous |
| `[` / `]` | Seek backward or forward five seconds |
| `r` / `R` | Toggle MPD random or repeat |
| `x`, `J` / `K` | Remove the queue entry or move it down/up |
| `s`, `X` | Shuffle or clear the queue. Clear asks for confirmation |
| `o` | In Albums or Songs, switch between sorting by title and by artist |
| `:` | Open the command palette. Type to filter, use ↑/↓ to select, then press Enter |
| `?`, `q` | Show key help or quit |
| `w` | Save the current queue or Dream preview as a new playlist |
| `a` | Append all tracks from an open stored playlist to the active queue |
| `A` | Add the selection to an existing stored playlist by name |
| `c`, `e`, `X` | Create, rename, or delete a playlist from the Playlists list |
| `J` / `K`, `x` | Move or remove a track inside an open playlist |

Song rows show track lengths, and album rows show total lengths when MPD knows every track length. Artists show album and song counts. Narrow layouts may hide some columns.

The filter changes only what you see. Queue and Dream actions that apply to the whole list still use the full list. Move commands use positions in the full queue or stored playlist.

### Tray

Press `t` while browsing to collect the selected song, album, or artist. This skips songs already in the Tray. The Tray lasts until you quit and does not change playback while you edit it. If you previously bound `t` to another action, change that binding or assign a different key to `add_to_tray`.

- Press `4` to open the Tray. Use `J` / `K` to move a track and `x` to remove it.
- Press `C` in Queue or Tray to copy the live queue into the Tray. This keeps duplicate entries and asks before replacing a nonempty Tray.
- Press `Space` in the Tray to append the full Tray to Queue, or press `w` to save it as a playlist.
- Press `P` in the Tray to replace the live queue after confirmation. If the current queue entry stays in the Tray, it keeps its playback position. Otherwise the player stops it and does not start another track. The player rejects a replacement if the queue or current track changes before it checks the confirmed edit.
- Press `D` in the Tray to preview a Dream based on related artists or genres. The preview leaves the Tray and Queue unchanged.

### Dream

Press `7`, then `g` to generate a preview. Press `g` again to generate another sequence. Previewing makes no changes to MPD. Use `-dream-count` to set the target length (default: 25).

- Press `Enter` to append and play the sequence.
- Press `Space` to append it without starting playback.
- Press `P` to replace the queue and start playback. This asks for confirmation.
- Press `w` to save the preview as a native MPD playlist without changing the active queue. Open it with `6`.

Playing a library or stored playlist track appends it and starts playback. It does not replace the queue. Playback, queue, library, and stored playlist changes from other MPD clients appear automatically when the server supports MPD notifications. If notifications are unavailable, the player polls for playback and queue changes. Press `Ctrl+R` to refresh the library or playlists manually.

Wide terminals put navigation on the left and show details and square album covers for the highlighted item on the right when there is room. The Selection panel puts the cover above the title and its metadata. The Songs list keeps album names out of the rows but still accepts them in the filter. A pixel rämpd logo takes the place of missing covers. Narrow terminals keep the navigation above the list. The playback panel shows the track MPD is playing. Use `-artwork=false` to hide the selection panel.

## Checks

```sh
go test ./...
go vet ./...
go test -race ./...
```

On Linux with `mpd` and `ffmpeg` installed, run the integration test:

```sh
RAMPD_MPD_INTEGRATION=1 go test -count=1 ./internal/mpd
```

The test creates a temporary library and uses a private Unix socket and a null audio output. It does not use your normal MPD server.

For a keyboard smoke test, install `tmux`, `python3`, `mpd`, and `ffmpeg`, then run `bash scripts/smoke.sh`. See the [release checklist](docs/release-checklist.md) for build and packaging checks.
