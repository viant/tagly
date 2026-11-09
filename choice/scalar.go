package choice

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
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
	// Nullable is representation-level nullability, independent of pointers.
	Nullable           bool
	OpenIntegers       bool
	OpenStrings        bool
	Minimum            json.Number
	Maximum            json.Number
	Integers           []json.Number
	Strings            []string
	StringAlternatives []StringAlternative
}

// StringAlternative is one string branch of an input anyOf schema. Consumers
// must emit both minLength and maxLength as Length, including zero, unless
// VariableLength is true (Length is then only the minimum). Exact rune
// length prevents JavaScript's $ from accepting before a final newline, without
// lookahead syntax unsupported by Go's regexp-based schema validators.
type StringAlternative struct {
	Pattern        string
	Length         int
	VariableLength bool
}

func (e *Enum) parseAliases(tag reflect.StructTag) error {
	_, hasAlias := tag.Lookup("enumAlias")
	_, hasCase := tag.Lookup("enumCase")
	trimPolicy, hasTrim := tag.Lookup("enumTrim")
	integerPolicy, hasInteger := tag.Lookup("enumInteger")
	stringPolicy, hasUnknown := tag.Lookup("enumUnknownString")
	unknownIntegerPolicy, hasUnknownInteger := tag.Lookup("enumUnknownInteger")
	if e.Accept != Scalar {
		if hasAlias || hasCase || hasTrim || hasInteger || hasUnknown || hasUnknownInteger {
			return fmt.Errorf("enum input policies require enumAccept scalar")
		}
		return nil
	}
	if hasTrim {
		if trimPolicy != "space" {
			return fmt.Errorf("unsupported enumTrim %q", trimPolicy)
		}
		e.trimSpace = true
	}
	if hasInteger {
		if e.Type.Kind() == reflect.String || e.Style != Ordinal {
			return fmt.Errorf("integer passthrough requires integer storage and ordinal output")
		}
		if integerPolicy != "passthrough" {
			return fmt.Errorf("unsupported enumInteger %q", integerPolicy)
		}
		e.integerPassthrough = true
	}
	if hasUnknownInteger {
		if hasInteger {
			return fmt.Errorf("enumUnknownInteger cannot combine with integer passthrough")
		}
		if e.Style != defaultStyle(e.Type) {
			return fmt.Errorf("enumUnknownInteger requires output matching the storage representation")
		}
		number, err := e.declaredFallback(unknownIntegerPolicy)
		if err != nil {
			return err
		}
		e.unknownInteger = &number
		// Known and unknown numeric inputs share the same bounded domain.
		for _, entry := range e.Entries {
			if err := e.validateFallbackInteger(entry.Ordinal); err != nil {
				return err
			}
		}
	}
	if hasUnknown {
		var number json.Number
		var err error
		if e.Type.Kind() == reflect.String {
			if e.Style != Literal {
				return fmt.Errorf("string fallback requires literal output")
			}
			number, err = e.declaredFallback(stringPolicy)
		} else {
			if e.Style != Ordinal {
				return fmt.Errorf("integer string fallback requires ordinal output")
			}
			number, err = ordinal(stringPolicy)
			if err == nil {
				_, err = e.storage(number)
			}
		}
		if err != nil {
			return err
		}
		e.unknownString = &number
	}

	switch tag.Get("enumCase") {

	case "":
	case "upper":
		e.upper = true
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

// validateFallbackInteger checks input bounds before an unknown value is mapped.
// Integer storage retains its actual width; string storage uses native int.
func (e *Enum) validateFallbackInteger(code json.Number) error {
	if e.Type.Kind() != reflect.String {
		_, err := e.storage(code)
		return err
	}
	if _, err := strconv.ParseInt(string(code), 10, strconv.IntSize); err != nil {
		return fmt.Errorf("enum integer %s exceeds native int input range", code)
	}
	return nil
}

// declaredFallback resolves a canonical choice, never a new storage value.
func (e *Enum) declaredFallback(raw string) (json.Number, error) {
	number, err := ordinal(raw)
	if err != nil {
		return "", err
	}
	for _, entry := range e.Entries {
		if entry.Ordinal == number {
			return number, nil
		}
	}
	return "", fmt.Errorf("enum fallback refers to undeclared ordinal %s", number)
}

func (e *Enum) equalInput(a, b string) bool {
	return a == b || e.fold && strings.EqualFold(a, b) || e.upper && strings.ToUpper(a) == strings.ToUpper(b)
}

func (e *Enum) addAlias(text string, index int) error {
	if e.trimSpace {
		text = strings.TrimSpace(text)
	}
	for _, old := range e.aliases {
		if !e.equalInput(old.text, text) {
			continue
		}
		if old.entry != index && !(e.union != nil && e.Entries[old.entry].Literal == e.Entries[index].Literal) {
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
		if e.unknownInteger != nil {
			if err := e.validateFallbackInteger(code); err != nil {
				return nil, err
			}
		}
		if e.integerPassthrough {
			if _, err := e.storage(code); err != nil {
				return nil, err
			}
			return json.Marshal(code)
		}
		for i, entry := range e.Entries {
			if entry.Ordinal == code {
				index = i
				break
			}
		}
	case string:
		if e.trimSpace {
			actual = strings.TrimSpace(actual)
		}
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
		var fallback *json.Number
		switch value.(type) {
		case string:
			fallback = e.unknownString
		case json.Number:
			fallback = e.unknownInteger
		}
		if fallback != nil {
			if e.Type.Kind() != reflect.String {
				return json.Marshal(*fallback)
			}
			for i, entry := range e.Entries {
				if entry.Ordinal == *fallback {
					index = i
					break
				}
			}
		}
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
	if e.union != nil {
		return e.scalarUnionDomain()
	}
	if e.Accept != Scalar {
		return nil
	}
	result := &InputDomain{OpenIntegers: e.integerPassthrough || e.unknownInteger != nil, OpenStrings: e.unknownString != nil}
	if result.OpenIntegers {
		result.Minimum, result.Maximum = e.integerBounds()
	}
	for _, entry := range e.Entries {
		result.Integers = append(result.Integers, entry.Ordinal)
	}
	if result.OpenStrings {
		return result
	}
	for _, alias := range e.aliases {
		result.Strings = append(result.Strings, alias.text)
		pattern := enumInputPattern(alias.text, e.fold, e.upper)
		if e.trimSpace {
			pattern = trimSpacePattern + "*(?:" + pattern + ")" + trimSpacePattern + "*"
		}
		result.StringAlternatives = append(result.StringAlternatives, StringAlternative{
			Pattern:        "^(?:" + pattern + ")$",
			Length:         utf8.RuneCountInString(alias.text),
			VariableLength: e.trimSpace,
		})
	}
	return result
}

// Go strings.TrimSpace uses Unicode White_Space. Literal Unicode characters
// keep this class valid in both Go and ECMAScript schema regexp engines.
// U+180E and U+FEFF are intentionally absent; generic \\s is not equivalent.
const trimSpacePattern = `[\x09-\x0D\x20` + "\u0085\u00a0\u1680\u2000-\u200a\u2028\u2029\u202f\u205f\u3000" + `]`

func enumInputPattern(text string, fold, upper bool) string {
	if !fold && !upper {
		return regexp.QuoteMeta(text)
	}
	var result strings.Builder
	for _, r := range text {
		group := []rune{r}
		if upper {
			normalized := unicode.ToUpper(r)
			group = []rune{normalized}
			for _, span := range unicode.CaseRanges {
				for candidate := rune(span.Lo); candidate <= rune(span.Hi); candidate++ {
					if candidate != normalized && unicode.ToUpper(candidate) == normalized {
						group = append(group, candidate)
					}
				}
			}
		} else {
			for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
				group = append(group, next)
			}
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

func (e *Enum) integerBounds() (json.Number, json.Number) {
	typ := e.Type
	if typ.Kind() == reflect.String {
		typ = reflect.TypeOf(int(0))
	}
	bits := typ.Bits()
	if typ.Kind() >= reflect.Int && typ.Kind() <= reflect.Int64 {
		maximum := int64(^uint64(0) >> uint(65-bits))
		minimum := -maximum - 1
		return json.Number(strconv.FormatInt(minimum, 10)), json.Number(strconv.FormatInt(maximum, 10))
	}
	maximum := ^uint64(0) >> uint(64-bits)
	return "0", json.Number(strconv.FormatUint(maximum, 10))
}
