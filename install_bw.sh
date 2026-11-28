#!/bin/bash
set -e

usage() {
  echo "Usage: $0 -a <asset-name>" >&2
  echo "Example: $0 -a bw-bit-macos-arm64-2026.11.0" >&2
  exit "${1:-2}"
}

asset=""
while getopts ":a:h" opt; do
  case "$opt" in
    a) asset="$OPTARG" ;;
    h) usage 0 ;;
    :) usage 2 ;;
    \?) usage 2 ;;
  esac
done
shift $((OPTIND - 1))
if [[ -z "$asset" || $# -ne 0 ]]; then
  usage 2
fi

# Accept the asset name with or without the .zip suffix.
asset="${asset%.zip}"
version="${asset##*-}"

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
archive_name="${asset}.zip"
url="https://github.com/bitwarden/clients/releases/download/cli-v${version}/${archive_name}"
archive="$script_dir/$archive_name"

# unzip -o replaces an existing bw executable in the project directory.
curl -fL "$url" -o "$archive"
unzip -o "$archive" -d "$script_dir"
rm -f "$archive"
chmod +x "$script_dir/bw"
echo "Set bitwarden.cli_path to: $script_dir/bw"
