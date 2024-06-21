package generic_gorm

import (
	"context"
	log "github.com/sirupsen/logrus"
)

const (
	loggerCtxName = "x-logger-ctx"
)

func GetLoggerFromContext(ctx context.Context) *log.Entry {
	entry := ctx.Value(loggerCtxName)
	if logEntry, ok := entry.(*log.Entry); ok {
		return logEntry
	}

	return log.WithContext(ctx)
}

func ContextWithLogger(ctx context.Context, logEntry *log.Entry) context.Context {
	return context.WithValue(ctx, loggerCtxName, logEntry)
}
