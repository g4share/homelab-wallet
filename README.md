# Homelab Wallet

## Contents

- [About](#about)
- [Development](#development)
- [Deploy](#deploy-with-ansible)

## About

Homelab Wallet is a Go command-line application that retrieves selected values from Bitwarden Password Manager Secure Notes and writes them to files or standard output.

It is intended for services that need secrets at runtime. Store the source values in Bitwarden, describe which values to select in the app configuration, and let Homelab Wallet materialize them where the service can read them instead of hard-coding them in application or deployment files.

### Where secrets are stored

The Bitwarden Password Manager vault is the source of truth: the original secrets stay in your online vault, and Homelab Wallet does not delete or change them. The app reads the selected values when it runs and writes copies to the configured output destinations. Those copies are local to the machine running the app; use an appropriate destination, such as a RAM-backed directory, if they should not persist on disk. Vault access remains protected by your Bitwarden account and credentials.

### How it works

1. The app invokes the [Bitwarden CLI (`bw`)](https://bitwarden.com/help/cli/) to sync and read the vault.
2. It finds the Secure Note named in the configuration and parses the JSON in that note's **Notes** field.
3. It applies the configured JSONPath query. A query must return exactly one value.
4. It writes the result to a file or to standard output.

Custom-field references in resolved JSON values use `${FIELD_NAME}`. Use `$${...}` when the literal text `${...}` should be preserved.

## Development

### Prerequisites

- Go, using the version declared in [`go.mod`](go.mod)
- A Bitwarden Password Manager account
- Bitwarden's standalone native CLI executable, `bw`

For local development on an Apple Silicon Mac, install a matching CLI release asset with:

```sh
./install_bw.sh -a bw-bit-macos-arm64-2026.11.1
```

The asset name is `bw-bit-macos-arm64-2026.11.1`: `bit` identifies the licensed Bitwarden CLI build, `macos` is the platform, `arm64` is the Apple Silicon architecture, and `2026.11.1` is the release version. To install another release, use the matching asset name from the [Bitwarden CLI releases](https://github.com/bitwarden/clients/releases). The script installs `bw` in the project directory and replaces an existing `bw`; the app uses it by default when `cli_path` is empty.

### Configure Bitwarden

Create a Bitwarden Secure Note and put the JSON payload in its **Notes** text field. The format inside this field is defined by the application. For example:

```json
{
  "name": "beszel-agent",
  "items": [
    {
      "type": "BESZEL_AGENT_KEY",
      "value": "${BESZEL_AGENT_KEY}"
    },
    {
      "schema": "rpi4",
      "type": "BESZEL_AGENT_TOKEN",
      "value": "${BESZEL_AGENT_TOKEN-RPI4}"
    },
    {
      "schema": "rpi5",
      "type": "BESZEL_AGENT_TOKEN",
      "value": "${BESZEL_AGENT_TOKEN-RPI5}"
    }
  ]
}
```

Add Bitwarden custom fields to the same Secure Note, using the placeholder names exactly, and put the actual values in those fields. For sensitive values, choose the **Hidden** field type so Bitwarden masks the value in its interface. Homelab Wallet can still read it through the CLI after the vault is unlocked. See [Bitwarden's custom fields guide](https://bitwarden.com/help/custom-fields/).

![Bitwarden custom fields with the secret values hidden](docs/images/bitwarden-custom-fields.png#gh-light-mode-only)
![Bitwarden custom fields with the secret values hidden](docs/images/bitwarden-custom-fields-dark.png#gh-dark-mode-only)

Custom-field references follow these rules:

- The app searches only custom fields on the same Secure Note.
- Names must match exactly, including letter case.
- Text and Hidden fields resolve to strings.
- Checkbox/Boolean fields resolve to JSON `true` or `false`.
- Linked fields are not supported.
- Escape a literal `${...}` as `$${...}` to prevent reference resolution.
- Use `schema` and `type` in the JSON payload so the configured JSONPath query can select the right secret, such as the `rpi4` or `rpi5` token.
- The JSON `name` labels the payload. The app configuration's `note` must match the Secure Note's Bitwarden title.

### Configure the app locally

By default, the app reads `config.json` from the current directory. Use `-config` to select another file. Set `note` to the exact title of the Secure Note you created above.

This example uses the [Bitwarden EU cloud](https://vault.bitwarden.eu). Set `server_url` to `https://vault.bitwarden.eu` for an EU account. See Bitwarden's [server region guide](https://bitwarden.com/help/choose-my-server/) if your account is in another region.

A configuration has this shape:

```json
{
  "bitwarden": {
    "cli_path": "bw",
    "server_url": "https://vault.bitwarden.eu",
    "client_id": "file:///path/to/client-id",
    "client_secret": "file:///path/to/client-secret",
    "master_password": "file:///path/to/master-password"
  },
  "secrets": [
    {
      "note": "beszel-agent",
      "query": "$.items[?@.type == \"BESZEL_AGENT_TOKEN\" && @.schema == \"rpi4\"].value",
      "output": "file:///run/homelab-wallet/secrets/BESZEL_AGENT_TOKEN"
    }
  ]
}
```

Configuration notes:

- `client_id`, `client_secret`, and `master_password` can be literal strings or `file://` references.
- If the `bw` CLI instance is unauthenticated, the app needs `client_id` and `client_secret` to log in, unless you sign in with `bw login` first.
- If the app runs in a terminal, `master_password` may be empty; the Bitwarden CLI then prompts for it during `bw unlock`. See the [Bitwarden CLI unlock instructions](https://bitwarden.com/help/cli/#unlock).
- A systemd service has no interactive terminal. For production, provide `master_password` through a `file://` reference backed by systemd credentials.
- Each `secrets` entry has a JSONPath query over the Secure Note's **Notes** JSON and an output destination.
- Use `file://<path>` to write a file or `stdout` to print the value. File outputs support `group`, `mode`, and `dir_mode` options. For example:

  ```text
  file:///dev/shm/homelab-wallet/secrets/beszel/BESZEL_AGENT_TOKEN?group=dock-beszel-agent&mode=0240&dir_mode=0710
  ```

  `group` sets the file and directory group; without it, normal filesystem ownership rules apply. `mode` sets the output file permissions and defaults to `0600`. `dir_mode` sets the parent directory permissions and defaults to `0700`. Permission values use octal notation.
- Treat the configuration as sensitive; do not commit real credentials to the repository.

#### Plaintext at runtime

Homelab Wallet reads `client_id`, `client_secret`, and `master_password` from the configured values or `file://` paths as plaintext while it runs. In production, systemd credentials encrypt these Bitwarden credentials at rest and provide decrypted files to the service at runtime through `CREDENTIALS_DIRECTORY`.

Values selected from the Secure Note, such as the Beszel token and key, are written as plaintext to the output paths configured in `secrets`. In production, those paths can be under `/dev/shm`, which is memory-backed. These output files are separate from the encrypted Bitwarden credential files.

### Run

Run from the project directory:

```sh
go run ./cmd/homelab-wallet
```

To select a configuration file or log level:

```sh
go run ./cmd/homelab-wallet -config /path/to/config.json -log-level debug
```

The supported log levels are `debug`, `info`, `warn`, and `error`. The default is `info`; `HOMELAB_WALLET_LOG_LEVEL` can set the default level.

Build a local executable with:

```sh
go build -trimpath -o homelab-wallet ./cmd/homelab-wallet
```

## Deploy with Ansible

The Ansible playbook deploys Homelab Wallet to the Raspberry Pi in the inventory. It:

1. Creates the restricted `bitwarden` group and service account, and sets `/var/lib/bitwarden` ownership and permissions.
2. Downloads the selected native Bitwarden CLI release into the service home, extracts `bw`, sets its ownership and executable permissions, then removes the archive.
3. Builds the Go application on the Mac for Linux ARM64 and installs the binary in `/var/lib/bitwarden`.
4. Installs the inventory host's configuration as `/var/lib/bitwarden/config.json` with restricted permissions.
5. Checks that the Raspberry Pi has systemd 250 or newer and creates `/etc/credstore.encrypted` with root-only access.
   - `systemd-creds` encrypts credentials with a host key and, when available and configured, binds them to the TPM. The current Raspberry Pi setup does not detect a usable TPM.
   - Without TPM binding, the host key is stored on the same system disk as the encrypted credentials. If that disk is not encrypted separately, someone with physical access can obtain both and decrypt the credentials.
   - The `root` user on the running Raspberry Pi can access credentials when systemd supplies them to the service, regardless of TPM. See the [`systemd-creds` documentation](https://www.freedesktop.org/software/systemd/man/latest/systemd-creds.html).
6. Prompts for and encrypts these Bitwarden credentials with `systemd-creds` when their encrypted files are missing:
   - `client_id` → `bw-client-id` — Bitwarden Password Manager API client ID.
   - `client_secret` → `bw-client-secret` — Bitwarden Password Manager API client secret.
   - `master_password` → `bw-master-password` — Bitwarden Password Manager master password.
7. Sets root-only permissions on the encrypted credential files. Existing encrypted credentials are kept unless `force_credentials=true` is set.
8. Installs the [`homelab-wallet.service`](deploy/ansible/roles/homelab_wallet/files/homelab-wallet.service) systemd service file and the Docker ordering drop-in, reloads systemd, enables the Homelab Wallet service, and restarts it.

A systemd unit is a configuration file that tells systemd how to run a service, including its user, credentials, and dependencies.

The relevant part of the deployed configuration looks like this:

```json
{
  "bitwarden": {
    "master_password": "file://${CREDENTIALS_DIRECTORY}/bw-master-password"
  },
  "secrets": [
    {
      "note": "beszel-agent",
      "query": "$.items[?@.type == \\"BESZEL_AGENT_KEY\\"].value",
      "output": "file:///dev/shm/homelab-wallet/secrets/beszel/BESZEL_AGENT_KEY?group=dock-beszel-agent&mode=0240&dir_mode=0710"
    }
  ]
}
```

`CREDENTIALS_DIRECTORY` is an environment variable set by systemd for the service. The linked service file tells systemd to decrypt `bw-master-password` from `/etc/credstore.encrypted`; systemd exposes it as a plaintext credential file under that directory while the service runs. The value is the Bitwarden vault master password, which Homelab Wallet passes to `bw` to unlock the vault.

Homelab Wallet reads `BESZEL_AGENT_KEY` from the selected Bitwarden Secure Note and writes it as plaintext to `/dev/shm/homelab-wallet/secrets/beszel/BESZEL_AGENT_KEY`. Docker Compose mounts it inside the container at `/run/secrets/BESZEL_AGENT_KEY`, where Beszel reads it through `BESZEL_AGENT_KEY_FILE`. Beszel supports this file-based variable; see its [official environment-variable documentation](https://beszel.dev/guide/environment-variables). For other container images, verify that the application supports a file-based variable before using this pattern.

The secret name and its mounted path are chosen in the Compose configuration and must match the path configured in the container's environment. In [`beszel-agent.yml`](deploy/examples/beszel-agent.yml), `/var/lib/beszel-agent` is the persistent data volume; the secret files are mounted separately under `/run/secrets`.

The playbook does not create the container-specific `dock-beszel-agent` account; create it separately as described below.

### Prerequisites

- Run the playbook from the project root on your Mac. Install Ansible, Go, and Git there. Ansible cross-compiles the app for Linux ARM64 locally.
- The Raspberry Pi must be reachable by SSH, have `sudo` access, and run systemd 250 or newer for encrypted credentials.
- Create a dedicated non-login Linux user and group for each container that needs access to a secret share. This is the identity the container process runs as, specified in Docker Compose with `user: "UID:GID"`. The helper reserves IDs `5000`–`5999`, creates a same-named host user and group, and sets the user's shell to `nologin`.

For the Beszel example, create `dock-beszel-agent` with UID:GID `5000:5000` on the Raspberry Pi before running Ansible. From the project root on your Mac, copy and run the helper on the Pi:

```sh
scp deploy/ansible/add-container-user.sh gmunteanu@10.0.0.14:~/add-container-user.sh
ssh -t gmunteanu@10.0.0.14 'sudo bash ~/add-container-user.sh dock-beszel-agent 5000'
```

In this example, Docker runs the Beszel process as UID:GID `5000:5000`. The host account `dock-beszel-agent` has the same IDs. Homelab Wallet runs as `bitwarden` and writes the secret files with group `dock-beszel-agent`; [`homelab-wallet.service`](deploy/ansible/roles/homelab_wallet/files/homelab-wallet.service) adds this group to the Bitwarden process so it can set the file group.

The host-specific [`config-10.0.0.14.json`](deploy/ansible/roles/homelab_wallet/files/config-10.0.0.14.json) sets the output group and permissions. For example:

```json
"output": "file:///dev/shm/homelab-wallet/secrets/beszel/BESZEL_AGENT_KEY?group=dock-beszel-agent&mode=0240&dir_mode=0710"
```

- `group=dock-beszel-agent` sets the group on the parent directory and secret file. Only processes with this group can access the file through its group permissions; unrelated unprivileged users and containers cannot. The host's `root` user remains privileged.
- `mode=0240` gives the `bitwarden` owner write permission and the `dock-beszel-agent` group read permission. Other users get no file permissions.
- `dir_mode=0710` gives the `bitwarden` owner full directory access and the group permission to traverse the directory. Other users get no directory access.

For multiple containers, create a different user and group with a different unused UID:GID for each one. Give each container only its own group and secrets. Add every required group to [`homelab-wallet.service`](deploy/ansible/roles/homelab_wallet/files/homelab-wallet.service)'s `SupplementaryGroups`, and set the matching `group` option and permissions in that host's configuration. This keeps one container's secret files separate from another's.

[`deploy/examples/beszel-agent.yml`](deploy/examples/beszel-agent.yml) is an example Docker Compose YAML file showing the matching container identity and file-backed secrets. Adapt it to your host; the Ansible playbook does not deploy this file automatically.

### Run the playbook

From the project root on your Mac:

```sh
ansible-playbook -i deploy/ansible/inventory.yml --ask-pass --ask-become-pass deploy/ansible/site.yml
```

Ansible asks for the SSH password and the `sudo` password. It prompts for each Bitwarden credential only when the corresponding encrypted credential is missing. Existing encrypted credentials are reused. To replace all three encrypted credentials, add `-e force_credentials=true`:

```sh
ansible-playbook -i deploy/ansible/inventory.yml --ask-pass --ask-become-pass -e force_credentials=true deploy/ansible/site.yml
```

The credential files are stored encrypted under `/etc/credstore.encrypted`. At service start, systemd makes the credentials available to Homelab Wallet as runtime files. Resolved secrets are written under `/dev/shm/homelab-wallet/secrets/`, which is memory-backed, and are mounted by Docker as service-scoped secrets under `/run/secrets`.

The inventory selects the matching host configuration from `deploy/ansible/roles/homelab_wallet/files/config-<ansible_host>.json`. For the current `rpi4` inventory entry, that file is `config-10.0.0.14.json`.

After deployment, check [`homelab-wallet.service`](deploy/ansible/roles/homelab_wallet/files/homelab-wallet.service), its status, and its logs on the Raspberry Pi:

```sh
sudo systemctl status homelab-wallet.service
sudo journalctl -u homelab-wallet.service -b --no-pager
```
