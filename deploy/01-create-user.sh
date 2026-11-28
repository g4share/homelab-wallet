#!/bin/bash
set -euo pipefail

# Creates the locked-down service account and its persistent home directory.
if [[ "$EUID" -ne 0 ]]; then
  echo "Run 01-create-user.sh as root." >&2
  exit 1
fi

if ! getent group bitwarden >/dev/null; then
  groupadd --system bitwarden
fi

if ! getent passwd bitwarden >/dev/null; then
  useradd --system --gid bitwarden --home-dir /var/lib/bitwarden \
    --create-home --shell /usr/sbin/nologin bitwarden
else
  usermod --gid bitwarden --home /var/lib/bitwarden \
    --shell /usr/sbin/nologin bitwarden
fi

install -d -o bitwarden -g bitwarden -m 0750 /var/lib/bitwarden
echo "Service account ready: bitwarden (no login shell)."
