package postgres

import (
	"context"
	"time"

	"github.com/exaring/otelpgx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	_ pgx.QueryTracer    = (*slowQueryTracer)(nil)
	_ pgx.BatchTracer    = (*slowQueryTracer)(nil)
	_ pgx.CopyFromTracer = (*slowQueryTracer)(nil)
	_ pgx.PrepareTracer  = (*slowQueryTracer)(nil)
	_ pgx.ConnectTracer  = (*slowQueryTracer)(nil)

	_ pgxpool.AcquireTracer = (*slowQueryTracer)(nil)
)

type slowQueryTracer struct {
	inner     *otelpgx.Tracer
	threshold time.Duration
	logger    *zap.Logger
}

type tracerCtxKey struct{}

type tracerInfo struct {
	start time.Time
	sql   string
}

// QueryTracer

func (t *slowQueryTracer) TraceQueryStart(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData,
) context.Context {
	ctx = context.WithValue(ctx, tracerCtxKey{}, tracerInfo{
		start: time.Now(),
		sql:   data.SQL,
	})
	return t.inner.TraceQueryStart(ctx, conn, data)
}

func (t *slowQueryTracer) TraceQueryEnd(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData,
) {
	t.inner.TraceQueryEnd(ctx, conn, data)
	t.maybeLog(ctx, "slow query", data.Err,
		func(info tracerInfo) []zap.Field {
			return []zap.Field{zap.String("sql", info.sql)}
		},
	)
}

// BatchTracer

func (t *slowQueryTracer) TraceBatchStart(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchStartData,
) context.Context {
	ctx = context.WithValue(ctx, tracerCtxKey{}, tracerInfo{start: time.Now()})
	return t.inner.TraceBatchStart(ctx, conn, data)
}

func (t *slowQueryTracer) TraceBatchQuery(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchQueryData,
) {
	t.inner.TraceBatchQuery(ctx, conn, data)
}

func (t *slowQueryTracer) TraceBatchEnd(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceBatchEndData,
) {
	t.inner.TraceBatchEnd(ctx, conn, data)
	t.maybeLog(ctx, "slow batch", data.Err, nil)
}

// CopyFromTracer

func (t *slowQueryTracer) TraceCopyFromStart(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromStartData,
) context.Context {
	ctx = context.WithValue(ctx, tracerCtxKey{}, tracerInfo{start: time.Now()})
	return t.inner.TraceCopyFromStart(ctx, conn, data)
}

func (t *slowQueryTracer) TraceCopyFromEnd(
	ctx context.Context, conn *pgx.Conn, data pgx.TraceCopyFromEndData,
) {
	t.inner.TraceCopyFromEnd(ctx, conn, data)
	t.maybeLog(ctx, "slow copy", data.Err, nil)
}

// PrepareTracer

func (t *slowQueryTracer) TracePrepareStart(
	ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareStartData,
) context.Context {
	return t.inner.TracePrepareStart(ctx, conn, data)
}

func (t *slowQueryTracer) TracePrepareEnd(
	ctx context.Context, conn *pgx.Conn, data pgx.TracePrepareEndData,
) {
	t.inner.TracePrepareEnd(ctx, conn, data)
}

// ConnectTracer

func (t *slowQueryTracer) TraceConnectStart(
	ctx context.Context, data pgx.TraceConnectStartData,
) context.Context {
	return t.inner.TraceConnectStart(ctx, data)
}

func (t *slowQueryTracer) TraceConnectEnd(
	ctx context.Context, data pgx.TraceConnectEndData,
) {
	t.inner.TraceConnectEnd(ctx, data)
}

// AcquireTracer

func (t *slowQueryTracer) TraceAcquireStart(
	ctx context.Context, pool *pgxpool.Pool, data pgxpool.TraceAcquireStartData,
) context.Context {
	return t.inner.TraceAcquireStart(ctx, pool, data)
}

func (t *slowQueryTracer) TraceAcquireEnd(
	ctx context.Context, pool *pgxpool.Pool, data pgxpool.TraceAcquireEndData,
) {
	t.inner.TraceAcquireEnd(ctx, pool, data)
}

func (t *slowQueryTracer) maybeLog(
	ctx context.Context, msg string, err error, extra func(tracerInfo) []zap.Field,
) {
	if t.threshold <= 0 {
		return
	}

	info, ok := ctx.Value(tracerCtxKey{}).(tracerInfo)
	if !ok {
		return
	}

	if d := time.Since(info.start); d >= t.threshold {
		fields := []zap.Field{
			zap.Duration("duration", d),
			zap.Error(err),
		}
		if extra != nil {
			fields = append(fields, extra(info)...)
		}
		t.logger.Warn(msg, fields...)
	}
}
