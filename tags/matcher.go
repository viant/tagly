package tags

import (
	"bytes"
	"github.com/viant/parsly"
	"strings"
)

func matchPair(cursor *parsly.Cursor) (string, string) {
	key := ""
	value := ""
	var tokens = []*parsly.Token{scopeBlockMatcher, quotedMatcher}

	eqIndex := bytes.Index(cursor.Input[cursor.Pos:], []byte("="))
	comaIndex := bytes.Index(cursor.Input[cursor.Pos:], []byte(","))
	enclosedIndex := bytes.Index(cursor.Input[cursor.Pos:], []byte("("))

	if eqIndex == -1 {
		if enclosedIndex != -1 && enclosedIndex < comaIndex {
			pos := cursor.Pos
			cursor.Pos += enclosedIndex
			if match := cursor.MatchAny(enclosedMatcher); match.Code == enclosedToken {
				key = string(cursor.Input[pos:cursor.Pos])
				return key, ""
			}
			cursor.Pos = pos
		}
		tokens = append(tokens, comaTerminatorMatcher)
	} else if comaIndex == -1 {
		tokens = append(tokens, eqTerminatorMatcher)
	} else if eqIndex < comaIndex {
		tokens = append(tokens, eqTerminatorMatcher)
	} else {
		tokens = append(tokens, comaTerminatorMatcher)
	}

	match := cursor.MatchAny(tokens...)

	switch match.Code {
	case scopeBlockToken:
		value = match.Text(cursor)
		match = cursor.MatchAny(comaTerminatorMatcher)
	case quotedToken:
		value = match.Text(cursor)
		match = cursor.MatchAny(comaTerminatorMatcher)
	case comaTerminatorToken:
		value = match.Text(cursor)
		value = value[:len(value)-1] //exclude ,
	case enclosedToken:
		value = match.Text(cursor)
		match = cursor.MatchAny(comaTerminatorMatcher)
	case eqTerminatorToken:
		key = match.Text(cursor)
		key = key[:len(key)-1]
		match = cursor.MatchAny(scopeBlockMatcher, quotedMatcher, comaTerminatorMatcher)
		switch match.Code {
		case scopeBlockToken, quotedToken:
			value = match.Text(cursor)
			match = cursor.MatchAny(comaTerminatorMatcher)
		case comaTerminatorToken:
			value = match.Text(cursor)
			value = value[:len(value)-1]
			cursor.Pos--

		default:
			if cursor.Pos < len(cursor.Input) {
				value = string(cursor.Input[cursor.Pos:])
				cursor.Pos = len(cursor.Input)
			}
		}
	default:
		if cursor.Pos < len(cursor.Input) {
			value = string(cursor.Input[cursor.Pos:])
			cursor.Pos = len(cursor.Input)
		}
	}

	if key != "" {
		return key, value
	}
	if index := strings.Index(value, "="); index != -1 {
		key = value[:index]
		value = value[index+1:]
	} else {
		key = value
		value = ""
	}
	return key, value
}

func matchElement(cursor *parsly.Cursor) string {
	value := ""

	comaIndex := bytes.Index(cursor.Input[cursor.Pos:], []byte(","))
	enclosedIndex := bytes.Index(cursor.Input[cursor.Pos:], []byte("("))

	if enclosedIndex != -1 && enclosedIndex < comaIndex {
		pos := cursor.Pos
		cursor.Pos += enclosedIndex
		if match := cursor.MatchAny(enclosedMatcher); match.Code == enclosedToken {
			return string(cursor.Input[pos:cursor.Pos])
		}
		cursor.Pos = pos
	}
	match := cursor.MatchAfterOptional(whitespaceMatcher, scopeBlockMatcher, quotedMatcher, comaTerminatorMatcher)
	switch match.Code {
	case scopeBlockToken:
		value = match.Text(cursor)
		match = cursor.MatchAny(comaTerminatorMatcher)
	case quotedToken:
		value = match.Text(cursor)
		match = cursor.MatchAny(comaTerminatorMatcher)
	case comaTerminatorToken:
		value = match.Text(cursor)
		value = value[:len(value)-1] //exclude ,
	default:
		if cursor.Pos < len(cursor.Input) {
			value = string(cursor.Input[cursor.Pos:])
			cursor.Pos = len(cursor.Input)
		}
	}
	return value
}
