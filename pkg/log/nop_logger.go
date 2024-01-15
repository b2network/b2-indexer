package log

type nopLogger struct{}

// Interface assertions
var _ Logger = (*nopLogger)(nil)

// NewNopLogger returns a logger that doesn't do anything.
func NewNopLogger() Logger { return &nopLogger{} }

// info level
func (nopLogger) Info(msg string, fields ...Field)               {}
func (nopLogger) Infof(format string, v ...interface{})          {}
func (nopLogger) Infow(msg string, keysAndValues ...interface{}) {}

// debug level
func (nopLogger) Debug(msg string, fields ...Field)               {}
func (nopLogger) Debugf(format string, v ...interface{})          {}
func (nopLogger) Debugw(msg string, keysAndValues ...interface{}) {}

// warn level
func (nopLogger) Warn(msg string, fields ...Field)               {}
func (nopLogger) Warnf(format string, v ...interface{})          {}
func (nopLogger) Warnw(msg string, keysAndValues ...interface{}) {}

// error level
func (nopLogger) Error(msg string, fields ...Field)               {}
func (nopLogger) Errorf(format string, v ...interface{})          {}
func (nopLogger) Errorw(msg string, keysAndValues ...interface{}) {}
func (nopLogger) ErrorR(format string, v ...interface{}) error    { return nil }

// panic level
func (nopLogger) Panic(msg string, fields ...Field)               {}
func (nopLogger) Panicf(format string, v ...interface{})          {}
func (nopLogger) Panicw(msg string, keysAndValues ...interface{}) {}

// fatal level
func (nopLogger) Fatal(msg string, fields ...Field)               {}
func (nopLogger) Fatalf(format string, v ...interface{})          {}
func (nopLogger) Fatalw(msg string, keysAndValues ...interface{}) {}

// WithValues adds some key-value pairs of context to a logger.
func (nopLogger) WithValues(keysAndValues ...interface{}) Logger { return nopLogger{} }

// WithName adds a new element to the logger's name.
// Successive calls with WithName continue to append
// suffixes to the logger's name.  It's strongly recommended
// that name segments contain only letters, digits, and hyphens
// (see the package documentation for more information).
func (nopLogger) WithName(name string) Logger { return nopLogger{} }

// Flush calls the underlying Core's Sync method, flushing any buffered
// log entries. Applications should take care to call Sync before exiting.
func (nopLogger) Flush() {}
