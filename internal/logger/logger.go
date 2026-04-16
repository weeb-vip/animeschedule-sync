package logger

import (
	"context"
	"sync"

	"go.uber.org/zap"
)

type ctxKey struct{}

var once sync.Once
var logger *zap.Logger

func Get() *zap.Logger {
	once.Do(func() {
		var err error
		logger, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	})
	return logger
}

func FromCtx(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	} else if l := logger; l != nil {
		return l
	}
	return zap.NewNop()
}

func WithCtx(ctx context.Context, l *zap.Logger) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		if lp == l {
			return ctx
		}
	}
	return context.WithValue(ctx, ctxKey{}, l)
}
