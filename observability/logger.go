package observability

import (
	"github.com/sirupsen/logrus"
)

// a global logger that all other loggers are derived from
var Glog = logrus.New().WithFields(logrus.Fields{
	"program": "go-filecoin",
})
