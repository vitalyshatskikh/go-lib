package config

type Config struct {
	App       AppConfig       `env-prefix:"APP_"`
	API       APIConfig       `env-prefix:"API_"`
	Metrics   MetricsConfig   `env-prefix:"METRICS_"`
	Logging   LoggingConfig   `env-prefix:"LOGGING_"`
	Telemetry TelemetryConfig `env-prefix:"TELEMETRY_"`
	Debug     bool            `env:"DEBUG" env-default:"false"`
}

type AppConfig struct {
	Name        string `env:"NAME" env-default:"my-app"`
	Version     string `env:"VERSION" env-default:"0.1.0"`
	Environment string `env:"ENVIRONMENT" env-default:"development"`
}

type APIConfig struct {
	Host string `env:"HOST" env-default:"0.0.0.0"`
	Port int    `env:"PORT" env-default:"8080"`
}

type MetricsConfig struct {
	Enabled bool   `env:"ENABLED" env-default:"true"`
	Host    string `env:"HOST" env-default:"0.0.0.0"`
	Port    int    `env:"PORT" env-default:"8081"`
	Path    string `env:"PATH" env-default:"/metrics"`
}

type LoggingConfig struct {
	Level string `env:"LEVEL" env-default:"info"`
}

type TelemetryConfig struct {
	Enabled         bool    `env:"ENABLED" env-default:"false"`
	ServiceName     string  `env:"SERVICE_NAME" env-default:"my-app"`
	TracingEndpoint string  `env:"TRACING_ENDPOINT" env-default:"localhost:4317"`
	SampleRate      float64 `env:"SAMPLE_RATE" env-default:"1.0"`
}
