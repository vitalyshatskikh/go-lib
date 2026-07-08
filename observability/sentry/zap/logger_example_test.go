package zap

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/vitalyshatskikh/go-lib/config"
	"github.com/vitalyshatskikh/go-lib/observability/sentry"
	sentrymock "github.com/vitalyshatskikh/go-lib/observability/sentry/mock"
)

func ExampleWrapLogger() {
	ms, stopMock := sentrymock.RunMockSentry("testkey", "testproject")
	defer stopMock()

	logger := zap.NewExample()
	cfg := &config.Config{
		Sentry: config.SentryConfig{
			DSN: config.SecretURL(ms.DSN),
		},
	}

	stopSentry, err := sentry.InitSentry(context.Background(), cfg, zap.NewNop())
	if err != nil {
		fmt.Println(err)
		return
	}

	logger.Error("first error")

	logger2 := WrapLogger(cfg, logger)
	logger2.Error("second error")

	_ = logger2.Sync()
	_ = stopSentry(context.Background())
	fmt.Println("sentry called once:", ms.Calls)

	// Output:
	// {"level":"error","msg":"first error"}
	// {"level":"error","msg":"second error"}
	// sentry called once: [POST /api/testproject/envelope/]
}

func ExampleWrapLogger_empty_dsn() {
	ms, stopMock := sentrymock.RunMockSentry("testkey", "testproject")
	defer stopMock()

	logger := zap.NewExample()
	cfg := &config.Config{}

	stopSentry, err := sentry.InitSentry(context.Background(), cfg, zap.NewNop())
	if err != nil {
		fmt.Println(err)
		return
	}

	logger.Error("first error")

	logger2 := WrapLogger(cfg, logger)
	logger2.Error("second error")

	_ = logger2.Sync()
	_ = stopSentry(context.Background())
	fmt.Println("sentry not called:", ms.Calls)

	// Output:
	// {"level":"error","msg":"first error"}
	// {"level":"error","msg":"second error"}
	// sentry not called: []
}
