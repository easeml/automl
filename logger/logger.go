package logger

// Logger represents a logging interface.
type Logger interface {
	WriteDebug(message string)
	WriteInfo(message string)
	WriteWarning(message string)
	WriteError(message string)
	WriteFatal(message string)

	WithFields(args ...interface{}) Logger
	WithStack(error) Logger
	WithError(error) Logger
}

// EmptyLogger is a logger that simply swallows all the logging function calls and does nothing.
type EmptyLogger struct{}

// WriteDebug writes a debug message to the logger.
func (logger *EmptyLogger) WriteDebug(message string) {}

// WriteInfo writes a debug message to the logger.
func (logger *EmptyLogger) WriteInfo(message string) {}

// WriteWarning writes a debug message to the logger.
func (logger *EmptyLogger) WriteWarning(message string) {}

// WriteError writes a debug message to the logger.
func (logger *EmptyLogger) WriteError(message string) {}

// WriteFatal writes a debug message to the logger.
func (logger *EmptyLogger) WriteFatal(message string) {}

// WithFields adds fields to the next logged message.
func (logger *EmptyLogger) WithFields(args ...interface{}) Logger { return logger }

// WithStack adds a stack trace from a given error.
func (logger *EmptyLogger) WithStack(error) Logger { return logger }

// WithError adds an error message from a given error.
func (logger *EmptyLogger) WithError(error) Logger { return logger }
