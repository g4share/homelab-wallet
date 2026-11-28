package resolver

import (
	"fmt"
	"os"
)

type fileResolver struct {
	fileReference
}

// NewFileInHandler creates a handler that reads a file reference.
func NewFileInHandler() InputHandler {
	return new(fileResolver)
}

func (r *fileResolver) Read(payload string) (string, error) {
	path, err := r.resolvePath(payload)
	if err != nil {
		return "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %q: %w", path, err)
	}

	return string(data), nil
}
