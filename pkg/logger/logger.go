package logger

// Logger - this interface is used to abstract the logging libraries used in MIC and NMI.
type Logger interface {
	Info(msg string)
	Error(err error)
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}
