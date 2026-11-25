#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 1 ]]; then
    printf 'Usage: bash scripts/package-release.sh [OUTPUT_DIRECTORY]\n' >&2
    exit 1
fi
root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
output=${1:-$root/dist/release}
mkdir -p -- "$output"
output=$(cd -- "$output" && pwd)
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
mkdir "$work/assets" "$work/archive"
cd "$root"

for os in linux darwin; do
    for arch in amd64 arm64; do
        CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -o "$work/archive/rampd" ./cmd/rampd
        tar -czf "$work/assets/rampd-$os-$arch.tar.gz" -C "$work/archive" rampd -C "$root" LICENSE README.md
    done
done
cp install.sh "$work/assets/install.sh"
(
    cd "$work/assets"
    if command -v sha256sum >/dev/null; then
        sha256sum rampd-*.tar.gz install.sh > SHA256SUMS
    else
        shasum -a 256 rampd-*.tar.gz install.sh > SHA256SUMS
    fi
)
cp "$work/assets/"* "$output/"
printf 'Release files: %s\n' "$output"
