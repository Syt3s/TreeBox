package logging

import (
	"context"
	"os"
	"path/filepath"
	"sync"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/syt3s/TreeBox/internal/branding"
)

type Options struct {
	ServiceName string
	Production  bool
	LogDir      string
}

var (
	mu     sync.RWMutex
	logger = zap.NewNop()
)

func Init(opts Options) (*zap.Logger, error) {
	if opts.ServiceName == "" {
		opts.ServiceName = branding.ServiceName
	}
	if opts.LogDir == "" {
		opts.LogDir = "logs"
	}

	if err := os.MkdirAll(opts.LogDir, 0o755); err != nil {
		return nil, err
	}

	level := zap.NewAtomicLevelAt(zap.DebugLevel)
	if opts.Production {
		level.SetLevel(zap.InfoLevel)
	}

	fileEncoderConfig := zap.NewProductionEncoderConfig()
	fileEncoderConfig.TimeKey = "timestamp"
	fileEncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	fileEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	consoleEncoderConfig := fileEncoderConfig
	consoleEncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

	appLogWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(opts.LogDir, "app.log"),
		MaxSize:    20,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	})
	errorLogWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(opts.LogDir, "error.log"),
		MaxSize:    20,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	})

	allLevels := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= level.Level()
	})
	errorLevels := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zap.ErrorLevel && lvl >= level.Level()
	})

	newLogger := zap.New(
		zapcore.NewTee(
			zapcore.NewCore(fileEncoder, appLogWriter, allLevels),
			zapcore.NewCore(fileEncoder, errorLogWriter, errorLevels),
			zapcore.NewCore(consoleEncoder, zapcore.Lock(os.Stdout), allLevels),
		),
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel),
		zap.Fields(zap.String("service", opts.ServiceName)),
	)

	mu.Lock()
	oldLogger := logger
	logger = newLogger
	mu.Unlock()

	zap.ReplaceGlobals(newLogger)
	_ = oldLogger.Sync()

	return newLogger, nil
}

func L() *zap.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return logger
}

func Sync() error {
	return L().Sync()
}

func FromContext(ctx context.Context) *zap.Logger {
	fields := TraceFields(ctx)
	if len(fields) == 0 {
		return L()
	}
	return L().With(fields...)
}

func TraceFields(ctx context.Context) []zap.Field {
	if ctx == nil {
		return nil
	}

	spanContext := trace.SpanFromContext(ctx).SpanContext()
	if !spanContext.IsValid() {
		return nil
	}

	fields := []zap.Field{
		zap.String("trace_id", spanContext.TraceID().String()),
	}
	if spanContext.SpanID().IsValid() {
		fields = append(fields, zap.String("span_id", spanContext.SpanID().String()))
	}
	return fields
}
