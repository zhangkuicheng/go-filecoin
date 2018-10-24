package main

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/filecoin-project/go-filecoin/commands"
	obs "github.com/filecoin-project/go-filecoin/observability"
)

func main() {
	// TODO implement help text like so:
	// https://github.com/ipfs/go-ipfs/blob/master/core/commands/root.go#L91
	// TODO don't panic if run without a command.
	configureLogging()
	obs.Glog.WithField("args", os.Args).Debug("go-filecoin running command")
	code, _ := commands.Run(os.Args, os.Stdin, os.Stdout, os.Stderr)
	os.Exit(code)
}

func configureLogging() {
	// log to stderr
	obs.Glog.Logger.SetOutput(os.Stderr)
	// default log level is "info"
	obs.Glog.Logger.SetLevel(logrus.InfoLevel)

	// json or human format
	if os.Getenv("FIL_LOG_FORMAT") == "json" {
		obs.Glog.Logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		obs.Glog.Logger.SetFormatter(&logrus.TextFormatter{})
	}

	// log level other than default
	if level := os.Getenv("FIL_LOG_LEVEL"); level != "" {
		switch strings.ToLower(level) {
		case "trace":
			obs.Glog.Logger.SetLevel(logrus.TraceLevel)
		case "debug":
			obs.Glog.Logger.SetLevel(logrus.DebugLevel)
		case "info":
			obs.Glog.Logger.SetLevel(logrus.InfoLevel)
		case "warn":
			obs.Glog.Logger.SetLevel(logrus.WarnLevel)
		case "panic":
			obs.Glog.Logger.SetLevel(logrus.PanicLevel)
		case "fatal":
			obs.Glog.Logger.SetLevel(logrus.FatalLevel)
		default:
			obs.Glog.Warnf("%s is not a valid level, defaulting to info", level)
		}
	}

	// log to a file
	if file := os.Getenv("FIL_LOG_FILE"); file != "" {
		obs.Glog.Infof("setting log file to: %s", file)
		f, err := os.Create(file)
		if err != nil {
			obs.Glog.WithError(err).Errorf("FIL_LOG_FILE value: %s invalid", file)
		}
		obs.Glog.Logger.SetOutput(f)
	}

}
