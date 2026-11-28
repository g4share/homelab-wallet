# Homelab Wallet

Homelab Wallet is a Go command-line application that retrieves selected values from Bitwarden Password Manager Secure Notes and writes them to files or standard output.

It is intended for services that need secrets at runtime. Store the source values in Bitwarden, describe which values to select in the app configuration, and let Homelab Wallet materialize them where the service can read them instead of hard-coding them in application or deployment files.

## Where secrets are stored

The Bitwarden Password Manager vault is the source of truth: the original secrets stay in your online vault, and Homelab Wallet does not delete or change them. The app reads the selected values when it runs and writes copies to the configured output destinations. Those copies are local to the machine running the app; use an appropriate destination, such as a RAM-backed directory, if they should not persist on disk. Vault access remains protected by your Bitwarden account and credentials.

## How it works

1. The app invokes the [Bitwarden CLI (`bw`)](https://bitwarden.com/help/cli/) to sync and read the vault.
2. It finds the Secure Note named in the configuration and parses the JSON in that note's **Notes** field.
3. It applies the configured JSONPath query. A query must return exactly one value.
4. It writes the result to a file or to standard output.

Custom-field references in resolved JSON values use `${FIELD_NAME}`. Use `$${...}` when the literal text `${...}` should be preserved.

## Requirements

- Go version specified in [`go.mod`](go.mod)
- A Bitwarden Password Manager account
- Bitwarden's standalone native CLI executable, `bw`

For local development on an Apple Silicon Mac, install a matching CLI release asset with:

```sh
./install_bw.sh -a bw-bit-macos-arm64-2026.11.1
```

The asset name is `bw-bit-macos-arm64-2026.11.1`: `bit` identifies the licensed Bitwarden CLI build, `macos` is the platform, `arm64` is the Apple Silicon architecture, and `2026.11.1` is the release version. To install another release, use the matching asset name from the [Bitwarden CLI releases](https://github.com/bitwarden/clients/releases). The script installs `bw` in the project directory and replaces an existing `bw`; the app uses it by default when `cli_path` is empty.

## Configure Bitwarden

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

## Configure the app locally

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

### Plaintext at runtime

Homelab Wallet reads `client_id`, `client_secret`, and `master_password` from the configured values or `file://` paths as plaintext while it runs. In production, systemd credentials encrypt these Bitwarden credentials at rest and provide decrypted files to the service at runtime through `CREDENTIALS_DIRECTORY`.

Values selected from the Secure Note, such as the Beszel token and key, are written as plaintext to the output paths configured in `secrets`. In production, those paths can be under `/dev/shm`, which is memory-backed. These output files are separate from the encrypted Bitwarden credential files.

## Run

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
