# Homelab Wallet — Project Guide

## Purpose

Homelab Wallet is a Go application that reads selected data from a Bitwarden Password Manager vault and writes resolved values to configured destinations. The intended account is hosted by Bitwarden. The app uses Bitwarden's standalone native CLI (`bw`) to access the vault.

## Bitwarden data model

- The current example uses a Secure Note titled `beszel-agent`.
- The note's **Notes** field contains JSON. Its `name` labels the payload, and its `items` array holds values with fields such as `type`, `schema`, and `value`.
- The app selects the Secure Note by the title configured as `note`. The JSON `name` does not replace that title.
- Values may refer to custom fields on the same Secure Note with `${FIELD_NAME}`. Names match exactly and are case-sensitive.
- Text and Hidden custom fields resolve to strings. Checkbox/Boolean fields resolve to JSON booleans. Linked fields are unsupported.
- Escape a literal `${...}` as `$${...}`.
- `schema` and `type` are ordinary JSON fields that a configured JSONPath query can use to select an item.

## Application design

- Keep Bitwarden CLI operations in the Bitwarden client. Keep JSONPath selection and custom-field substitution in the application/data-processing layer, outside the CLI client.
- Keep configuration parsing and JSON handling on Go's standard library where practical. Avoid unnecessary dependencies.
- Use the standalone native `bw` executable. Do not recommend the Homebrew CLI formula, which installs Node.js.
- Bitwarden Password Manager API key credentials and the master password are used for authentication. `session_token` is not part of the application's configuration contract.
- Credential values may be literal strings or `file://` references.
- Output handlers include `file://` and `stdout`. File outputs support `group`, `mode`, and `dir_mode`; default file and directory modes are `0600` and `0700` respectively.
- Bitwarden Secrets Manager (`bws`) machine access tokens are a separate product and do not provide access to Password Manager Secure Notes.

## Configuration and sensitive data

- Do not hard-code credentials or include real credentials in documentation, examples, logs, or commits.
- Keep machine-specific configuration out of shared examples. Provide `config.example.json` with placeholders when an example file is needed.
- Never read or modify `config.json` unless the user explicitly asks.
- Treat vault values, resolved secrets, and credential files as sensitive. Do not print their contents while debugging; prefer status, names, paths, and redacted errors.

## Working in this repository

- Make only changes needed for the user's request. Do not run tools, commands, or checks unless requested or necessary to complete that work.
- Keep generated build and cache files inside the project.
- Preserve the existing separation between Bitwarden access, application processing, configuration, and output resolvers.
- When changing behavior, update the relevant documentation and examples so they match the implementation.
