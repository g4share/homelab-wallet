package secrets

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/theory/jsonpath"
)

// ProcessNote parses note JSON, selects values with JSONPath, and resolves custom field references.
func ProcessNote(noteJSON, query, note string, lookup func(string) (any, error)) ([]any, error) {
	if strings.TrimSpace(noteJSON) == "" {
		return nil, fmt.Errorf("secure note %q has an empty notes field", note)
	}

	var document any
	if err := json.Unmarshal([]byte(noteJSON), &document); err != nil {
		return nil, fmt.Errorf("notes field in secure note %q is not valid JSON: %w", note, err)
	}

	path, err := jsonpath.Parse(query)
	if err != nil {
		return nil, fmt.Errorf("invalid jsonpath for note %q: %w", note, err)
	}
	matches := path.Select(document)
	if len(matches) == 0 {
		return nil, fmt.Errorf("jsonpath for note %q returned no matches", note)
	}

	for i, match := range matches {
		matches[i], err = resolveCustomFieldReferences(match, note, lookup)
		if err != nil {
			return nil, err
		}
	}
	return matches, nil
}

func resolveCustomFieldReferences(value any, note string, lookup func(string) (any, error)) (any, error) {
	switch typedValue := value.(type) {
	case string:
		return resolveStringReference(typedValue, note, lookup)
	case []any:
		resolved := make([]any, len(typedValue))
		for i, item := range typedValue {
			value, err := resolveCustomFieldReferences(item, note, lookup)
			if err != nil {
				return nil, err
			}
			resolved[i] = value
		}
		return resolved, nil
	case map[string]any:
		resolved := make(map[string]any, len(typedValue))
		for key, item := range typedValue {
			value, err := resolveCustomFieldReferences(item, note, lookup)
			if err != nil {
				return nil, err
			}
			resolved[key] = value
		}
		return resolved, nil
	default:
		return value, nil
	}
}

func resolveStringReference(value, note string, lookup func(string) (any, error)) (any, error) {
	if strings.HasPrefix(value, "$${") && strings.HasSuffix(value, "}") {
		return value[1:], nil
	}
	if !strings.HasPrefix(value, "${") || !strings.HasSuffix(value, "}") {
		return value, nil
	}

	name := value[2 : len(value)-1]
	if name == "" {
		return nil, fmt.Errorf("custom field reference in Bitwarden note %q has an empty name", note)
	}
	return lookup(name)
}
