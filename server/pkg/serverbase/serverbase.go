package serverbase

import (
	"os"

	"github.com/outrigdev/outrig/pkg/base"
	"github.com/outrigdev/outrig/pkg/utilfn"
)

const OutrigLockFile = "outrig.lock"

type FDLock interface {
	Close() error
}

func EnsureHomeDir() error {
	outrigHomeDir := utilfn.ExpandHomeDir(base.OutrigHome)
	return os.MkdirAll(outrigHomeDir, 0755)
}
