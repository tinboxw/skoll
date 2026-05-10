package logging

// NewZapCompatibleLogger returns a Zap-backed logger abstraction.
func NewZapCompatibleLogger(level string) Logger {
	return newZapLogger(level)
}
