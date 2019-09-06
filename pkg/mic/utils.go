package mic

import "github.com/golang/glog"

// Log - this struct abstracts the logging libraries into single interface.
type Log struct {
}

// Info - log the messages to be info
func (l Log) Info(msg string) {
	glog.Info(msg)
}

// Infof - log the messages to be info messages and formatted.
func (l Log) Infof(format string, args ...interface{}) {
	glog.Infof(format, &args)
}

// Errorf - log the messages to be error messages and formatted.
func (l Log) Errorf(format string, args ...interface{}) {
	glog.Errorf(format, &args)
}

// Error - log the messages to be error
func (l Log) Error(err error) {
	glog.Error(err)
}
