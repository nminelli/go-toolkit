// Package log uses ZAP fmt, which is a small wrapper around [Uber log package](https://godoc.org/go.uber.org/zap).
package log

import (
	"os"

	"go.uber.org/zap"

	"github.com/nminelli/go-toolkit/telemetry/log/otelzap"
)

const _logLevelEnvName = "LOG_LEVEL"

// NewLogger is a reasonable production logging configuration.
// Logging is enabled at given level and above. The level can be later
// adjusted dynamically in runtime by calling SetLevel method.
//
// The logger writes directly into the OpenTelemetry exporter and does not
// support writing to a file or any other output.
//
// Stacktraces are automatically included on logs of ErrorLevel and above.
func NewLogger(opts ...Option) Logger {
	opts = append(_defaultOption, opts...)

	var cfg logConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	var zapOptions []zap.Option
	if cfg.caller {
		zapOptions = append(zapOptions, zap.AddCaller(), zap.AddCallerSkip(1))
	}

	if cfg.stacktrace {
		zapOptions = append(zapOptions, zap.AddStacktrace(zap.ErrorLevel))
	}

	logLevel := ParseLevel(os.Getenv(_logLevelEnvName))
	core := otelzap.NewOtelCore(LP(), otelzap.WithLevel(logLevel))
	l := zap.New(core, zapOptions...)
	zap.ReplaceGlobals(l)

	return &logger{
		Logger: otelzap.L(),
	}
}

// logger provides a fast, leveled, structured logging. All methods are safe
// for concurrent use.
//
// The logger is designed for contexts in which every microsecond and every
// allocation matters, so its API intentionally favors performance and type
// safety over brevity. For most applications, the SugaredLogger strikes a
// better balance between performance and ergonomics.
type logger struct {
	*otelzap.Logger
}

var _ Logger = (*logger)(nil)

// With creates a child logger and adds structured context to it. Fields added
// to the child don't affect the parent, and vice versa.
func (l *logger) With(fields ...Field) Logger {
	child := l.Logger.With(fields...)
	return &logger{
		Logger: child,
	}
}

type logConfig struct {
	caller     bool
	stacktrace bool
}

// Option configures a Logger.
type Option func(s *logConfig)

// WithCaller lets the caller configure whether to include a "caller" tag in the
// log specifying the package/file:line in which the log occurred.
//
// Default value is "true", take into consideration that in order to obtain the
// caller value reflection is used, which has a runtime cost.
func WithCaller(t bool) Option {
	return func(s *logConfig) {
		s.caller = t
	}
}

// StacktraceOnError lets the caller configure whether to include a stacktrace
// on "Error" or higher log levels.
//
// Default value is "true", take into consideration that in order to obtain the
// stacktrace, reflection is used, which has a non-trivial runtime cost.
func StacktraceOnError(t bool) Option {
	return func(s *logConfig) {
		s.stacktrace = t
	}
}

// Default options used when constructing a logger.
var _defaultOption = []Option{
	StacktraceOnError(true),
	WithCaller(true),
}
