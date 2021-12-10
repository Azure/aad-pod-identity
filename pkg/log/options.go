package log

import (
	"flag"
	"fmt"

	"k8s.io/component-base/config"
	json "k8s.io/component-base/logs/json"
	"k8s.io/klog/v2"
)

const (
	logFormatFlagName = "log-format"
	// default log format
	textLogFormat = "text"
	jsonLogFormat = "json"
)

// Options has klog format parameters.
type Options struct {
	LogFormat string
}

// NewOptions return new klog options.
func NewOptions() *Options {
	return &Options{
		LogFormat: textLogFormat,
	}
}

// AddFlags adds log-format flag.
func (o *Options) AddFlags() {
	fs := flag.CommandLine
	fs.StringVar(&o.LogFormat, logFormatFlagName, textLogFormat, fmt.Sprintf("Sets the logging format. One of (%s|%s)", textLogFormat, jsonLogFormat))
}

// Validate validates the log-format flag.
func (o *Options) Validate() error {
	if o.LogFormat != textLogFormat && o.LogFormat != jsonLogFormat {
		return fmt.Errorf("unknown logging format %s. Only \"%s\" and \"%s\" are supported", o.LogFormat, textLogFormat, jsonLogFormat)
	}

	return nil
}

// Apply set klog logger from LogFormat type.
func (o *Options) Apply() error {
	if err := o.Validate(); err != nil {
		return err
	}

	if o.LogFormat == jsonLogFormat {
		logger, _ := json.Factory{}.Create(config.FormatOptions{})
		klog.SetLogger(logger)
	}

	return nil
}
