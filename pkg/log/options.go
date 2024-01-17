package log

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap/zapcore"
)

const (
	consoleFormat = "console"
	jsonFormat    = "json"
)

var validLevels = []string{"debug", "info", "warn", "error", "panic", "fatal"}

// options for create log
type Options struct {
	// destination of log
	OutputPaths []string `json:"output-paths"`
	// the error log of zap
	ErrorOutputPaths []string `json:"error-output-paths"`
	// log level  "info debug warn error panic fatal"
	Level string `json:"level"`
	// json or console
	Format string `json:"format"`
	// show file, function, line number in log
	DisableCaller     bool `json:"disable-caller"`
	DisableStacktrace bool `json:"disable-stacktrace"`
	// console format can enable color
	EnableColor bool   `json:"enable-color"`
	Development bool   `json:"development"`
	Name        string `json:"name"`
}

func NewOptions() *Options {
	return &Options{
		Level:             zapcore.InfoLevel.String(),
		DisableCaller:     false,
		DisableStacktrace: false,
		Format:            consoleFormat,
		EnableColor:       false,
		Development:       false,
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
	}
}

func (opt *Options) Validate() error {
	errorMsg := make([]string, 0)
	if opt.Format != consoleFormat && opt.Format != jsonFormat {
		errorMsg = append(errorMsg, fmt.Sprintf("Invalid format, should be: %v or %v", consoleFormat, jsonFormat))
	}
	valid := false
	for _, val := range validLevels {
		if val == opt.Level {
			valid = true
			break
		}
	}
	if !valid {
		errorMsg = append(errorMsg, fmt.Sprintf("Invalid level, should be oneof %v", strings.Join(validLevels, ",")))
	}
	if len(errorMsg) != 0 {
		return errors.New(strings.Join(validLevels, ";"))
	}
	return nil
}
