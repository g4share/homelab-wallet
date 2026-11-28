package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Config contains the Bitwarden settings and secret selection rules.
type Config struct {
	Bitwarden Bitwarden     `json:"bitwarden"`
	Secrets   []SecretQuery `json:"secrets"`
}

// Bitwarden contains the settings used to access the Bitwarden CLI.
type Bitwarden struct {
	CLIPath        string `json:"cli_path"`
	ServerURL      string `json:"server_url"`
	ClientID       string `json:"client_id"`
	ClientSecret   string `json:"client_secret"`
	MasterPassword string `json:"master_password"`
}

// SecretQuery selects a value from a Bitwarden secure note.
type SecretQuery struct {
	Note   string `json:"note"`
	Query  string `json:"query"`
	Output string `json:"output"`
}

// Load reads, defaults, and validates the JSON configuration at path.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}
	if strings.TrimSpace(cfg.Bitwarden.CLIPath) == "" {
		cfg.Bitwarden.CLIPath = "bw"
	}
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// validate checks that every secret selection has a note and query.
func validate(cfg Config) error {
	if len(cfg.Secrets) == 0 {
		return errors.New("at least one entry is required in secrets")
	}
	for index, selection := range cfg.Secrets {
		if strings.TrimSpace(selection.Note) == "" {
			return fmt.Errorf("secrets[%d].note must be set", index)
		}
		if strings.TrimSpace(selection.Query) == "" {
			return fmt.Errorf("secrets[%d].query for note %q must be set", index, selection.Note)
		}
		if strings.TrimSpace(selection.Output) == "" {
			return fmt.Errorf("secrets[%d].output for note %q must be set", index, selection.Note)
		}
	}
	return nil
}
