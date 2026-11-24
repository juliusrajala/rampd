#!/usr/bin/env bash
set -euo pipefail
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
dir=$(mktemp -d)
server=''
tm() { tmux -S "$dir/tmux" -f /dev/null "$@"; }
cleanup() {
  tm kill-server 2>/dev/null || true
  if [[ -n $server ]]; then kill "$server" 2>/dev/null || true; wait "$server" 2>/dev/null || true; fi
  rm -rf -- "$dir"
}
trap cleanup EXIT

if [[ $# -gt 1 ]]; then
  printf 'Usage: %s [rampd-binary]\n' "$0" >&2
  exit 2
fi
if [[ $# -eq 1 ]]; then
  binary=$(realpath -- "$1")
else
  binary=$dir/rampd
  (cd -- "$root" && go build -o "$binary" ./cmd/rampd)
fi
# Ignore user settings and MPD environment variables during the test.
touch "$dir/rampd.toml"

mkdir "$dir/music" "$dir/playlists"
ffmpeg -v error -f lavfi -i sine=frequency=440:duration=30 -metadata 'artist=Pizza Artist' -metadata 'album=First Album' -metadata 'title=Smoke Track' "$dir/music/smoke.flac"
ffmpeg -v error -f lavfi -i sine=frequency=880:duration=30 -metadata 'artist=Pizza Artist' -metadata 'album=First Album' -metadata 'title=Zeta Track' "$dir/music/zeta.flac"
ffmpeg -v error -f lavfi -i sine=frequency=660:duration=30 -metadata 'artist=Pizza Artist' -metadata 'album=First Album' -metadata 'title=ZZ Related' "$dir/music/zz-related.flac"
ffmpeg -v error -f lavfi -i color=red:s=96x64 -frames:v 1 -threads 1 "$dir/music/cover.png"
printf 'music_directory "%s/music"\nplaylist_directory "%s/playlists"\ndb_file "%s/db"\nbind_to_address "%s/mpd"\naudio_output {\n type "null"\n name "Test"\n}\n' "$dir" "$dir" "$dir" "$dir" > "$dir/mpd.conf"
start_app() {
  tm new-session -d -s app -x 100 -y 24 -e TERM=xterm-256color \
    env -u MPD_HOST -u MPD_PORT -u MPD_PASSWORD XDG_STATE_HOME="$dir/state" \
    "$binary" -config "$dir/rampd.toml" -host "$dir/mpd"
}
start_app
expect_text() {
  for i in {1..60}; do
    if tm capture-pane -pt app | grep -Fq -- "$1"; then return; fi
    sleep .1
  done
  tm capture-pane -pt app
  printf 'Missing: %s\n' "$1" >&2
  exit 1
}
expect_absent() {
  for i in {1..60}; do
    if ! tm capture-pane -pt app | grep -Fq -- "$1"; then return; fi
    sleep .1
  done
  tm capture-pane -pt app
  printf 'Still visible: %s\n' "$1" >&2
  exit 1
}
expect_text 'MPD: offline'
expect_text '╭ Navigation'
tm capture-pane -pt app | grep -E '^╭ Navigation.*╮  ╭ Songs.*╮  ╭ SELECTION' >/dev/null
expect_text '▸ [1] Songs'
expect_text '[6] Playlists'
expect_text 'SESSION'
expect_text 'MIXES'
expect_text '╭ Playback ─'
mpd --no-daemon --stderr "$dir/mpd.conf" > "$dir/log" 2>&1 &
server=$!
for i in {1..30}; do
  [[ -S $dir/mpd ]] && break
  sleep .1
done
python - "$dir/mpd" <<'PY'
import socket, sys
with socket.socket(socket.AF_UNIX) as conn:
    conn.connect(sys.argv[1])
    f = conn.makefile('rwb')
    assert f.readline().startswith(b'OK MPD ')
    f.write(b'update\n')
    f.flush()
    assert f.readline().startswith(b'updating_db: ')
    assert f.readline() == b'OK\n'
PY
expect_text 'MPD: online'
expect_text 'Pizza Artist'
tm capture-pane -pt app | grep -B1 '^╭ Playback' | grep -Eq 'MPD: online *$'
tm send-keys -t app 3 Enter Enter Space j Space 5
expect_text '›   Smoke Track'
expect_text 'Zeta Track'
expect_text 'Queue · Artist'
tm send-keys -t app o
expect_text 'Queue · Album'
tm send-keys -t app o Enter
expect_text 'Playing · Smoke Track'
tm capture-pane -pt app | grep -E '^│ Playing · Smoke Track.*▄[█ ]{28,}▀ │$' >/dev/null
expect_text '│ ▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀▀ │'
expect_text '╯  ╰'
tm send-keys -t app j
expect_text '│ Zeta Track'
expect_text 'Playing · Smoke Track'
tm send-keys -t app :
expect_text '╭ Commands'
tm send-keys -lt app pause
expect_text 'Play or pause'
tm send-keys -t app Enter
expect_text 'Paused · Smoke Track'
tm send-keys -t app /
tm send-keys -lt app 'pizza smoke'
expect_text '[ / pizza smoke'
expect_text '› ♫ Smoke Track'
tm send-keys -t app Enter
expect_text '[ / pizza smoke ]'
tm resize-window -t app -x 38 -y 12
expect_text 'Tab: Queue'
expect_text '[ / pizza smoke ]'
expect_text '› ♫ Smoke Tr'
tm send-keys -t app Escape
expect_absent '[ / pizza smoke ]'
expect_text '› ♫ Smoke Tr'
tm resize-window -t app -x 100 -y 24
tm send-keys -t app 1 t
expect_text 'Added to Tray: 1'
tm send-keys -t app 5 C
expect_text 'Replace the Tray with a copy'
tm send-keys -t app y
expect_text 'Copied queue to Tray: 2 tracks.'
expect_text 'Edit tracks'
tm send-keys -t app J C-t
expect_text 'Whole Tray'
expect_absent 'Edit tracks'
tm send-keys -t app P
expect_text 'Replace the live queue with the Tray?'
tm send-keys -t app y
expect_text 'Replaced live queue with the Tray.'
expect_text 'Paused · Smoke Track'
tm send-keys -t app w
expect_text 'Save as:'
tm send-keys -lt app 'Tray Mix'
tm send-keys -t app Enter
expect_text 'Saved playlist "Tray Mix".'
tm send-keys -t app D
expect_text 'Dream / From Tray'
expect_text 'Dream preview · Tracks: 1'
expect_text 'ZZ Related'
tm send-keys -t app g
expect_text 'Dream preview · Tracks: 3'
expect_text 'Paused · Smoke Track'
tm send-keys -t app w
expect_text 'Save as:'
tm send-keys -lt app 'My Dream'
tm send-keys -t app Enter
expect_text 'Saved playlist "My Dream"'
tm send-keys -t app 6
expect_text 'My Dream'
expect_text 'Tray Mix'
tm send-keys -t app Home Enter
expect_text 'Playlists / My Dream'
expect_text 'Smoke Track'
tm send-keys -t app h c
expect_text 'Create:'
tm send-keys -lt app 'Road Mix'
tm send-keys -t app Enter
expect_text 'Created playlist "Road Mix".'
tm send-keys -t app 5 A
expect_text 'Add to:'
tm send-keys -lt app 'Road Mix'
tm send-keys -t app Enter
expect_text 'Added selection to "Road Mix".'
tm send-keys -t app 6
expect_text '    Road Mix'
tm send-keys -t app Home j e
expect_text 'Rename to:'
tm send-keys -t app C-u
tm send-keys -lt app 'Night Mix'
tm send-keys -t app Enter
expect_text 'Renamed playlist to "Night Mix".'
expect_text '›   Night Mix'
tm send-keys -t app X
expect_text 'Delete playlist Night Mix?'
tm send-keys -t app y
expect_text 'Deleted playlist "Night Mix".'
kill "$server"
wait "$server" 2>/dev/null || true
server=''
expect_text 'MPD: offline'
mpd --no-daemon --stderr "$dir/mpd.conf" > "$dir/log" 2>&1 &
server=$!
expect_text 'MPD: online'
tm send-keys -t app 5
expect_text 'Queue is empty.'
tm send-keys -t app q
for i in {1..60}; do
  if ! tm has-session -t app 2>/dev/null; then break; fi
  sleep .1
done
start_app
expect_text 'MPD: online'
tm send-keys -t app 4
expect_text 'Edit tracks'
expect_text '›   Zeta Track'
expect_text 'Smoke Track'
tm send-keys -t app 5
expect_text 'Queue is empty.'
tm send-keys -t app q
printf 'Terminal smoke passed: startup and reconnect, grouped browsing, playback, filtering, Tray, Dream and playlist editing.\n'
