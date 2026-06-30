//go:build integration

package postgres_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/database/postgres"
)

func ExampleNewPGXPool() {
	cfg, _ := config.Load()

	pool, err := postgres.NewPGXPool(cfg.Postgres, zap.NewNop())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer pool.Close()

	err = pool.Ping(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Release()

	var val string
	err = conn.QueryRow(context.Background(), "SELECT $1", "lala").Scan(&val)
	fmt.Println(val, err)

	// Output:
	// lala <nil>
}

func ExampleNewPGXPool_read_write() {
	cfg, _ := config.Load()
	// use master
	cfg.Postgres.TargetSessionAttrs = "primary"

	pool, _ := postgres.NewPGXPool(cfg.Postgres, zap.NewNop())
	defer pool.Close()

	conn, _ := pool.Acquire(context.Background())
	defer conn.Release()

	var val string
	var isInRecovery bool
	err := conn.QueryRow(context.Background(), "SELECT $1, pg_is_in_recovery();", "lala").Scan(&val, &isInRecovery)
	fmt.Println(val, isInRecovery, err)

	// Output:
	// lala false <nil>
}

func ExampleNewPGXPool_read_only() {
	cfg, _ := config.Load()
	// use replica
	cfg.Postgres.TargetSessionAttrs = "standby"

	pool, _ := postgres.NewPGXPool(cfg.Postgres, zap.NewNop())
	defer pool.Close()

	conn, _ := pool.Acquire(context.Background())
	defer conn.Release()

	var val string
	var isInRecovery bool
	err := conn.QueryRow(context.Background(), "SELECT $1, pg_is_in_recovery();", "lala").Scan(&val, &isInRecovery)
	fmt.Println(val, isInRecovery, err)

	// Output:
	// lala true <nil>
}

func ExampleNewPGXPool_prefer_replica() {
	cfg, _ := config.Load()
	// use replica if available, can fallback to master
	cfg.Postgres.TargetSessionAttrs = "prefer-standby"

	pool, _ := postgres.NewPGXPool(cfg.Postgres, zap.NewNop())
	defer pool.Close()

	conn, _ := pool.Acquire(context.Background())
	defer conn.Release()

	var val string
	var isInRecovery bool
	err := conn.QueryRow(context.Background(), "SELECT $1, pg_is_in_recovery();", "lala").Scan(&val, &isInRecovery)
	fmt.Println(val, isInRecovery, err)

	// Output:
	// lala true <nil>
}

func ExampleNewPGXPool_log_slow_query() {
	cfg, _ := config.Load()
	cfg.Postgres.SlowQueryThreshold = 1 * time.Nanosecond

	pool, _ := postgres.NewPGXPool(cfg.Postgres, newExampleLogger())
	defer pool.Close()

	conn, _ := pool.Acquire(context.Background())
	defer conn.Release()

	var val string
	var isInRecovery bool
	err := conn.QueryRow(context.Background(), "SELECT $1, pg_is_in_recovery();", "lala").Scan(&val, &isInRecovery)
	fmt.Println(val, isInRecovery, err)

	// Output:
	// {"level":"warn","msg":"slow query","duration":"<elapsed>","sql":"SELECT $1, pg_is_in_recovery();"}
	// lala false <nil>
}

func newExampleLogger() *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("<elapsed>")
		},
	}
	return zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		os.Stdout,
		zapcore.WarnLevel,
	))
}
