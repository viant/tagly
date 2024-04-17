package tags

import (
	"github.com/viant/parsly"
	"github.com/viant/parsly/matcher"
)

const (
	whitespaceToken = iota
	comaTerminatorToken
	scopeBlockToken
	eqTerminatorToken
	enclosedToken
	quotedToken
)

var (
	whitespaceMatcher     = parsly.NewToken(whitespaceToken, " ", matcher.NewWhiteSpace())
	comaTerminatorMatcher = parsly.NewToken(comaTerminatorToken, "coma", matcher.NewTerminator(',', true))
	eqTerminatorMatcher   = parsly.NewToken(eqTerminatorToken, "eq", matcher.NewTerminator('=', true))

	scopeBlockMatcher = parsly.NewToken(scopeBlockToken, "{ .... }", matcher.NewBlock('{', '}', '\\'))
	enclosedMatcher   = parsly.NewToken(enclosedToken, "( .... )", matcher.NewBlock('(', ')', '\\'))

	quotedMatcher = parsly.NewToken(quotedToken, "' .... '", matcher.NewQuote('\'', '\\'))
)
