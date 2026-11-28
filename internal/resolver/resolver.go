// Package resolver resolves input values and dispatches output destinations.
package resolver

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Matcher recognizes a value and returns the payload for a matching handler.
type Matcher interface {
	Match(value string) (payload string, matched bool)
}

// InputHandler reads or resolves an input value.
type InputHandler interface {
	Matcher
	Read(payload string) (string, error)
}

// OutputHandler writes values to a matching destination.
type OutputHandler interface {
	Matcher
	Write(payload, value string) (string, error)
}

// Resolver stores ordered handler lists under free-form group names.
type Resolver struct {
	inputGroups  map[string][]InputHandler
	outputGroups map[string][]OutputHandler
}

// New creates an empty resolver.
func New() *Resolver {
	return &Resolver{
		inputGroups:  make(map[string][]InputHandler),
		outputGroups: make(map[string][]OutputHandler),
	}
}

// RegisterInput adds an ordered input handler list for a group.
func (r *Resolver) RegisterInput(group string, handlers ...InputHandler) error {
	return registerGroup(r.inputGroups, group, handlers, func(handler InputHandler) bool {
		return handler == nil
	})
}

// RegisterOutput adds an ordered output handler list for a group.
func (r *Resolver) RegisterOutput(group string, handlers ...OutputHandler) error {
	return registerGroup(r.outputGroups, group, handlers, func(handler OutputHandler) bool {
		return handler == nil
	})
}

func registerGroup[T any](groups map[string][]T, group string, handlers []T, isNil func(T) bool) error {
	group = strings.TrimSpace(group)
	if group == "" {
		return errors.New("resolver group must be set")
	}
	if len(handlers) == 0 {
		return errors.New("at least one handler must be registered")
	}
	if slices.ContainsFunc(handlers, isNil) {
		return errors.New("handler must be set")
	}
	if _, exists := groups[group]; exists {
		return fmt.Errorf("resolver group %q is already registered", group)
	}
	groups[group] = handlers
	return nil
}

// ResolveInput runs the first matching input handler in the requested group.
func (r *Resolver) ResolveInput(group, label, value string) (string, error) {
	handlers, exists := r.inputGroups[group]
	if !exists {
		return "", fmt.Errorf("input resolver group %q is not registered", group)
	}
	return dispatch(group, label, value, handlers,
		func(handler InputHandler, value string) (string, bool) {
			return handler.Match(value)
		},
		func(handler InputHandler, payload string) (string, error) {
			return handler.Read(payload)
		},
	)
}

// WriteOutput runs the first matching output handler in the requested group.
func (r *Resolver) WriteOutput(group, label, destination string, value any) (string, error) {
	handlers, exists := r.outputGroups[group]
	if !exists {
		return "", fmt.Errorf("output resolver group %q is not registered", group)
	}
	text, err := outputValueToString(value)
	if err != nil {
		return "", fmt.Errorf("resolve %s.%s: encode output value: %w", group, label, err)
	}
	return dispatch(group, label, destination, handlers,
		func(handler OutputHandler, value string) (string, bool) {
			return handler.Match(value)
		},
		func(handler OutputHandler, payload string) (string, error) {
			return handler.Write(payload, text)
		},
	)
}

func outputValueToString(value any) (string, error) {
	if text, ok := value.(string); ok {
		return text, nil
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func dispatch[T any](group, label, value string, handlers []T, match func(T, string) (string, bool), execute func(T, string) (string, error)) (string, error) {
	for _, handler := range handlers {
		payload, matched := match(handler, value)
		if !matched {
			continue
		}
		result, err := execute(handler, payload)
		if err != nil {
			return "", fmt.Errorf("resolve %s.%s: %w", group, label, err)
		}
		return result, nil
	}
	return "", errors.New("no resolver handler matched")
}
