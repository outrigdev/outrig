package loginit

import (
	"github.com/outrigdev/outrig/pkg/collector/logprocess"
)

func init() {
	collector := logprocess.GetInstance()
	collector.InitCollector(nil)
}
