package jsonx

import "fmt"

type wrapperErrorKind uint8

const (
	wrapperUnknownMember wrapperErrorKind = iota + 1
	wrapperRequiredMember
)

// WrapperError identifies a native wrapper-policy validation failure whose
// message contains only a submitted member name or the declared wrapper name.
// Its private state is created only by the policy; the zero value is not public.
// Syntax, conversion and configuration errors do not acquire this authority.
type WrapperError struct {
	kind   wrapperErrorKind
	member string
}

// PublicMessage returns the bounded validation message. Consumers must recognize
// this concrete type and retain their transport status and error-owner policy;
// an arbitrary error implementing this method does not establish provenance.
func (e *WrapperError) PublicMessage() (string, bool) {
	if e == nil {
		return "", false
	}
	switch e.kind {
	case wrapperUnknownMember:
		return fmt.Sprintf("json: unknown field %q", e.member), true
	case wrapperRequiredMember:
		return fmt.Sprintf("%s is required", e.member), true
	}
	return "", false
}

func (e *WrapperError) Error() string {
	message, ok := e.PublicMessage()
	if !ok {
		return "invalid wrapper input"
	}
	return message
}
