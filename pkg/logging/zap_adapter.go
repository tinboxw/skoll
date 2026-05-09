package logging

// NewZapCompatibleLogger returns the default logger abstraction.
// It keeps this package API ready for a future Zap-backed implementation.
func NewZapCompatibleLogger(level string) Logger {
	return New(level)
}
