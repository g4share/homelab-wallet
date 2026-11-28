#!/bin/bash
set -euo pipefail

usage() {
  echo "Usage: $0 <account-name> <uid-gid>" >&2
  echo "Example: $0 dock-beszel-agent 5000" >&2
  exit 2
}

if [[ "$EUID" -ne 0 ]]; then
  echo "Run this script with sudo on the target host." >&2
  exit 1
fi

if [[ $# -ne 2 ]]; then
  usage
fi

account_name="$1"
account_id="$2"
user_shell=/usr/sbin/nologin

if [[ ! "$account_name" =~ ^[a-z_][a-z0-9_-]*$ ]]; then
  echo "Account name must start with a lowercase letter or underscore and contain only lowercase letters, digits, underscores, or hyphens." >&2
  exit 2
fi

if [[ ! "$account_id" =~ ^[0-9]+$ ]]; then
  echo "UID/GID must be a decimal number." >&2
  exit 2
fi

account_id_number=$((10#$account_id))
if (( account_id_number < 5000 || account_id_number > 5999 )); then
  echo "UID/GID must be in the reserved container account range 5000-5999." >&2
  exit 2
fi
account_id="$account_id_number"

if getent group "$account_name" >/dev/null; then
  existing_group_id="$(getent group "$account_name" | cut -d: -f3)"
  if [[ "$existing_group_id" != "$account_id" ]]; then
    echo "Group $account_name already exists with GID $existing_group_id, expected $account_id." >&2
    exit 1
  fi
elif getent group "$account_id" >/dev/null; then
  echo "GID $account_id is already assigned to another group." >&2
  exit 1
else
  groupadd --gid "$account_id" "$account_name"
fi

if getent passwd "$account_name" >/dev/null; then
  existing_user_id="$(id -u "$account_name")"
  existing_group_id="$(id -g "$account_name")"
  existing_home="$(getent passwd "$account_name" | cut -d: -f6)"
  existing_shell="$(getent passwd "$account_name" | cut -d: -f7)"
  if [[ "$existing_user_id" != "$account_id" || "$existing_group_id" != "$account_id" || "$existing_home" != /nonexistent || "$existing_shell" != "$user_shell" ]]; then
    echo "User $account_name exists but does not match the expected UID/GID, home, and shell." >&2
    exit 1
  fi
elif getent passwd "$account_id" >/dev/null; then
  echo "UID $account_id is already assigned to another user." >&2
  exit 1
else
  useradd \
    --uid "$account_id" \
    --gid "$account_name" \
    --home-dir /nonexistent \
    --no-create-home \
    --shell "$user_shell" \
    --password '!' \
    "$account_name"
fi

passwd --lock "$account_name" >/dev/null
echo "Host account ready: $account_name (UID:GID $account_id:$account_id, no login shell)."
