#!/bin/bash
set -euo pipefail

# Installs the cross-compiled app, downloaded bw binary, and selected Pi config.
if [[ "$EUID" -ne 0 ]]; then
  echo "Run 03-install-app.sh as root." >&2
  exit 1
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
for required_file in homelab-wallet bw config.json; do
  if [[ ! -f "$script_dir/$required_file" ]]; then
    echo "Required deployment file is missing: $required_file" >&2
    exit 1
  fi
done

install -d -o bitwarden -g bitwarden -m 0750 /var/lib/bitwarden
install -o bitwarden -g bitwarden -m 0755 "$script_dir/homelab-wallet" /var/lib/bitwarden/homelab-wallet
install -o bitwarden -g bitwarden -m 0755 "$script_dir/bw" /var/lib/bitwarden/bw
install -o bitwarden -g bitwarden -m 0600 "$script_dir/config.json" /var/lib/bitwarden/config.json
echo "Installed homelab-wallet, bw, and config.json in /var/lib/bitwarden."
