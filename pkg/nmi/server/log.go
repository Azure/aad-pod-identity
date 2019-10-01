package server

import (
	log "github.com/sirupsen/logrus"
)

// Log - this struct abstracts the logging libraries into single interface.
type Log struct {
}

// Info - log the messages to be info
func (l Log) Info(msg string) {
	log.Info(msg)
}

// Infof - log the messages to be info messages and formatted.
func (l Log) Infof(format string, args ...interface{}) {
	log.Infof(format, &args)
}

// Error - log the messages to be error
func (l Log) Error(err error) {
	log.Error(err)
}

// Errorf - log the messages to be error messages and formatted.
func (l Log) Errorf(format string, args ...interface{}) {
	log.Errorf(format, &args)
}
