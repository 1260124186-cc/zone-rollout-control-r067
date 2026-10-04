package platform

import (
	"encoding/base64"
	"errors"
	"strings"
)

var ErrInvalidCursor = errors.New("invalid cursor")

func EncodeCursor(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func DecodeCursor(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return "", ErrInvalidCursor
	}
	return string(decoded), nil
}
