package tags

import (
	"fmt"
	"github.com/viant/tagly/format/text"
	"reflect"
	"strconv"
	"strings"
)

const TagName = "tag"

type (
	//Tag defines generic tag
	Tag struct {
		Name   string
		Values Values
	}

	//Tags represents tags
	Tags []*Tag
)

// Stringify returns stringified tags representations
func (t Tags) Stringify() string {
	builder := strings.Builder{}
	for i, tag := range t {
		if i > 0 {
			builder.WriteString(" ")
		}
		if string(tag.Values) == "" {
			continue
		}
		builder.WriteString(tag.Name)
		builder.WriteString(":")
		value := strconv.Quote(string(tag.Values))
		builder.WriteString(value)
	}
	return builder.String()
}

// Append appends tag value element
func (e *Tag) Append(value string) {
	if value == "" {
		return
	}
	if e.Values == "" {
		e.Values = Values(value)
		return
	}
	e.Values = Values(string(e.Values) + "," + value)
}

// Lookup returns matched by name tag
func (t Tags) Lookup(name string) *Tag {
	for _, candidate := range t {
		if candidate.Name == name {
			return candidate
		}
	}
	return nil
}

// Index returns matched by name tag index
func (t Tags) Index(name string) int {
	for i, candidate := range t {
		if candidate.Name == name {
			return i
		}
	}
	return -1
}

func (t *Tags) SetIfNotFound(tag string, value string) {
	if t.Lookup(tag) != nil {
		return
	}
	*t = append(*t, &Tag{Name: tag, Values: Values(value)})
}

// Set sets tag value, if tag exists, overrides it
func (t *Tags) Set(tag string, value string) {
	if len(value) == 0 {
		return
	}
	aTag := t.Lookup(tag)
	if aTag == nil {
		aTag = &Tag{Name: tag}
		*t = append(*t, aTag)
	}
	aTag.Values = Values(value)
}

// SetTag sets tags, if tag exists, overrides it
func (t *Tags) SetTag(aTag *Tag) {
	if aTag == nil {
		return
	}
	if index := t.Index(aTag.Name); index != -1 {
		(*t)[index] = aTag
		return
	}
	*t = append(*t, aTag)
}

// Append appends tag values to existing tag or create a new tag
func (t *Tags) Append(tag string, value string) {
	if len(value) == 0 {
		return
	}
	aTag := t.Lookup(tag)
	if aTag == nil {
		aTag = &Tag{}
		*t = append(*t, aTag)
	}
	aTag.Append(value)
}

// NewTags create a tags for supplied tag literal
func NewTags(tagLiteral string) Tags {
	var result []*Tag
	for tagLiteral != "" {
		i := 0
		for i < len(tagLiteral) && tagLiteral[i] == ' ' {
			i++
		}
		tagLiteral = tagLiteral[i:]
		if tagLiteral == "" {
			break
		}
		i = 0
		for i < len(tagLiteral) && tagLiteral[i] > ' ' && tagLiteral[i] != ':' && tagLiteral[i] != '"' && tagLiteral[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tagLiteral) || tagLiteral[i] != ':' || tagLiteral[i+1] != '"' {
			break
		}
		name := tagLiteral[:i]
		tagLiteral = tagLiteral[i+1:]
		i = 1
		for i < len(tagLiteral) && tagLiteral[i] != '"' {
			if tagLiteral[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tagLiteral) {
			break
		}
		quotedValue := tagLiteral[:i+1]
		tagLiteral = tagLiteral[i+1:]
		value, err := strconv.Unquote(quotedValue)
		if err != nil {
			break
		}
		aTag := &Tag{Name: name, Values: Values(value)}
		result = append(result, aTag)
	}
	return result
}

// NewTag creates a tag for supplied tag type
func NewTag(name string, value interface{}) *Tag {
	if value == nil {
		return nil
	}

	rValue := reflect.ValueOf(value)
	if rValue.Kind() == reflect.Ptr {
		rValue = rValue.Elem()
	}
	rType := rValue.Type()
	ret := &Tag{Name: name}
	for i := 0; i < rValue.NumField(); i++ {
		fieldValue := rValue.Field(i)
		field := rType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		omitEmpty := false
		name := field.Tag.Get(TagName)
		if name == "-" {
			continue
		}
		if index := strings.Index(name, ",omitempty"); index != -1 {
			omitEmpty = true
			name = name[:index]
		}
		if name == "" {
			caseFormat := text.DetectCaseFormat(field.Name)
			name = caseFormat.Format(field.Name, text.CaseFormatLowerCamel)
		}

		value := fieldValue.Interface()

		switch actual := value.(type) {
		case string:
			if omitEmpty && actual == "" {
				continue
			}
			ret.Append(name + "=" + wrapValueIfNeeded(actual))
		case *string:
			if actual == nil {
				continue
			}
			ret.Append(name + "=" + wrapValueIfNeeded(*actual))
		case int:
			if omitEmpty && actual == 0 {
				continue
			}

			ret.Append(name + "=" + strconv.Itoa(actual))
		case *int:
			if actual == nil {
				continue
			}
			ret.Append(name + "=" + strconv.Itoa(*actual))
		case bool:
			if omitEmpty && !actual {
				continue
			}

			ret.Append(name + "=" + strconv.FormatBool(actual))
		case *bool:
			if actual == nil {
				continue
			}
			ret.Append(name + "=" + strconv.FormatBool(*actual))
		case float64:
			if omitEmpty && actual == 0.0 {
				continue
			}

			ret.Append(name + "=" + strconv.FormatFloat(actual, 'f', -1, 32))
		default:
			aText := fmt.Sprintf("%v", actual)
			ret.Append(name + "=" + wrapValueIfNeeded(aText))
		}
	}
	return ret
}

func wrapValueIfNeeded(actual string) string {
	if strings.Contains(actual, ",") && !strings.HasPrefix(actual, "{") {
		actual = "{" + actual + "}"
	}
	return actual
}
