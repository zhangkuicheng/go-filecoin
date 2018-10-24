package commands

import (
	ob "github.com/filecoin-project/go-filecoin/observability"
	"github.com/sirupsen/logrus"
)

var log = ob.Glog.Logger.WithField("pkg", "commands")

// TODO move to own package
var hbLog = logrus.New().WithFields(logrus.Fields{
	"pkg":     "commands",
	"service": "heartbeat",
})
