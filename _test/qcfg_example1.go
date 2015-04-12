package main

import (
	"fmt"
	"github.com/LDCS/qcfg"
)

var cfgFile = "./_sample.cfg"

func main() {
	qcfg := qcfg.NewCfg("cfg1", cfgFile, false)

	defaultNumProcs, defaultUser, defaultMilliseconds, defaultRatio := 1, "nobody", int64(42000000000), 0.3
	fmt.Printf("user=%s numprocs=%d millisecs=%d ratio=%g\n",
		qcfg.Str("someblock", "somerow", "user", defaultUser),
		qcfg.Int("thirdblock", "some-row", "numProcs", defaultNumProcs),
		qcfg.Int64("block4", "anotherrow", "millis", defaultMilliseconds),
		qcfg.Float64("anotherblock", "job", "ratio", defaultRatio))
}
