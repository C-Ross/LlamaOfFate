package engine

import (
	"encoding/json"
	"fmt"
	"strconv"
)

// FlexInt is an int that tolerates both JSON numbers (3) and JSON strings ("3")
// during unmarshaling. LLMs sometimes return numeric values as quoted strings.
type FlexInt int

// UnmarshalJSON implements json.Unmarshaler, accepting both 3 and "3".
func (fi *FlexInt) UnmarshalJSON(data []byte) error {
	// Try number first (most common case)
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*fi = FlexInt(n)
		return nil
	}

	// Try quoted string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("FlexInt: cannot convert %q to int: %w", s, err)
		}
		*fi = FlexInt(n)
		return nil
	}

	return fmt.Errorf("FlexInt: cannot unmarshal %s", string(data))
}

// MarshalJSON implements json.Marshaler, always encoding as a JSON number.
func (fi FlexInt) MarshalJSON() ([]byte, error) {
	return json.Marshal(int(fi))
}

// Int returns the underlying int value.
func (fi FlexInt) Int() int {
	return int(fi)
}
