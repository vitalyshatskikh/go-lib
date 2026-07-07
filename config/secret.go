package config

import (
	"encoding/json"
	"net/url"
)

const Mask = "xxxxxx"

// SecretStr is a string type that masks its value in String() and
// MarshalJSON/MarshalText to prevent accidental leakage in logs,
// error messages, or serialization. Use SecretValue() to access the actual string.
type SecretStr string

func (s SecretStr) String() string {
	if s == "" {
		return ""
	}
	return Mask
}

func (s SecretStr) SecretValue() string {
	return string(s)
}

func (s SecretStr) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s SecretStr) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// SecretURL is a string type that masks the userinfo part (user:password)
// of a URL in String(), MarshalJSON(), and MarshalText() to prevent
// accidental leakage in logs, error messages, or serialization.
// Use SecretValue() to access the actual URL string.
type SecretURL string

func (u SecretURL) String() string {
	if u == "" {
		return ""
	}
	parsed, err := url.Parse(string(u))
	if err != nil {
		return Mask
	}
	if parsed.User != nil {
		parsed.User = url.UserPassword(Mask, Mask)
	}
	return parsed.String()
}

func (u SecretURL) SecretValue() string {
	return string(u)
}

func (u SecretURL) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.String())
}

func (u SecretURL) MarshalText() ([]byte, error) {
	return []byte(u.String()), nil
}
