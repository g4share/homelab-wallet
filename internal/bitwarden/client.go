package bitwarden

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"homelab-wallet/internal/config"
)

// Client wraps the Bitwarden CLI operations used by the application.
type Client struct {
	cliPath string
	config  config.Bitwarden
}

type cliStatus struct {
	Status    string `json:"status"`
	ServerURL string `json:"serverUrl"`
}

// ResolveCLI finds the CLI executable, resolving relative paths from the config file.
func ResolveCLI(path, configPath string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("bitwarden.cli_path must be set in config.json")
	}
	if !filepath.IsAbs(path) && strings.ContainsRune(path, os.PathSeparator) {
		path = filepath.Join(filepath.Dir(configPath), path)
		var err error
		path, err = filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("resolve Bitwarden CLI path: %w", err)
		}
	}
	resolved, err := exec.LookPath(path)
	if err != nil {
		return "", fmt.Errorf("configured Bitwarden CLI path %q is not executable; set bitwarden.cli_path to its full path", path)
	}
	return resolved, nil
}

// New creates a Bitwarden CLI client.
func New(cfg config.Bitwarden, cliPath string) *Client {
	return &Client{cliPath: cliPath, config: cfg}
}

// Session authenticates and unlocks the vault, then returns its session token.
func (c *Client) session() (string, error) {
	serverURL := strings.TrimRight(strings.TrimSpace(c.config.ServerURL), "/")
	status, err := c.readStatus()
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(status.Status, "unauthenticated") &&
		!strings.EqualFold(status.Status, "locked") &&
		!strings.EqualFold(status.Status, "unlocked") {
		return "", fmt.Errorf("unsupported Bitwarden CLI status %q", status.Status)
	}

	currentServerURL := strings.TrimRight(strings.TrimSpace(status.ServerURL), "/")
	if serverURL != "" && !strings.EqualFold(currentServerURL, serverURL) {
		if !strings.EqualFold(status.Status, "unauthenticated") {
			if _, err := runCLI(c.cliPath, os.Environ(), "logout"); err != nil {
				return "", fmt.Errorf("could not log out before changing the Bitwarden server: %w", err)
			}
		}
		if _, err := runCLI(c.cliPath, os.Environ(), "config", "server", serverURL); err != nil {
			return "", fmt.Errorf("could not configure the Bitwarden server %q: %w", serverURL, err)
		}
		status, err = c.readStatus()
		if err != nil {
			return "", err
		}
	}

	if strings.EqualFold(status.Status, "unauthenticated") {
		if c.config.ClientID == "" || c.config.ClientSecret == "" {
			return "", errors.New("fill in bitwarden.client_id and bitwarden.client_secret in config.json, or sign in with bw login first")
		}
		env := withEnv(os.Environ(), map[string]string{
			"BW_CLIENTID":     c.config.ClientID,
			"BW_CLIENTSECRET": c.config.ClientSecret,
		})
		if _, err := runCLI(c.cliPath, env, "login", "--apikey"); err != nil {
			return "", errors.New("could not log in to Bitwarden with the API key; check client_id and client_secret")
		}
	} else if !strings.EqualFold(status.Status, "locked") && !strings.EqualFold(status.Status, "unlocked") {
		return "", fmt.Errorf("unsupported Bitwarden CLI status %q", status.Status)
	}

	unlockArgs := []string{"unlock", "--raw"}
	env := os.Environ()
	if c.config.MasterPassword != "" {
		env = withEnv(env, map[string]string{"BW_MASTER_PASSWORD": c.config.MasterPassword})
		unlockArgs = []string{"unlock", "--passwordenv", "BW_MASTER_PASSWORD", "--raw"}
	}
	output, err := runCLI(c.cliPath, env, unlockArgs...)
	if err != nil {
		return "", errors.New("could not unlock the Bitwarden vault; check master_password or unlock interactively")
	}
	session := strings.TrimSpace(string(output))
	if session == "" {
		return "", errors.New("received an empty session token from the Bitwarden CLI")
	}
	return session, nil
}

func (c *Client) readStatus() (cliStatus, error) {
	statusData, err := runCLI(c.cliPath, os.Environ(), "status")
	if err != nil {
		return cliStatus{}, fmt.Errorf("could not run `bw status`: %w", err)
	}
	var status cliStatus
	if err := json.Unmarshal(statusData, &status); err != nil {
		return cliStatus{}, errors.New("received an unrecognized status response from the Bitwarden CLI")
	}
	return status, nil
}

// Sync authenticates, unlocks, syncs the local vault, and returns the session token.
func (c *Client) Sync() (string, error) {
	session, err := c.session()
	if err != nil {
		return "", err
	}
	env := withEnv(os.Environ(), map[string]string{"BW_SESSION": session})
	if _, err := runCLI(c.cliPath, env, "sync"); err != nil {
		return "", fmt.Errorf("could not sync the Bitwarden vault: %w", err)
	}
	return session, nil
}

// SecureNotes returns all secure notes, preserving their Notes contents as raw strings.
func (c *Client) SecureNotes(session string) ([]SecureNote, error) {
	env := withEnv(os.Environ(), map[string]string{"BW_SESSION": session})
	output, err := runCLI(c.cliPath, env, "list", "items")
	if err != nil {
		return nil, fmt.Errorf("could not list Bitwarden items: %w", err)
	}

	var items []secureNoteWithID
	if err := json.Unmarshal(output, &items); err != nil {
		return nil, fmt.Errorf("received unexpected item list from the Bitwarden CLI: %w", err)
	}

	const secureNoteType = 2
	notes := make([]SecureNote, 0, len(items))
	for _, item := range items {
		if item.Type != secureNoteType {
			continue
		}
		notes = append(notes, SecureNote{Name: item.Name, Notes: item.Notes, Fields: item.Fields})
	}
	return notes, nil
}

func runCLI(cli string, env []string, args ...string) ([]byte, error) {
	cmd := exec.Command(cli, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return stdout.Bytes(), nil
}

func withEnv(env []string, values map[string]string) []string {
	result := make([]string, 0, len(env)+len(values))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if _, replace := values[key]; !replace {
			result = append(result, entry)
		}
	}
	for key, value := range values {
		result = append(result, key+"="+value)
	}
	return result
}
