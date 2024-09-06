package crdbx

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
)

// Cursor handles encoding and decoding a pagination cursor values.
type Cursor string

// Values parses the cursor returning the values.
func (c *Cursor) Values() (url.Values, error) {
	if c == nil || *c == "" {
		return nil, nil
	}

	cursor, err := base64.URLEncoding.DecodeString(string(*c))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPaginationCursor, err)
	}

	values, err := url.ParseQuery(string(cursor))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidPaginationCursor, err)
	}

	return values, nil
}

// Set lets you define include a field in the cursor.
func (c *Cursor) Set(key, value string) error {
	var (
		values = url.Values{}
		err    error
	)

	if c != nil {
		values, err = c.Values()
		if err != nil {
			return err
		}
	}

	values.Set(key, value)

	cursor, err := NewCursorFromValues(values)
	if err != nil {
		return err
	}

	*c = *cursor

	return nil
}

// String returns a string value of the cursor.
func (c *Cursor) String() string {
	if c == nil {
		return ""
	}

	return string(*c)
}

// StringPtr returns a pointer to the string value of the cursor.
func (c *Cursor) StringPtr() *string {
	if c == nil {
		return nil
	}

	v := c.String()

	return &v
}

// MarshalJSON implements json marshaller.
func (c *Cursor) MarshalJSON() ([]byte, error) {
	if c == nil || *c == "" {
		return []byte("null"), nil
	}

	return json.Marshal(string(*c))
}

// NewCursor creates a new pagination cursor from key/value pairs.
func NewCursor(kvpairs ...string) (*Cursor, error) {
	if len(kvpairs) == 0 {
		return nil, nil
	}

	if len(kvpairs)%2 != 0 {
		return nil, fmt.Errorf("%w: incorrect key/value pairs", ErrInvalidPaginationCursor)
	}

	values := url.Values{}

	for i := 0; i < len(kvpairs); i += 2 {
		if kvpairs[i+1] != "" {
			values.Set(kvpairs[i], kvpairs[i+1])
		}
	}

	return NewCursorFromValues(values)
}

// NewCursorFromValues creates a new pagination cursor from url values.
func NewCursorFromValues(values url.Values) (*Cursor, error) {
	if values == nil {
		return nil, nil
	}

	cursor := Cursor(base64.URLEncoding.EncodeToString([]byte(values.Encode())))

	return &cursor, nil
}
