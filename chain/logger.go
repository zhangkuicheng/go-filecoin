package chain

import (
	ob "github.com/filecoin-project/go-filecoin/observability"
)

var log = ob.Glog.Logger.WithField("pkg", "chain")
var logSyncer = log.WithField("service", "syncer")
var logStore = log.WithField("service", "store")
