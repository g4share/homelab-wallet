package bitwarden

import (
	"fmt"
	"strings"
)

const (
	fieldTypeText    = 0
	fieldTypeHidden  = 1
	fieldTypeBoolean = 2
)

// SecureNote is the application's representation of a Bitwarden item, containing its name, Notes content, and custom fields.
type SecureNote struct {
	Name   string        `json:"name"`
	Notes  string        `json:"notes"`
	Fields []CustomField `json:"fields"`
}

// CustomField is a custom field returned by the Bitwarden CLI.
type CustomField struct {
	Name  string `json:"name"`
	Value any    `json:"value"`
	Type  int    `json:"type"`
}

type secureNoteWithID struct {
	ID     string        `json:"id"`
	Type   int           `json:"type"`
	Name   string        `json:"name"`
	Notes  string        `json:"notes"`
	Fields []CustomField `json:"fields"`
}

// ResolveCustomField returns a custom field value converted according to its Bitwarden field type.
func (note SecureNote) ResolveCustomField(name string) (any, error) {
	var match *CustomField
	for i := range note.Fields {
		if note.Fields[i].Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("custom field %q appears more than once in Bitwarden note %q", name, note.Name)
		}
		match = &note.Fields[i]
	}
	if match == nil {
		return nil, fmt.Errorf("custom field %q was not found in Bitwarden note %q", name, note.Name)
	}

	switch match.Type {
	case fieldTypeText, fieldTypeHidden:
		if match.Value == nil {
			return "", nil
		}
		value, ok := match.Value.(string)
		if !ok {
			return nil, fmt.Errorf("custom field %q in Bitwarden note %q does not contain text", name, note.Name)
		}
		return value, nil
	case fieldTypeBoolean:
		return booleanFieldValue(name, note.Name, match.Value)
	default:
		return nil, fmt.Errorf("custom field %q in Bitwarden note %q has unsupported type %d", name, note.Name, match.Type)
	}
}

func booleanFieldValue(name, note string, value any) (bool, error) {
	switch typedValue := value.(type) {
	case bool:
		return typedValue, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(typedValue)) {
		case "true":
			return true, nil
		case "false", "":
			return false, nil
		default:
			return false, fmt.Errorf("boolean custom field %q in Bitwarden note %q has invalid value %q", name, note, typedValue)
		}
	case nil:
		return false, nil
	default:
		return false, fmt.Errorf("boolean custom field %q in Bitwarden note %q has an invalid value type", name, note)
	}
}
