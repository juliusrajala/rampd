#!/usr/bin/env bash
set -euo pipefail

fail() { printf 'rampd installer: %s\n' "$*" >&2; exit 1; }
usage() {
    printf '%s\n' 'Usage: bash install.sh [--with-mpd MUSIC_DIRECTORY]' \
        'Without arguments, install only rampd into ~/.local/bin.' \
        'With --with-mpd, also prepare a separate local MPD configuration.'
}

music_dir=''
case ${1:-} in
    '') [[ $# -eq 0 ]] || fail 'unexpected argument' ;;
    --help|-h) usage; exit 0 ;;
    --with-mpd)
        [[ $# -eq 2 && -d $2 ]] || fail '--with-mpd needs an existing music directory'
        music_dir=$(cd -P -- "$2" && pwd)
        ;;
    *) usage >&2; exit 1 ;;
esac

case $(uname -s) in
    Linux) os=linux ;;
    Darwin) os=darwin ;;
    *) fail 'only Linux and macOS are supported' ;;
esac
case $(uname -m) in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) fail 'only x86-64 and ARM64 are supported' ;;
esac

base=https://github.com/juliusrajala/r-mpd/releases/latest/download
archive=rampd-$os-$arch.tar.gz
work=$(mktemp -d)
staged=''
cleanup() {
    if [[ -n $staged ]]; then rm -f -- "$staged"; fi
    rm -rf -- "$work"
}
trap cleanup EXIT
fetch() { curl --fail --silent --show-error --location --output "$work/$1" "$base/$1"; }
fetch SHA256SUMS
fetch "$archive"
expected=$(awk -v name="$archive" '$2 == name { print $1 }' "$work/SHA256SUMS")
[[ $expected =~ ^[0-9a-f]{64}$ ]] || fail "no unique checksum found for $archive"
if command -v sha256sum >/dev/null; then
    digest=$(sha256sum < "$work/$archive")
else
    command -v shasum >/dev/null || fail 'sha256sum or shasum is required'
    digest=$(shasum -a 256 < "$work/$archive")
fi
[[ ${digest%% *} == "$expected" ]] || fail "checksum did not match for $archive"
tar -xOzf "$work/$archive" rampd > "$work/rampd"

bin_dir=$HOME/.local/bin
mkdir -p -- "$bin_dir"
[[ ! -d $bin_dir/rampd ]] || fail "$bin_dir/rampd is a directory"
staged=$(mktemp "$bin_dir/.rampd.XXXXXXXX")
install -m 0755 "$work/rampd" "$staged"
mv -f -- "$staged" "$bin_dir/rampd"
staged=''
if [[ ! -e $bin_dir/rämpd && ! -L $bin_dir/rämpd ]]; then
    ln -s rampd "$bin_dir/rämpd"
fi
printf 'Installed %s\n' "$bin_dir/rampd"
case :$PATH: in
    *:"$bin_dir":*) ;;
    *) printf 'Add %s to PATH to use the rampd command.\n' "$bin_dir" ;;
esac

[[ -n $music_dir ]] || exit 0
state=${XDG_STATE_HOME:-$HOME/.local/state}
[[ $state == /* ]] || state=$HOME/.local/state
state=$state/rampd/mpd
socket=$state/socket
config=$state/mpd.conf
for path in "$music_dir" "$state"; do
    case $path in
        *'"'*|*'\'*|*$'\n'*|*$'\r'*) fail 'MPD paths must not contain quotes, backslashes, or line breaks' ;;
    esac
done
if (( $(LC_ALL=C printf '%s' "$socket" | wc -c) > 103 )); then
    fail 'MPD socket path is too long; use a shorter absolute XDG_STATE_HOME'
fi

if ! command -v mpd >/dev/null || ! command -v mpc >/dev/null; then
    privilege=()
    if [[ $os == linux && $(id -u) != 0 ]]; then privilege=(sudo); fi
    if [[ $os == darwin ]] && command -v brew >/dev/null; then
        brew install mpd mpc
    elif [[ $os == linux ]] && command -v pacman >/dev/null; then
        "${privilege[@]}" pacman -S --needed mpd mpc
    elif [[ $os == linux ]] && command -v apt-get >/dev/null; then
        "${privilege[@]}" apt-get install mpd mpc
    elif [[ $os == linux ]] && command -v dnf >/dev/null; then
        "${privilege[@]}" dnf install mpd mpc
    else
        fail 'install mpd and mpc for your system, then run this command again'
    fi
fi

umask 077
mkdir -p -- "$state/playlists"
if [[ -e $config || -L $config ]]; then
    printf 'Kept existing MPD configuration: %s\n' "$config"
else
    (set -o noclobber; printf '%s\n' \
        "music_directory \"$music_dir\"" \
        "playlist_directory \"$state/playlists\"" \
        "db_file \"$state/database\"" \
        "state_file \"$state/state\"" \
        "sticker_file \"$state/stickers.db\"" \
        "log_file \"$state/mpd.log\"" \
        "pid_file \"$state/mpd.pid\"" \
        "bind_to_address \"$socket\"" \
        'restore_paused "yes"' > "$config")
    printf 'Created separate MPD configuration: %s\n' "$config"
fi
printf '\nStart this MPD instance, index your music, then open rampd:\n'
printf '  mpd %q\n  mpc -h %q update\n  MPD_PASSWORD= MPD_PORT=6600 %q -host %q\n' \
    "$config" "$socket" "$bin_dir/rampd" "$socket"
printf '\nDo not run the mpd command again while this instance is running.\n'
