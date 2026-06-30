package restapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/vitalyshatskikh/go-lib/config"
)

func Example() {
	logger := newExampleLogger()

	cfg, err := config.Load()
	if err != nil {
		fmt.Println(err)
		return
	}
	cfg.API.Host = "localhost"
	cfg.API.Port = 18080

	exampleSpec := `{"openapi":"3.0.0","info":{"title":"Test"}}`

	srv, err := New(cfg, WithLogger(logger), WithOpenAPI(strings.NewReader(exampleSpec)))
	if err != nil {
		fmt.Println(err)
		return
	}

	wg := sync.WaitGroup{}
	wg.Go(func() {
		if err := srv.Start(); err != nil && errors.Is(err, http.ErrServerClosed) {
			fmt.Println(err)
		}
	})

	_, err = http.Get("http://localhost:18080/ping")
	if err != nil {
		fmt.Println(err)
		return
	}

	_, err = http.Get("http://localhost:18080/docs")
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = srv.Shutdown(ctx)
	wg.Wait()

	_ = logger.Sync()
	// Output:
	// {"level":"info","msg":"enabling openapi endpoint"}
	// {"level":"info","msg":"starting api server","addr":"localhost:18080"}
	// {"level":"info","msg":"GET /docs","time":"<time>","latency":"<elapsed>","remote_ip":"","host":"localhost:18080","method":"GET","uri":"/docs","status":200,"size":"<size>","user_agent":"Go-http-client/1.1","trace_id":""}
	// {"level":"info","msg":"shutting down servers"}
	// {"level":"info","msg":"servers shut down successfully"}
}

func newExampleLogger() *zap.Logger {
	encoderCfg := zapcore.EncoderConfig{
		MessageKey:  "msg",
		LevelKey:    "level",
		EncodeLevel: zapcore.LowercaseLevelEncoder,
		EncodeTime: func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("<time>")
		},
		EncodeDuration: func(d time.Duration, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString("<elapsed>")
		},
	}
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		os.Stdout,
		zapcore.InfoLevel,
	)
	return zap.New(&maskingCore{Core: core})
}

type maskingCore struct {
	zapcore.Core
}

func (c *maskingCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	for i, f := range fields {
		if f.Key == "size" {
			fields[i] = zap.String("size", "<size>")
		}
	}
	return c.Core.Write(entry, fields)
}

func (c *maskingCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

func (c *maskingCore) With(fields []zapcore.Field) zapcore.Core {
	return &maskingCore{Core: c.Core.With(fields)}
}
