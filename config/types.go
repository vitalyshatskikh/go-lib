package config

import (
	"net"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	App       AppConfig       `env-prefix:"APP_"`
	API       APIConfig       `env-prefix:"API_"`
	Metrics   MetricsConfig   `env-prefix:"METRICS_"`
	Logging   LoggingConfig   `env-prefix:"LOGGING_"`
	Telemetry TelemetryConfig `env-prefix:"TELEMETRY_"`
	Sentry    SentryConfig    `env-prefix:"SENTRY_"`
	Debug     bool            `env:"DEBUG" env-default:"false"`

	Postgres PostgresConfig `env-prefix:"POSTGRES_"`
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
	Level     string `env:"LEVEL" env-default:"info"`
	AddCaller bool   `env:"ADD_CALLER" env-default:"false"`
}

type TelemetryConfig struct {
	Enabled         bool    `env:"ENABLED" env-default:"false"`
	TracingEndpoint string  `env:"TRACING_ENDPOINT" env-default:"localhost:4317"`
	SampleRate      float64 `env:"SAMPLE_RATE" env-default:"1.0"`
}

type SentryConfig struct {
	DSN           SecretURL     `env:"DSN" env-default:""`
	Levels        []string      `env:"LEVELS" env-default:"warn,error"`
	SampleRate    float64       `env:"SAMPLE_RATE" env-default:"1.0"`
	FlushTimeout  time.Duration `env:"FLUSH_TIMEOUT" env-default:"5s"`
	EnableTracing bool          `env:"ENABLE_TRACING" env-default:"false"`
	// Debug Sentry SDK
	Debug bool `env:"DEBUG" env-default:"false"`
}

type PostgresConfig struct {
	DSN SecretStr `env:"DSN" env-default:""`

	Hosts    []string  `env:"HOSTS" env-default:"localhost:15432,localhost:15433"`
	User     string    `env:"USER" env-default:"postgres"`
	Password SecretStr `env:"PASSWORD" env-default:"postgres"`
	Database string    `env:"DATABASE" env-default:"postgres"`
	// SSLMode
	//
	// Values:
	//   - "disable" - only try a non-SSL connection
	//   - "allow" - first try a non-SSL connection; if that fails, try an SSL connection
	//   - "prefer" - first try an SSL connection; if that fails, try a non-SSL connection
	//   - "require" - only try an SSL connection. If a root CA file is present, verify the certificate in the same way as if verify-ca was specified
	//   - "verify-ca" - only try an SSL connection, and verify that the server certificate is issued by a trusted certificate authority (CA)
	//   - "verify-full" - only try an SSL connection, verify that the server certificate is issued by a trusted CA and that the requested server host name matches that in the certificate
	SSLMode string `env:"SSLMODE" env-default:"prefer"`
	// TargetSessionAttrs used to specify the required state of a server before a connection is established.
	//
	// Useful values:
	//   - "primary" - server must not be in hot standby mode (master)
	//   - "standby" - server must be in hot standby mode (replica)
	//   - "prefer-standby" - first try to find a standby server, but if none of the listed hosts is a standby server, try again in any mode
	//
	// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNECT-TARGET-SESSION-ATTRS
	TargetSessionAttrs string `env:"TARGET_SESSION_ATTRS" env-default:"primary"`

	// Pool options

	MaxConns          int32         `env:"MAX_CONNS" env-default:"10"`
	MinConns          int32         `env:"MIN_CONNS" env-default:"0"`
	MaxConnLifetime   time.Duration `env:"MAX_CONN_LIFETIME" env-default:"1h"`
	MaxConnIdleTime   time.Duration `env:"MAX_CONN_IDLE_TIME" env-default:"30m"`
	HealthCheckPeriod time.Duration `env:"HEALTH_CHECK_PERIOD" env-default:"1m"`

	// Timeout options

	PingTimeout    time.Duration `env:"PING_TIMEOUT" env-default:"30s"`
	ConnectTimeout time.Duration `env:"CONNECT_TIMEOUT" env-default:"30s"`

	// Observability options

	PoolName           string        `env:"POOL_NAME" env-default:""`
	SlowQueryThreshold time.Duration `env:"SLOW_QUERY_THRESHOLD" env-default:"0"`
}

func (c *PostgresConfig) ConnString() string {
	if c.DSN != "" {
		return c.DSN.SecretValue()
	}

	hosts := c.Hosts
	if len(hosts) == 0 {
		hosts = []string{"localhost:5432"}
	}

	hostPorts := make([]string, len(hosts))
	for i, host := range hosts {
		h, p, err := net.SplitHostPort(host)
		if err != nil {
			hostPorts[i] = net.JoinHostPort(host, "5432")
		} else {
			hostPorts[i] = net.JoinHostPort(h, p)
		}
	}

	u := &url.URL{
		Scheme: "postgres",
		Host:   strings.Join(hostPorts, ","),
		Path:   c.Database,
	}
	if c.User != "" {
		if c.Password != "" {
			u.User = url.UserPassword(c.User, c.Password.SecretValue())
		} else {
			u.User = url.User(c.User)
		}
	}

	q := u.Query()
	q.Set("sslmode", c.SSLMode)
	q.Set("target_session_attrs", c.TargetSessionAttrs)
	u.RawQuery = q.Encode()

	return u.String()
}
