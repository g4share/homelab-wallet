package resolver

import (
	"fmt"
	"maps"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

// fileOutputOption is a parsed query parameter and the handler that applies it.
type fileOutputOption struct {
	key     string
	value   string
	handler fileOutputOptionHandler
}

type fileOutputOptions struct {
	fileMode os.FileMode
	dirMode  os.FileMode
	items    []fileOutputOption
}

// fileOutputOptionHandler applies one query parameter to an output file.
type fileOutputOptionHandler func(path, value string) error

// fileOutputOptionHandlers maps supported query keys to their filesystem handlers.
// Add a handler here to support another file output option.
var fileOutputOptionHandlers = map[string]fileOutputOptionHandler{
	"group": applyFileOutputGroup,
}

var supportedFileOutputOptions = map[string]bool{
	"group":    true,
	"mode":     true,
	"dir_mode": true,
}

// parseFileOutputOptions parses the optional query after a file:// output path.
// File and directory modes default to 0600 and 0700; query values override
// those defaults. Unsupported or repeated keys are errors.
func parseFileOutputOptions(payload string) (fileOutputOptions, error) {
	values := url.Values{
		"mode":     {"0600"},
		"dir_mode": {"0700"},
	}
	queryValues, err := url.ParseQuery(payload)
	if err != nil {
		return fileOutputOptions{}, fmt.Errorf("parse query: %w", err)
	}
	maps.Copy(values, queryValues)

	options := fileOutputOptions{}
	for _, key := range []string{"group", "mode", "dir_mode"} {
		valuesForKey, exists := values[key]
		if !exists {
			continue
		}
		if len(valuesForKey) != 1 {
			return fileOutputOptions{}, fmt.Errorf("file output option %q must be specified once", key)
		}
		value := valuesForKey[0]
		if key == "group" {
			options.items = append(options.items, fileOutputOption{
				key: key, value: value, handler: fileOutputOptionHandlers[key],
			})
		} else {
			mode, err := parseFilePermissionMode(value)
			if err != nil {
				return fileOutputOptions{}, fmt.Errorf("parse %s: %w", key, err)
			}
			if key == "mode" {
				options.fileMode = mode
			} else {
				options.dirMode = mode
			}
		}
	}
	for key := range values {
		if !supportedFileOutputOptions[key] {
			return fileOutputOptions{}, fmt.Errorf("unsupported file output option %q", key)
		}
	}
	return options, nil
}

// applyFileOutputGroup sets the output directory and file group to an existing host group.
// The writing process must be permitted to change files to this group.
func applyFileOutputGroup(path, value string) error {
	group, err := user.LookupGroup(value)
	if err != nil {
		return fmt.Errorf("look up group %q: %w", value, err)
	}
	groupID, err := strconv.Atoi(group.Gid)
	if err != nil {
		return fmt.Errorf("parse GID for group %q: %w", value, err)
	}
	directory := filepath.Dir(path)
	if err := os.Chown(directory, -1, groupID); err != nil {
		return fmt.Errorf("set group for directory %q: %w", directory, err)
	}
	if err := os.Chown(path, -1, groupID); err != nil {
		return fmt.Errorf("set group for file %q: %w", path, err)
	}
	return nil
}

// parseFilePermissionMode parses a permission value written in octal notation.
func parseFilePermissionMode(value string) (os.FileMode, error) {
	mode, err := strconv.ParseUint(value, 8, 9)
	if err != nil {
		return 0, fmt.Errorf("permissions %q must be an octal mode from 0000 to 0777", value)
	}
	return os.FileMode(mode), nil
}
