#!/bin/bash
set -euo pipefail

# Downloads and extracts the Bitwarden CLI asset selected by the deployment profile.
usage() {
  exit_code="${1:-2}"
  echo "Usage: $0 -a <asset-name>" >&2
  echo "Example: $0 -a bw-bit-linux-arm64-2026.11.0" >&2
  exit "$exit_code"
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

asset="${asset%.zip}"
version="${asset##*-}"
archive_name="${asset}.zip"
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
archive_path="$script_dir/$archive_name"
download_url="https://github.com/bitwarden/clients/releases/download/cli-v${version}/${archive_name}"

echo "Downloading ${asset}..."
curl -fL "$download_url" -o "$archive_path"
unzip -o "$archive_path" -d "$script_dir"
rm -f -- "$archive_path"
chmod +x "$script_dir/bw"
echo "Installed the Bitwarden CLI in the deployment staging directory."
