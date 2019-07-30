package mic

import "github.com/golang/glog"

// Log - this struct abstracts the logging libraries into single interface.
type Log struct {
}

// Info - log the messages to be info
func (l Log) Info(msg string) {
	glog.Info(msg)
}

// Errorf - log the messages to be error messages and formatted.
func (l Log) Errorf(format string, args ...interface{}) {
	glog.Errorf(format, &args)
}
