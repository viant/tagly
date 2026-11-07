package choice

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/viant/tagly/tags"
)

type enumAlias struct {
	text  string
	entry int
}

// InputDomain describes the two branches of an opt-in scalar enum. Integers
// retain their exact JSON identity. String alternatives include Unicode
// simple-fold equivalents when selected.
// Ordinary enum modes continue to use WireType and WireValues.
type InputDomain struct {
	Integers           []json.Number
	Strings            []string
	StringAlternatives []StringAlternative
}

// StringAlternative is one string branch of an input anyOf schema. Consumers
// must emit both minLength and maxLength as Length, including zero. Exact rune
// length prevents JavaScript's $ from accepting before a final newline, without
// lookahead syntax unsupported by Go's regexp-based schema validators.
type StringAlternative struct {
	Pattern string
	Length  int
}

func (e *Enum) parseAliases(tag reflect.StructTag) error {
	_, hasAlias := tag.Lookup("enumAlias")
	_, hasCase := tag.Lookup("enumCase")
	if e.Accept != Scalar {
		if hasAlias || hasCase {
			return fmt.Errorf("enumAlias and enumCase require enumAccept scalar")
		}
		return nil
	}
	switch tag.Get("enumCase") {
	case "":
	case "fold":
		e.fold = true
	default:
		return fmt.Errorf("unsupported enumCase %q", tag.Get("enumCase"))
	}
	for i, entry := range e.Entries {
		if err := e.addAlias(entry.Literal, i); err != nil {
			return err
		}
	}
	parsed, err := tags.Parse(string(tag))
	if err != nil {
		return err
	}
	for _, item := range parsed {
		if item.Name != "enumAlias" {
			continue
		}
		code, text, ok := strings.Cut(string(item.Values), ":")
		if !ok {
			return fmt.Errorf("enumAlias requires ordinal:string")
		}
		number, err := ordinal(code)
		if err != nil {
			return err
		}
		index := -1
		for i, entry := range e.Entries {
			if entry.Ordinal == number {
				index = i
				break
			}
		}
		if index == -1 {
			return fmt.Errorf("enumAlias refers to undeclared ordinal %s", number)
		}
		if err := e.addAlias(text, index); err != nil {
			return err
		}
	}
	return nil
}

func (e *Enum) equalInput(a, b string) bool {
	return a == b || e.fold && strings.EqualFold(a, b)
}

func (e *Enum) addAlias(text string, index int) error {
	for _, old := range e.aliases {
		if !e.equalInput(old.text, text) {
			continue
		}
		if old.entry != index {
			return fmt.Errorf("ambiguous enum input %q", text)
		}
		return nil
	}
	e.aliases = append(e.aliases, enumAlias{text, index})
	return nil
}

func (e *Enum) decodeScalar(value any) ([]byte, error) {
	index := -1
	switch actual := value.(type) {
	case json.Number:
		code, err := ordinal(string(actual))
		if err != nil {
			return nil, err
		}
		for i, entry := range e.Entries {
			if entry.Ordinal == code {
				index = i
				break
			}
		}
	case string:
		for _, alias := range e.aliases {
			if e.equalInput(alias.text, actual) {
				index = alias.entry
				break
			}
		}
	default:
		return nil, fmt.Errorf("scalar enum input requires a JSON integer or string")
	}
	if index < 0 {
		return nil, fmt.Errorf("unknown enum input %v", value)
	}
	var stored any = e.Entries[index].Ordinal
	if e.Type.Kind() == reflect.String {
		stored = e.Entries[index].Literal
	}
	if _, err := e.storage(stored); err != nil {
		return nil, err
	}
	return json.Marshal(stored)
}

// InputDomain returns nil for legacy single-representation enums. The returned
// slices are independent of the compiled plan. Numeric token spelling (1 vs
// 1.0) is a runtime lexical restriction that JSON Schema cannot express.
func (e *Enum) InputDomain() *InputDomain {
	if e.Accept != Scalar {
		return nil
	}
	result := &InputDomain{}
	for _, entry := range e.Entries {
		result.Integers = append(result.Integers, entry.Ordinal)
	}
	for _, alias := range e.aliases {
		result.Strings = append(result.Strings, alias.text)
		result.StringAlternatives = append(result.StringAlternatives, StringAlternative{
			Pattern: "^(?:" + enumInputPattern(alias.text, e.fold) + ")$",
			Length:  utf8.RuneCountInString(alias.text),
		})
	}
	return result
}

func enumInputPattern(text string, fold bool) string {
	if !fold {
		return regexp.QuoteMeta(text)
	}
	var result strings.Builder
	for _, r := range text {
		group := []rune{r}
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			group = append(group, next)
		}
		if len(group) == 1 {
			result.WriteString(regexp.QuoteMeta(string(r)))
			continue
		}
		sort.Slice(group, func(i, j int) bool { return group[i] < group[j] })
		result.WriteByte('[')
		for _, member := range group {
			result.WriteString(regexp.QuoteMeta(string(member)))
		}
		result.WriteByte(']')
	}
	return result.String()
}
