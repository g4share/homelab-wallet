package resolver

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type fileOutputResolver struct {
	fileReference
}

// NewFileOutHandler creates a handler that writes values to a file.
func NewFileOutHandler() OutputHandler {
	return new(fileOutputResolver)
}

func (r *fileOutputResolver) Write(payload, value string) (string, error) {
	pathPayload, optionPayload, _ := strings.Cut(payload, "?")
	path, err := r.resolvePath(pathPayload)
	if err != nil {
		return "", err
	}
	options, err := parseFileOutputOptions(optionPayload)
	if err != nil {
		return "", fmt.Errorf("parse file output options: %w", err)
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve output path %q: %w", path, err)
	}

	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, options.dirMode); err != nil {
		return "", fmt.Errorf("create directory for secret file %q: %w", path, err)
	}
	if err := writeSecretFile(path, directory, value, options.fileMode, options.dirMode); err != nil {
		return "", fmt.Errorf("write secret file %q: %w", path, err)
	}
	for _, option := range options.items {
		if err := option.handler(path, option.value); err != nil {
			return "", fmt.Errorf("apply file output option %q: %w", option.key, err)
		}
	}
	fileInfo, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("inspect output file %q: %w", path, err)
	}
	directoryInfo, err := os.Stat(directory)
	if err != nil {
		return "", fmt.Errorf("inspect output directory %q: %w", directory, err)
	}
	group := "unchanged"
	groupIndex := slices.IndexFunc(options.items, func(option fileOutputOption) bool {
		return option.key == "group"
	})
	if groupIndex >= 0 {
		group = options.items[groupIndex].value
	}
	slog.Debug("file output permissions",
		"path", path,
		"mode", fmt.Sprintf("%04o", fileInfo.Mode().Perm()),
		"dir_mode", fmt.Sprintf("%04o", directoryInfo.Mode().Perm()),
		"group_option", group,
	)

	return path, nil
}

func writeSecretFile(path, directory, value string, fileMode, directoryMode os.FileMode) (returnErr error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, fileMode)
	if err != nil {
		return fmt.Errorf("open output file: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil && returnErr == nil {
			returnErr = fmt.Errorf("close output file: %w", err)
		}
	}()

	if err := file.Chmod(fileMode); err != nil {
		return fmt.Errorf("set permissions on output file: %w", err)
	}
	if err := os.Chmod(directory, directoryMode); err != nil {
		return fmt.Errorf("set permissions on output directory: %w", err)
	}
	if err := file.Truncate(0); err != nil {
		return fmt.Errorf("truncate output file: %w", err)
	}
	if _, err := file.Write([]byte(value)); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}
	return nil
}
