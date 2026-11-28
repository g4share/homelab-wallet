package resolver

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

const filePrefix = "file://"

type fileReference struct{}

func (fileReference) Match(value string) (string, bool) {
	if !strings.HasPrefix(value, filePrefix) {
		return "", false
	}
	return strings.TrimPrefix(value, filePrefix), true
}

func (fileReference) resolvePath(payload string) (string, error) {
	path, err := expandEnvironment(payload)
	if err != nil {
		return "", err
	}
	if path == "" {
		return "", errors.New("file path must be set after " + filePrefix)
	}
	return path, nil
}

func expandEnvironment(path string) (string, error) {
	var expansionErr error
	expanded := os.Expand(path, func(name string) string {
		value, exists := os.LookupEnv(name)
		if !exists || value == "" {
			if expansionErr == nil {
				expansionErr = fmt.Errorf("environment variable %s is not set", name)
			}
			return ""
		}
		return value
	})
	if expansionErr != nil {
		return "", expansionErr
	}
	return expanded, nil
}
