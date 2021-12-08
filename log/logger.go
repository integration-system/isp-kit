package log

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Level zapcore.Level

const (
	ErrorLevel = Level(zap.ErrorLevel)
	InfoLevel  = Level(zap.InfoLevel)
	DebugLevel = Level(zap.DebugLevel)
)

type Adapter struct {
	devMode      bool
	rotation     *Rotation
	initialLevel Level

	logger *zap.Logger
	level  zap.AtomicLevel
}

func New(opts ...Option) (*Adapter, error) {
	a := &Adapter{
		devMode:      false,
		rotation:     nil,
		initialLevel: InfoLevel,
	}
	for _, opt := range opts {
		opt(a)
	}

	cfg := zap.NewProductionConfig()
	if a.devMode {
		cfg = zap.NewDevelopmentConfig()
	}
	if a.rotation != nil {
		outputUrl := rotationToUrl(*a.rotation)
		cfg.OutputPaths = append(cfg.OutputPaths, outputUrl.String())
	}
	level := zap.NewAtomicLevelAt(zapcore.Level(a.initialLevel))
	cfg.Level = level
	logger, err := cfg.Build(zap.WithCaller(false), zap.AddStacktrace(zap.FatalLevel))
	if err != nil {
		return nil, errors.Wrap(err, "build logger")
	}
	a.logger = logger
	a.level = level

	return a, nil
}

func (a *Adapter) Info(ctx context.Context, message interface{}, fields ...Field) {
	a.Log(ctx, InfoLevel, message, fields...)
}

func (a *Adapter) Error(ctx context.Context, message interface{}, fields ...Field) {
	a.Log(ctx, ErrorLevel, message, fields...)
}

func (a *Adapter) Debug(ctx context.Context, message interface{}, fields ...Field) {
	a.Log(ctx, DebugLevel, message, fields...)
}

func (a *Adapter) Log(ctx context.Context, level Level, message interface{}, fields ...Field) {
	entry := a.logger.Check(zapcore.Level(level), fmt.Sprintf("%v", message))
	if entry != nil {
		arr := append(ContextLogValues(ctx), fields...)
		zapFields := make([]zap.Field, len(arr))
		for i := range arr {
			zapFields[i] = zap.Field(arr[i])
		}
		entry.Write(zapFields...)
	}
}

func (a *Adapter) SetLevel(level Level) {
	a.level.SetLevel(zapcore.Level(level))
}

func (a *Adapter) Close() error {
	return a.logger.Sync()
}
