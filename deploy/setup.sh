#!/bin/bash
set -euo pipefail

# Runs the target installation steps in order. Invoke this script as root.
if [[ "$EUID" -ne 0 ]]; then
  echo "Run setup.sh with sudo." >&2
  exit 1
fi
if [[ $# -ne 1 || -z "$1" ]]; then
  echo "Usage: setup.sh <Bitwarden CLI asset-name>" >&2
  exit 2
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
bw_asset="$1"

echo "Step 1/5: create the restricted bitwarden service account."
bash "$script_dir/01-create-user.sh"

echo "Step 2/5: download the native Bitwarden CLI on this target."
bash "$script_dir/02-install-bw.sh" -a "$bw_asset"

echo "Step 3/5: install the Go app and target config into the service home."
bash "$script_dir/03-install-app.sh"

echo "Step 4/5: prompt for credentials here and encrypt them with systemd."
bash "$script_dir/04-encrypt-credentials.sh"

echo "Step 5/5: install and enable the systemd service."
bash "$script_dir/05-install-service.sh"

echo "Setup finished. Check the service with: systemctl status homelab-wallet.service"
