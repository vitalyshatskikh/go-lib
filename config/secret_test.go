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
			want:   "******",
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
			got := tt.secret.Value()
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
			want:   `"******"`,
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
			want:   "******",
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
