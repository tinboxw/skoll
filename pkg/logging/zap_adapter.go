package logging

// NewZapCompatibleLogger returns a Zap-backed logger abstraction.
func NewZapCompatibleLogger(level string) Logger {
	return newZapLogger(Options{Level: level, Dir: defaultOutput.Dir, File: defaultOutput.File})
}
