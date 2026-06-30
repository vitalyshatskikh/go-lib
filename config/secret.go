package config

import "encoding/json"

// SecretStr is a string type that masks its value in String() and
// MarshalJSON/MarshalText to prevent accidental leakage in logs,
// error messages, or serialization. Use Value() to access the actual string.
type SecretStr string

func (s SecretStr) String() string {
	if s == "" {
		return ""
	}
	return "******"
}

func (s SecretStr) Value() string {
	return string(s)
}

func (s SecretStr) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s SecretStr) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}
