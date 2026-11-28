package resolver

import "os"

type stdoutResolver struct{}

// NewStdoutHandler creates a handler that writes text to stdout.
func NewStdoutHandler() OutputHandler {
	return new(stdoutResolver)
}

func (*stdoutResolver) Match(destination string) (string, bool) {
	return "", destination == "stdout"
}

func (*stdoutResolver) Write(_, value string) (string, error) {
	_, err := os.Stdout.Write([]byte(value + "\n"))
	return "", err
}
