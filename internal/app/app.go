package app

import (
	"fmt"
	"log/slog"

	"homelab-wallet/internal/bitwarden"
	"homelab-wallet/internal/config"
	"homelab-wallet/internal/resolver"
	"homelab-wallet/internal/secrets"
)

// App holds dependencies configured before the application runs.
type App struct {
	valueResolver *resolver.Resolver
}

// New creates an application with its input and output handlers registered.
func New() (*App, error) {
	valueResolver := resolver.New()
	if err := valueResolver.RegisterInput("bitwarden", resolver.NewFileInHandler(), resolver.NewCleartextHandler()); err != nil {
		return nil, err
	}
	if err := valueResolver.RegisterOutput("secret", resolver.NewFileOutHandler(), resolver.NewStdoutHandler()); err != nil {
		return nil, err
	}
	return &App{valueResolver: valueResolver}, nil
}

func (application *App) Run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if err := resolveBitwardenValues(&cfg.Bitwarden, application.valueResolver); err != nil {
		return err
	}

	cliPath, err := bitwarden.ResolveCLI(cfg.Bitwarden.CLIPath, configPath)
	if err != nil {
		return err
	}
	client := bitwarden.New(cfg.Bitwarden, cliPath)

	session, err := client.Sync()
	if err != nil {
		return err
	}
	notes, err := client.SecureNotes(session)
	if err != nil {
		return err
	}

	notesByName := make(map[string][]bitwarden.SecureNote, len(notes))
	for _, note := range notes {
		notesByName[note.Name] = append(notesByName[note.Name], note)
	}
	for index, selection := range cfg.Secrets {
		matchingNotes := notesByName[selection.Note]
		if len(matchingNotes) == 0 {
			return fmt.Errorf("could not find a Bitwarden secure note with the exact title %q", selection.Note)
		}
		if len(matchingNotes) > 1 {
			return fmt.Errorf("found multiple Bitwarden secure notes with the exact title %q", selection.Note)
		}
		item := matchingNotes[0]

		matches, err := secrets.ProcessNote(item.Notes, selection.Query, selection.Note, item.ResolveCustomField)
		if err != nil {
			return err
		}
		if len(matches) != 1 {
			return fmt.Errorf("query for note %q returned %d matches; exactly one is required", selection.Note, len(matches))
		}
		filePath, err := application.valueResolver.WriteOutput(
			"secret",
			fmt.Sprintf("secrets[%d].output", index),
			selection.Output,
			matches[0],
		)
		if err != nil {
			return err
		}
		if filePath != "" {
			slog.Info("wrote secret file", "path", filePath)
		}
	}
	return nil
}

func resolveBitwardenValues(cfg *config.Bitwarden, valueResolver *resolver.Resolver) error {
	fields := []*string{
		&cfg.ClientID,
		&cfg.ClientSecret,
		&cfg.MasterPassword,
	}
	for _, field := range fields {
		resolved, err := valueResolver.ResolveInput("bitwarden", *field, *field)
		if err != nil {
			return err
		}
		*field = resolved
	}
	return nil
}
