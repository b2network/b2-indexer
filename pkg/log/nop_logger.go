package log

type nopLogger struct{}

// Interface assertions
var _ Logger = (*nopLogger)(nil)

// NewNopLogger returns a logger that doesn't do anything.
func NewNopLogger() Logger { return &nopLogger{} }

// info level
func (nopLogger) Info(_ string, _ ...Field)        {}
func (nopLogger) Infof(_ string, _ ...interface{}) {}
func (nopLogger) Infow(_ string, _ ...interface{}) {}

// debug level
func (nopLogger) Debug(_ string, _ ...Field)        {}
func (nopLogger) Debugf(_ string, _ ...interface{}) {}
func (nopLogger) Debugw(_ string, _ ...interface{}) {}

// warn level
func (nopLogger) Warn(_ string, _ ...Field)        {}
func (nopLogger) Warnf(_ string, _ ...interface{}) {}
func (nopLogger) Warnw(_ string, _ ...interface{}) {}

// error level
func (nopLogger) Error(_ string, _ ...Field)              {}
func (nopLogger) Errorf(_ string, _ ...interface{})       {}
func (nopLogger) Errorw(_ string, _ ...interface{})       {}
func (nopLogger) ErrorR(_ string, _ ...interface{}) error { return nil }

// panic level
func (nopLogger) Panic(_ string, _ ...Field)        {}
func (nopLogger) Panicf(_ string, _ ...interface{}) {}
func (nopLogger) Panicw(_ string, _ ...interface{}) {}

// fatal level
func (nopLogger) Fatal(_ string, _ ...Field)        {}
func (nopLogger) Fatalf(_ string, _ ...interface{}) {}
func (nopLogger) Fatalw(_ string, _ ...interface{}) {}

// WithValues adds some key-value pairs of context to a logger.
func (nopLogger) WithValues(_ ...interface{}) Logger { return nopLogger{} }

// WithName adds a new element to the logger's name.
// Successive calls with WithName continue to append
// suffixes to the logger's name.  It's strongly recommended
// that name segments contain only letters, digits, and hyphens
// (see the package documentation for more in_ion).
func (nopLogger) WithName(name string) Logger { return nopLogger{} }

// Flush calls the underlying Core's Sync method, flushing any buffered
// log entries. Applications should take care to call Sync before exiting.
func (nopLogger) Flush() {}
