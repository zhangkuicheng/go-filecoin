package repo

import (
	ob "github.com/filecoin-project/go-filecoin/observability"
)

var log = ob.Glog.Logger.WithField("pkg", "repo")
