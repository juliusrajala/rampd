#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 || ! -d $1 ]]; then
    printf 'Usage: bash scripts/test-install.sh RELEASE_FILES_DIRECTORY\n' >&2
    exit 1
fi
export RELEASE_FIXTURE=$(cd -- "$1" && pwd)
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
work=$(mktemp -d)
server=''
cleanup() {
    if [[ -n $server ]]; then kill "$server" 2>/dev/null || true; wait "$server" 2>/dev/null || true; fi
    rm -rf -- "$work"
}
trap cleanup EXIT
if [[ ${RAMPD_INSTALL_MPD_TEST:-} == 1 ]]; then
    real_mpd=$(command -v mpd)
    real_mpc=$(command -v mpc)
fi
mkdir "$work/tools" "$work/home" "$work/music library"
export PATH="$work/tools:$PATH"
export HOME="$work/home"
export XDG_STATE_HOME="$work/state"
export XDG_CONFIG_HOME="$work/config"

# Keep downloads, package managers, and MPD isolated from the host.
printf '%s\n' '#!/usr/bin/env bash' 'set -euo pipefail' \
    'while [[ $1 != --output ]]; do shift; done' \
    '[[ $3 == https://github.com/juliusrajala/r-mpd/releases/latest/download/* ]]' \
    'cp "$RELEASE_FIXTURE/${3##*/}" "$2"' \
    'if [[ ${CORRUPT_ARCHIVE:-} == yes && $3 == *.tar.gz ]]; then printf x >> "$2"; fi' > "$work/tools/curl"
for name in mpd mpc sudo brew pacman apt-get dnf; do
    printf '#!/usr/bin/env bash\nexit 99\n' > "$work/tools/$name"
done
chmod +x "$work/tools/"*

# Verify one archive for each platform without running foreign binaries.
for variant in linux-amd64 linux-arm64 darwin-amd64 darwin-arm64; do
    case $variant in linux-*) os=Linux ;; *) os=Darwin ;; esac
    case $variant in *-amd64) arch=x86_64 ;; *) arch=arm64 ;; esac
    printf '#!/usr/bin/env bash\ncase "$1" in -s) echo %s ;; -m) echo %s ;; *) exit 1 ;; esac\n' "$os" "$arch" > "$work/tools/uname"
    chmod +x "$work/tools/uname"
    bash "$root/install.sh"
    tar -xOzf "$RELEASE_FIXTURE/rampd-$variant.tar.gz" rampd > "$work/expected"
    cmp "$work/expected" "$HOME/.local/bin/rampd"
    cmp "$work/expected" "$HOME/.local/bin/rämpd"
    [[ ! -e $XDG_STATE_HOME && ! -e $XDG_CONFIG_HOME ]]
done
if CORRUPT_ARCHIVE=yes bash "$root/install.sh" > "$work/bad.log" 2>&1; then
    printf 'installer accepted a corrupted archive\n' >&2
    exit 1
fi
grep -q 'checksum did not match' "$work/bad.log"
cmp "$work/expected" "$HOME/.local/bin/rampd"

bash "$root/install.sh" --with-mpd "$work/music library" > "$work/setup.log"
config=$XDG_STATE_HOME/rampd/mpd/mpd.conf
grep -Fx "music_directory \"$work/music library\"" "$config"
grep -Fx "bind_to_address \"$XDG_STATE_HOME/rampd/mpd/socket\"" "$config"
grep -Fx "sticker_file \"$XDG_STATE_HOME/rampd/mpd/stickers.db\"" "$config"
[[ ! -e $XDG_CONFIG_HOME ]]
printf '# Local setting\n' >> "$config"
cp "$config" "$work/expected-conf"
bash "$root/install.sh" --with-mpd "$work/music library" > "$work/reinstall.log"
cmp "$work/expected-conf" "$config"
grep -q 'Kept existing MPD configuration' "$work/reinstall.log"
if [[ ${RAMPD_INSTALL_MPD_TEST:-} == 1 ]]; then
    socket=$XDG_STATE_HOME/rampd/mpd/socket
    "$real_mpd" --no-daemon --stderr "$config" > "$work/mpd.log" 2>&1 &
    server=$!
    for i in {1..40}; do
        [[ -S $socket ]] && break
        kill -0 "$server" 2>/dev/null || break
        sleep .1
    done
    if ! [[ -S $socket && -f $XDG_STATE_HOME/rampd/mpd/stickers.db ]]; then
        printf 'The generated MPD instance did not start.\n' >&2
        head -60 "$work/mpd.log" >&2
        exit 1
    fi
    "$real_mpc" -h "$socket" --wait update >/dev/null
fi
printf 'Installer checks passed for all four release archives.\n'
