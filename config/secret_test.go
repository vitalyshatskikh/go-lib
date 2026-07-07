package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecretStr_String(t *testing.T) {
	tests := []struct {
		name   string
		secret SecretStr
		want   string
	}{
		{
			name:   "non-empty returns masked",
			secret: SecretStr("s3cret!"),
			want:   "xxxxxx",
		},
		{
			name:   "empty returns empty",
			secret: SecretStr(""),
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.secret.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSecretStr_Value(t *testing.T) {
	tests := []struct {
		name   string
		secret SecretStr
		want   string
	}{
		{
			name:   "non-empty returns original",
			secret: SecretStr("s3cret!"),
			want:   "s3cret!",
		},
		{
			name:   "empty returns empty",
			secret: SecretStr(""),
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.secret.SecretValue()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSecretStr_MarshalJSON(t *testing.T) {
	tests := []struct {
		name   string
		secret SecretStr
		want   string
	}{
		{
			name:   "non-empty returns masked quoted",
			secret: SecretStr("s3cret!"),
			want:   `"xxxxxx"`,
		},
		{
			name:   "empty returns empty quoted",
			secret: SecretStr(""),
			want:   `""`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.secret)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}

func TestSecretStr_MarshalText(t *testing.T) {
	tests := []struct {
		name   string
		secret SecretStr
		want   string
	}{
		{
			name:   "non-empty returns masked",
			secret: SecretStr("s3cret!"),
			want:   "xxxxxx",
		},
		{
			name:   "empty returns empty",
			secret: SecretStr(""),
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.secret.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestSecretURL_String_WhenHasUserInfo_ThenMasks(t *testing.T) {
	tests := []struct {
		name string
		url  SecretURL
		want string
	}{
		{
			name: "user and password",
			url:  SecretURL("postgres://user:pass@localhost:5432/db"),
			want: "postgres://xxxxxx:xxxxxx@localhost:5432/db",
		},
		{
			name: "user only",
			url:  SecretURL("https://token@example.com/api"),
			want: "https://xxxxxx:xxxxxx@example.com/api",
		},
		{
			name: "no credentials",
			url:  SecretURL("https://example.com/api"),
			want: "https://example.com/api",
		},
		{
			name: "empty",
			url:  SecretURL(""),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.url.String()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSecretURL_String_WhenInvalidURL_ThenReturnsMasked(t *testing.T) {
	url := SecretURL("not a :// valid url")
	assert.Equal(t, "xxxxxx", url.String())
}

func TestSecretURL_Value(t *testing.T) {
	tests := []struct {
		name string
		url  SecretURL
		want string
	}{
		{
			name: "non-empty returns original",
			url:  SecretURL("postgres://user:pass@localhost:5432/db"),
			want: "postgres://user:pass@localhost:5432/db",
		},
		{
			name: "empty returns empty",
			url:  SecretURL(""),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.url.SecretValue()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSecretURL_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		url  SecretURL
		want string
	}{
		{
			name: "with credentials returns masked",
			url:  SecretURL("https://user:pass@example.com"),
			want: `"https://xxxxxx:xxxxxx@example.com"`,
		},
		{
			name: "without credentials returns unchanged",
			url:  SecretURL("https://example.com"),
			want: `"https://example.com"`,
		},
		{
			name: "empty returns empty",
			url:  SecretURL(""),
			want: `""`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.url)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, string(got))
		})
	}
}

func TestSecretURL_MarshalText(t *testing.T) {
	tests := []struct {
		name string
		url  SecretURL
		want string
	}{
		{
			name: "with credentials returns masked",
			url:  SecretURL("https://user:pass@example.com"),
			want: "https://xxxxxx:xxxxxx@example.com",
		},
		{
			name: "without credentials returns unchanged",
			url:  SecretURL("https://example.com"),
			want: "https://example.com",
		},
		{
			name: "empty returns empty",
			url:  SecretURL(""),
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.url.MarshalText()
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}
