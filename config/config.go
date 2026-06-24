package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

func Load() (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func LoadFromEnvFile(path string) (*Config, error) {
	cfg := &Config{}

	if err := cleanenv.ReadConfig(path, cfg); err != nil {
		return nil, err
	}

	if err := cleanenv.UpdateEnv(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
