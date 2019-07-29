package logger

// Logger - this interface is used to abstract the logging libraries used in MIC and NMI.
type Logger interface {
	Info(msg string)
	Errorf(format string, args ...interface{})
}
