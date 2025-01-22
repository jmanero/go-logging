package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Level extends zapcore.Level with pflag.Flag methods
type Level struct {
	zap.AtomicLevel
}

// NewLevel instantiates a new Level register for an initial zapcore.Level
func NewLevel(lvl zapcore.Level) *Level {
	return &Level{
		AtomicLevel: zap.NewAtomicLevelAt(lvl),
	}
}

// Set implements the pflag.Flag setter
func (lvl *Level) Set(val string) error {
	l, err := zapcore.ParseLevel(val)
	if err != nil {
		return err
	}

	lvl.SetLevel(l)
	return nil
}

// Type implements the pflag.Flag interface for usage printing
func (*Level) Type() string {
	return "zap.Level"
}
