package server

import log "github.com/sirupsen/logrus"

// Log - this struct abstracts the logging libraries into single interface.
type Log struct {
}

// Info - log the messages to be info
func (l Log) Info(msg string) {
	log.Info(msg)
}

// Errorf - log the messages to be error messages and formatted.
func (l Log) Errorf(format string, args ...interface{}) {
	log.Errorf(format, &args)
}
