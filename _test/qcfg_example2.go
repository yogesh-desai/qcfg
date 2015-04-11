package main

import (
    "fmt"
    "github.com/LDCS/qcfg"
)

var cfgFile = "./_sample.cfg"

func main() {
    qcfg := qcfg.NewCfg("cfg1", cfgFile, false)

	defaultNumProcs, defaultUser, defaultMilliseconds, defaultRatio	:= 1, "nobody", int64(42000000000), 0.3
    fmt.Printf("user=%s numprocs=%d millisecs=%d ratio=%g\n",
		qcfg.Str    ("someblock", "somerow", "user",       defaultUser),
		qcfg.Int    ("thirdblock", "some-row", "numProcs", defaultNumProcs),
		qcfg.Int64  ("block4", "anotherrow", "millis",     defaultMilliseconds),
		qcfg.Float64("anotherblock", "job", "ratio",       defaultRatio))

	// Now a nested block example, but done in 3 different ways

	// First way, traverse to the Parent block of the block holding interesting data, and lookup values
	cfg1 := qcfg.GetBlock([]string{"oneblock", "lowerblock0"})
	if cfg1 != nil {
		fmt.Printf("user=%s age=%d milli=%d ratio=%g\n",
			cfg1.Str    ("lowerblock", "inner-row", "user",  "nobody"),
			cfg1.Int    ("lowerblock", "inner-row", "age",   1),
			cfg1.Int64  ("lowerblock", "inner-row", "milli", 2),
			cfg1.Float64("lowerblock", "inner-row", "ratio", 0.1))
	}

	// Second way, traverse to the block itself (not its parent) and lookup values
	cfg2	:= qcfg.GetBlock([]string{"oneblock", "lowerblock0", "lowerblock"})
	fmt.Printf("user=%s age=%d milli=%d ratio=%g\n",
		cfg2.SelfStr    ("inner-row", "user",  "nobody"),
		cfg2.SelfInt    ("inner-row", "age",   1),
		cfg2.SelfInt64  ("inner-row", "milli", 2),
		cfg2.SelfFloat64("inner-row", "ratio", 0.1))

	// Third way, combine traversal and lookup
	fmt.Printf("user=%s age=%d milli=%d ratio=%g\n",
		qcfg.NestedStr    ([]string{"oneblock", "lowerblock0", "lowerblock"}, "inner-row", "user",  "nobody"),
		qcfg.NestedInt    ([]string{"oneblock", "lowerblock0", "lowerblock"}, "inner-row", "age",   1),
		qcfg.NestedInt64  ([]string{"oneblock", "lowerblock0", "lowerblock"}, "inner-row", "milli", 2),
		qcfg.NestedFloat64([]string{"oneblock", "lowerblock0", "lowerblock"}, "inner-row", "ratio", 0.1))

}
