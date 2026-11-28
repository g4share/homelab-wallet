#!/bin/bash
set -euo pipefail

# Prompts on the target and stores each value as an encrypted systemd credential.
if [[ "$EUID" -ne 0 ]]; then
  echo "Run 04-encrypt-credentials.sh as root." >&2
  exit 1
fi

if ! command -v systemd-creds >/dev/null 2>&1; then
  echo "systemd-creds is not installed." >&2
  exit 1
fi

systemd_version="$(systemd-creds --version | awk 'NR == 1 {print $2}')"
if [[ ! "$systemd_version" =~ ^[0-9]+$ ]] || (( systemd_version < 250 )); then
  echo "systemd 250 or newer is required for LoadCredentialEncrypted." >&2
  exit 1
fi

credential_dir=/etc/credstore.encrypted
install -d -o root -g root -m 0700 "$credential_dir"
umask 077

encrypt_credential() {
  local name="$1"
  local prompt="$2"
  local value

  printf 'Credential name: %s\n' "$name" >/dev/tty
  value="$(systemd-ask-password -n "$prompt")"
  if [[ -z "$value" ]]; then
    echo "Credential value must not be empty." >&2
    return 1
  fi
  printf '%s' "$value" | systemd-creds encrypt \
    --name="$name" - "$credential_dir/$name"
  unset value
  echo "Encrypted credential saved: $credential_dir/$name"
}

encrypt_credential bw-client-id "Bitwarden client ID"
encrypt_credential bw-client-secret "Bitwarden client secret"
encrypt_credential bw-master-password "Bitwarden master password"
