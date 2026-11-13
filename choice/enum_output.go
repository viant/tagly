package choice

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/viant/tagly/tags"
)

// parseOutputAliases compiles output-only normalization into the enum plan.
// Input aliases and their case/whitespace policies have a separate owner.
func (e *Enum) parseOutputAliases(tag reflect.StructTag) error {
	_, hasAlias := tag.Lookup("enumOutputAlias")
	_, hasUnknown := tag.Lookup("enumOutputUnknown")
	if !hasAlias && !hasUnknown {
		return nil
	}
	if e.Type.Kind() != reflect.String || e.Style != Literal {
		return fmt.Errorf("enum output policies require string storage and literal output")
	}
	parsed, err := tags.Parse(string(tag))
	if err != nil {
		return err
	}
	e.outputLiterals = make(map[string]string, len(e.Entries))
	ordinals := make(map[string]string, len(e.Entries))
	for _, entry := range e.Entries {
		e.outputLiterals[entry.Literal] = entry.Literal
		ordinals[string(entry.Ordinal)] = entry.Literal
	}
	for _, item := range parsed {
		switch item.Name {
		case "enumOutputUnknown":
			if e.rejectOutputUnknown {
				return fmt.Errorf("duplicate enumOutputUnknown declaration")
			}
			if string(item.Values) != "reject" {
				return fmt.Errorf("enumOutputUnknown must be reject, got %q", item.Values)
			}
			e.rejectOutputUnknown = true
		case "enumOutputAlias":
			code, token, ok := strings.Cut(string(item.Values), ":")
			if !ok {
				return fmt.Errorf("enumOutputAlias requires ordinal:string")
			}
			number, err := ordinal(code)
			if err != nil {
				return fmt.Errorf("enumOutputAlias: %w", err)
			}
			literal, ok := ordinals[string(number)]
			if !ok {
				return fmt.Errorf("enumOutputAlias refers to undeclared ordinal %s", number)
			}
			if _, exists := e.outputLiterals[token]; exists {
				return fmt.Errorf("duplicate or colliding enum output token %q", token)
			}
			e.outputLiterals[token] = literal
		}
	}
	return nil
}
