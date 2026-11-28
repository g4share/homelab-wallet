#!/bin/bash
set -euo pipefail

# Installs the unit, orders Docker after it, and enables/starts the wallet service.
if [[ "$EUID" -ne 0 ]]; then
  echo "Run 05-install-service.sh as root." >&2
  exit 1
fi

script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
for required_file in homelab-wallet.service docker.service.conf; do
  if [[ ! -f "$script_dir/$required_file" ]]; then
    echo "$required_file is missing from the deployment staging directory." >&2
    exit 1
  fi
done

install -o root -g root -m 0644 "$script_dir/homelab-wallet.service" \
  /etc/systemd/system/homelab-wallet.service

docker_dropin_dir=/etc/systemd/system/docker.service.d
install -d -o root -g root -m 0755 "$docker_dropin_dir"
install -o root -g root -m 0644 "$script_dir/docker.service.conf" \
  "$docker_dropin_dir/homelab-wallet.conf"

systemctl daemon-reload
systemctl enable --now homelab-wallet.service
echo "Installed and started homelab-wallet.service. Docker will start after it on future starts."
