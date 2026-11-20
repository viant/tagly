package choice

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/francoispqt/gojay"
)

func (e *Enum) inputOrdinal(number json.Number) (json.Number, error) {
	if e.gojayInt {
		return gojayOrdinal(number)
	}
	return ordinal(number.String())
}

// DecodeJSON has already validated a single JSON number and retained its spelling.
func gojayOrdinal(number json.Number) (ordinal json.Number, err error) {
	defer func() {
		if recover() != nil {
			ordinal, err = "", fmt.Errorf("invalid numeric enum input")
		}
	}()
	var id int
	if err := gojay.Unsafe.Unmarshal([]byte(number), &id); err != nil {
		return "", fmt.Errorf("invalid numeric enum input: %w", err)
	}
	return json.Number(strconv.Itoa(id)), nil
}
