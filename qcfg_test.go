package qcfg

import (
	"os"
	"io/ioutil"
	"testing"
	"math"
	"sort"
)

var cfgFile = "_sample.cfg"

// To test NewCfg()
func TestInitFile(t *testing.T) {
	NewCfg("TestInitFile", cfgFile, false)
}

// To test NewCfgMem()
func TestInitMem(t *testing.T) {
	NewCfgMem("TestInitMem")
}

// To test Str()
func TestStr(t *testing.T) {
	cfg := NewCfg("TestStr", cfgFile, false)
	if cfg.Str("thirdblock", "anotherrow", "end_time", "BLANK") != "235000" {
		t.Fail()
	}
	if cfg.Str("AXCFTWERdsr54", "GTERTR545", "end_time", "BLANK") != "BLANK" {
		t.Fail()
	}
}

// To test Int()
func TestInt(t *testing.T) {
    cfg := NewCfg("TestInt", cfgFile, false)
    if cfg.Int("thirdblock", "some-row", "numProcs", -1) != 8 {
        t.Fail()
    }
	if cfg.Int("AXCFTWERdsr54", "GTERTR545", "end_time", -11) != -11 {
		t.Fail()
	}
}

// To test Int64()
func TestInt64(t *testing.T) {
    cfg := NewCfg("TestInt64", cfgFile, false)
    if cfg.Int64("block4", "anotherrow", "millis", int64(0)) != int64(123456789) {
        t.Fail()
    }
	if cfg.Int64("AXCFTWERdsr54", "GTERTR545", "end_time", int64(-123456789)) != int64(-123456789) {
		t.Fail()
	}
}

// To test Float64()
func TestFloat64(t *testing.T) {
    cfg := NewCfg("TestFloat64", cfgFile, false)
    if math.Abs(cfg.Float64("anotherblock", "job", "ratio", 9999.99) - 0.3) > 0.000001 {
		t.Log(cfg.Float64("anotherblock", "job", "ratio", 9999.99))
        t.Fail()
    }
	if cfg.Float64("AXCFTWERdsr54", "GTERTR545", "end_time", 0.0) != 0.0 {
		t.Fail()
	}
}

// To test GetBlocks()
func TestGetBlocks(t *testing.T) {
	cfg := NewCfg("TestGetBlocks", cfgFile, false)
	actualblocks := []string{"oneblock", "thirdblock", "block4", "someblock", "anotherblock"}
	blocks := cfg.GetBlocks()
	if !isSetEqual(actualblocks, blocks) {
		t.Errorf("Expected block names = %+v\nActual block names = %+v\n", actualblocks, blocks)
	}
}

// to test GetRows()
func TestGetRows(t *testing.T) {
	cfg := NewCfg("TestGetRows", cfgFile, false)
	actualrows := []string{"somerow", "another-row", "lmirror", "proxy"}
	rows := cfg.GetRows("someblock")
	if !isSetEqual(actualrows, rows) {
		t.Errorf("Expected row names = %+v\nActual row names = %+v\n", actualrows, rows)
	}
	rows = cfg.GetRows("AXCFTWERdsr54")
	if len(rows) != 0 {
		t.Fail()
	}
}

// to test GetCols()
func TestGetCols(t *testing.T) {
	cfg := NewCfg("TestGetCols", cfgFile, false)
	actualcols := []string{"active", "prereqlist", "actionlist", "days", "start_time", "end_time", "watch_path", 
		"region", "datelist", "period", "freq", "ratio", "TZ"}
	cols := cfg.GetCols("anotherblock", "job")
	if !isSetEqual(actualcols, cols) {
		t.Errorf("Expected col names = %+v\nActual col names = %+v\n", actualcols, cols)
	}
	cols = cfg.GetCols("AXCFTWERdsr54", "RTERfrc4545")
	if len(cols) != 0 {
		t.Fail()
	}
}

// to test RowExists()
func TestRowExists(t *testing.T) {
	cfg := NewCfg("TestGetCols", cfgFile, false)
	if cfg.RowExists("anotherblock", "job") == false {
		t.Fail()
	}
	if cfg.RowExists("anotherblock", "ERTRertryYT545") == true {
		t.Fail()
	}
	if cfg.RowExists("AXCFTWERdsr54", "ERTRertryYT545") == true {
		t.Fail()
	}
}

// to test Split()
func TestSplit(t *testing.T) {
	cfg := NewCfg("TestSplit", cfgFile, false)
	plugins := cfg.Split("someblock", "lmirror", "plugins", "")
	if len(plugins) != 2 {
		t.Fail()
	} else if plugins[0] != "transpath" || plugins[1] != "split" {
		t.Fail()
	}
}

// To test EditEntry()
func TestEditEntry(t *testing.T) {
	cfg := NewCfg("TestEditEntry", cfgFile, false)
	cfg.EditEntry("thirdblock", "anotherrow", "end_time", "225000")
	if cfg.Str("thirdblock", "anotherrow", "end_time", "BLANK") != "225000" {
		t.Fail()
	}
}

// To test CfgWrite()
func TestCfgWrite(t *testing.T) {
    cfg := NewCfg("TestCfgWrite_1", cfgFile, false)
	tempfp, err := ioutil.TempFile("", "TestCfgWrite")
	if err != nil {
		t.Error("Error creating temp file for testing CfgWrite(), err =", err)
	}
	tempfile := tempfp.Name()
	err = tempfp.Close()
	if err != nil {
		t.Log("Warning : Could not close the tempfile", tempfile, "err =", err)
	}
	cfg.CfgWrite(tempfile)
	cfg = NewCfg("TestCfgWrite_2", tempfile, false)
	if cfg.Str("block4", "anotherrow", "millis", "BLANK") != "123456789" {
		t.Fail()
	}
	err = os.Remove(tempfile)
	if err != nil {
		t.Log("Warning : Could not remove temp file", tempfile, "err =", err)
	}
}


func isSetEqual(a, b []string) bool {
	if len(a) != len(b) { return false }
	sort.Strings(a)
	sort.Strings(b)
	for ii := 0; ii < len(a); ii++ {
		if a[ii] != b[ii] { return false }
	}
	return true
}
