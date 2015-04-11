// Package qcfg provides for reading a specific config file format into memory, query its elements, and write the representation to file.
//
// The format is tuned for ease of creation/maintenance of configuration of applications written in certain languages by a specific set of people.
//
// Candidates passed over include XML (human-unreadable), JSON (quote-burdened), Unix ini files (unhierarchical), Java properties files (unhierarchical), YAML (indentatious), eval'ed code (language specific).
//
// One could easily implement a diff program to highlight non-cosmetic changes (whitespace, comments and file/row/block reordering) to be used prior to checkin of config changes.
//
// The format is plain text with named blocks as the main construct.
// Blocks contain named rows, and rows contained named columns.
// Blocks can also contain named blocks, hence the hierarchy.
// Blocks require 3 lines to define.
//
// (1) A line containing "%block SomeBlockName" which names the block.
// (2) An immediately following line with an opening "{".
// (3) After its content ends, a line containing a closing "}"
//
// At any point, files may include other files using "%include /some/other/file".
// This facilitates sharing common config between apps.
//
// While it should work otherwise, maintainability dictates you should
//
// (1) put the blockname and its opening/closing parens as well as the lines defining the ROWS of the block WITHIN THE SAME FILE.
// (2) use the %include file directive to include the definition of large embedded block hierarchies.
// (3) use the %include file directive to include the definition of several blocks that have related meaning.
//
// Whitespace at beginning of line or end of line is ignored.
// Comments start at the Hash char (#) and extend to EOL.
// Empty lines are ignored.
// Block, row and column names are trimmed, so also column values.
//
// Rows are defined by rowname on the left followed by "::" followed by list of column name=val pairs, each terminated by semi-colon.
// Rows may be continued to the next line by the appearance of "+=" on the left of further column name-val pairs.
//
// Env-variable substitution is not yet implemented.
package qcfg

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/user"
	"strings"
)

type cfgRow struct {
	name string
	cols map[string]string
}

// CfgBlock struct holds the in-memory representation of a top-level config file
type CfgBlock struct {
	name  string                 // block name
	fname string                 // file containing the block
	rows  map[string](*cfgRow)   // non-recursive
	tbls  map[string](*CfgBlock) // recursive
}

var cfgs = make(map[string]*CfgBlock) // The set of top-level configs initialized within the program

func cleanLine(_line *[]byte) {
	// remove leading and trailing blanks
	nn := bytes.Index(*_line, []byte("#"))
	if nn >= 0 {
		*_line = (*_line)[:nn]
	}
	line := bytes.TrimLeft(*_line, " 	")
	line = bytes.TrimRight(line, " 	")
	*_line = line
}

func getFilename(_line []byte) []byte {
	parts := bytes.SplitN(_line, []byte(" "), 2)
	if len(parts) > 1 {
		if bytes.Equal([]byte("%include"), bytes.ToLower(parts[0])) {
			cleanLine(&parts[1])
			if parts[1][0] == '"' {
				return parts[1][1 : len(parts[1])-1]
			}
			return parts[1]
		}
	}
	return nil
}

func lineIsInclude(_line []byte) bool {
	if (len(_line) > 8) && bytes.Equal([]byte("%include"), _line[:8]) {
		return true
	}
	return false
}

func lineIsBlockEnd(_line []byte) bool {
	if (len(_line) > 0) && (_line[0] == '}') {
		return true
	}
	return false
}

func lineIsBlockNew(_line []byte) bool {
	parts := bytes.SplitN(_line, []byte(" "), 2)
	if len(parts) > 1 {
		if bytes.Equal([]byte("%block"), bytes.ToLower(parts[0])) {
			return true
		}
	}
	return false
}

// Block xxx
func getBlockname(_line []byte) []byte {
	parts := bytes.SplitN(_line, []byte(" "), 2)
	if len(parts) > 1 {
		if bytes.Equal([]byte("%block"), bytes.ToLower(parts[0])) {
			cleanLine(&parts[1])
			return parts[1]
		}
	}
	fmt.Printf("getBlockname: line(%s) is not block\n", string(_line))
	return []byte("") // Should never be called such that it would reach here
}

func (cfg CfgBlock) loadRow(_line []byte, _fname string, _rowName []byte) []byte {
	cleanLine(&_line)

	add := false

	var rowData []byte
	if len(_rowName) < 1 {
		nnNew := bytes.Index(_line, []byte("::"))
		nnAdd := bytes.Index(_line, []byte("+="))
		if (nnNew >= 0) && (nnAdd > 0) {
			if nnNew > nnAdd {
				_rowName = _line[:nnAdd]
				rowData = _line[nnAdd+2:]
			} else {
				_rowName = _line[:nnNew]
				rowData = _line[nnNew+2:]
				add = true
			}
		} else if nnAdd > 0 {
			_rowName = _line[:nnAdd]
			rowData = _line[nnAdd+2:]
			add = true
		} else if nnNew > 0 {
			_rowName = _line[:nnNew]
			rowData = _line[nnNew+2:]
		} else {
			fmt.Printf("loadRow: could not understand cleaned line (%s)\n", string(_line))
			return nil
		}
		cleanLine(&_rowName)
		cleanLine(&rowData)
	} else {
		add = true
		rowData = _line
	}

	row, ok := cfg.rows[(string)(_rowName)]
	if (ok && !add) || (!ok) {
		cfg.rows[(string)(_rowName)] = &cfgRow{string(_rowName), make(map[string]string, 1)}
		row, ok = cfg.rows[(string)(_rowName)]
		if !ok {
			fmt.Printf("loadRow:  rowName(%s) not found in rows, added, still not found\n", string(_rowName))
		}
	}

	cols := bytes.Split(rowData, []byte(";"))

	var kvarr [][]byte
	for str := range cols {
		kvarr = bytes.SplitN(cols[str], []byte("="), 2)
		cleanLine(&kvarr[0])
		if len(kvarr[0]) < 1 {
			continue
		}
		if len(kvarr) < 2 {
			continue
		}
		cleanLine(&kvarr[1])
		row.cols[(string)(kvarr[0])] = (string)(kvarr[1])
	}
	return _rowName
}

// Recursive call to read a Block
func (cfg CfgBlock) loadBlock(_rdr *bufio.Reader, _fname string, _tblname string, _verbose bool) {
	done := false
	var prevRow []byte
	for done == false {
		buf, err := _rdr.ReadBytes('\n')
		if err != nil {
			done = true
		}
		if len(buf) < 1 {
			continue
		}
		buf = buf[:len(buf)-1]
		cleanLine(&buf)
		if len(buf) < 1 {
			continue
		}
		if false {
		} else if lineIsInclude(buf) {
			// recursive call, which assumes there was no partially unconsumed line
			fname2 := expandUser(string(getFilename(bytes.TrimSpace(buf))))
			fpNew, err := os.Open(fname2)
			if err != nil {
				panic("could not open file " + fname2)
			}
			rdrNew := bufio.NewReader(fpNew)
			cfg.loadCfgFile(fname2, rdrNew, _verbose)
			fpNew.Close()
		} else if lineIsBlockEnd(buf) {
			// processBlock, which assumes there was no partially unconsumed line
			return
		} else if lineIsBlockNew(buf) {
			// processBlock, which assumes there was no partially unconsumed line
			name2 := string(getBlockname(buf))
			cfg.tbls[name2] = &CfgBlock{name2, _fname, make(map[string](*cfgRow), 1), make(map[string](*CfgBlock), 1)}
			cfg.tbls[name2].loadBlock(_rdr, _fname, name2, _verbose)
		} else if (len(buf) > 2) && (buf[0] == '+') && (buf[1] == '=') {
			prevRow = cfg.loadRow(buf[2:], _fname, prevRow)
		} else if (len(buf) > 0) && (buf[0] == '{') {
		} else {
			prevRow = cfg.loadRow(buf, _fname, nil)
		}
	}
	return
}

// Recursive call to read a file
func (cfg CfgBlock) loadCfgFile(_fname string, _rdr *bufio.Reader, _verbose bool) {
	done := false
	var prevRow []byte
	for done == false {
		buf, err := _rdr.ReadBytes('\n')
		if err != nil {
			done = true
		}
		if len(buf) < 1 {
			continue
		}
		buf = buf[:len(buf)-1]
		cleanLine(&buf)
		if len(buf) < 1 {
			continue
		}
		if false {
		} else if lineIsInclude(buf) {
			// recursive call, which assumes there was no partially unconsumed line
			fname2 := expandUser(string(getFilename(bytes.TrimSpace(buf))))
			if _verbose {
				fmt.Println("qcfg.loadCfgFile: opening file", fname2)
			}
			fpNew, err := os.Open(fname2)
			if err != nil {
				panic("could not open file " + fname2)
			}
			rdrNew := bufio.NewReader(fpNew)
			cfg.loadCfgFile(fname2, rdrNew, _verbose)
			fpNew.Close()
		} else if lineIsBlockEnd(buf) {
			// processBlock, which assumes there was no partially unconsumed line
			return
		} else if lineIsBlockNew(buf) {
			// processBlock, which assumes there was no partially unconsumed line
			name2 := string(getBlockname(buf))
			cfg.tbls[name2] = &CfgBlock{name2, _fname, make(map[string](*cfgRow), 1), make(map[string](*CfgBlock), 1)}
			cfg.tbls[name2].loadBlock(_rdr, _fname, name2, _verbose)
		} else if (len(buf) > 2) && (buf[0] == '+') && (buf[1] == '=') {
			fmt.Printf("loadCfgFile: will loadRow add(%s)\n", string(buf), _verbose)
			prevRow = cfg.loadRow(buf[2:], _fname, prevRow)
		} else if (len(buf) > 0) && (buf[0] == '{') {
		} else {
			fmt.Printf("loadCfgFile: will loadRow new(%s)\n", string(buf))
			prevRow = cfg.loadRow(buf, _fname, nil)
		}
	}
}

// NewCfg reads ()and parses) a new top-level config file (and recursively any config files that are included)
func NewCfg(_name string, _fname string, _verbose bool) *CfgBlock {
	_fname = expandUser(_fname)
	cfg, ok := cfgs[_name]
	if ok {
		return cfg
	}
	cfg = &CfgBlock{_name, _fname, make(map[string](*cfgRow), 1), make(map[string](*CfgBlock), 1)}
	cfgs[_name] = cfg
	cfg, ok = cfgs[_name]
	if !ok {
		panic("cfg: NewCfg failed for cfg " + _name)
	}
	if cfg == nil {
		panic("cfg: NewCfg cfg could not be created for cfg" + _name)
	}
	fpNew, err := os.Open(_fname)
	if err != nil {
		panic("could not open file " + _fname)
	}
	rdr := bufio.NewReader(fpNew)
	cfg.loadCfgFile(_fname, rdr, _verbose)
	fpNew.Close()
	return cfg
}

// NewCfgMem creates an empty in-memory representation of a config file.
// Use it to create config files programmatically
func NewCfgMem(_name string) *CfgBlock {
	cfg, ok := cfgs[_name]
	if ok {
		panic("cfg: NewCfgMem or NewCfg already called for cfg " + _name)
	}
	cfg = &CfgBlock{_name, "", make(map[string](*cfgRow), 1), make(map[string](*CfgBlock), 1)}
	cfgs[_name] = cfg
	cfg, ok = cfgs[_name]
	if !ok {
		panic("cfg: NewCfgMem failed for cfg " + _name)
	}
	if cfg == nil {
		panic("cfg: NewCfgMem cfg could not be created for cfg" + _name)
	}
	return cfg
}

// EditEntry updates en element of the in-memory representation of a config file.
// Use it to modify the configuration for subsequent use of the instance, or in preparation to write a modified config file
func (cfg *CfgBlock) EditEntry(_tbl, _row, _col, value string) {
	tbl, ok := cfg.tbls[_tbl]
	if ok == false {
		cfg.tbls[_tbl] = &CfgBlock{cfg.name, "", make(map[string](*cfgRow), 1), make(map[string](*CfgBlock), 1)}
		tbl = cfg.tbls[_tbl]
	}
	row, ok := tbl.rows[_row]
	if ok == false {
		tbl.rows[_row] = &cfgRow{_row, make(map[string]string)}
		row = tbl.rows[_row]
	}
	row.cols[_col] = value
}

// CfgWrite is used to programmatically create a new config file by writing out its in-memory representation
func (cfg CfgBlock) CfgWrite(_filename string) {
	_filename = expandUser(_filename)
	fp, err := os.OpenFile(_filename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		panic("Cannot open filename : " + _filename + " Error :" + err.Error())
	}
	bb := bufio.NewWriter(fp)
	for tbl, tblcontent := range cfg.tbls {
		bb.WriteString("\n%block " + tbl + "\n{\n")
		for rowname, rowcontent := range tblcontent.rows {
			bb.WriteString("\t" + rowname + "\t:: ")
			for colname, value := range rowcontent.cols {
				bb.WriteString(colname + "=" + value + "; ")
			}
			bb.WriteString("\n")
		}
		bb.WriteString("\n}\n")
	}
	bb.Flush()
	fp.Close()
}

// Str is used to query an element of the in-memory representation of the config file, as type string.  It returns the specified default if the element is missing
func (cfg CfgBlock) Str(_tbl, _row, _col string, _def string) string {
	tbl, ok := cfg.tbls[_tbl]
	if !ok {
		fmt.Printf("did not find tbl (%s)\n", _tbl)
		return _def
	}
	row, ok := tbl.rows[_row]
	if !ok {
		fmt.Printf("did not find tbl (%s) has row (%s)\n", _tbl, _row)
		return _def
	}
	if !ok {
		return _def
	}
	col, ok := row.cols[_col]
	if !ok {
		return _def
	}
	return col
}

// SelfStr applies Str() on self
func (cfg CfgBlock) SelfStr(_row, _col string, _def string) string {
	row, ok := cfg.rows[_row]
	if !ok {
		fmt.Printf("did not find row (%s)\n", _row)
		return _def
	}
	if !ok {
		return _def
	}
	col, ok := row.cols[_col]
	if !ok {
		return _def
	}
	return col
}

// NestedStr applies Str() on a nested block
func (cfg CfgBlock) NestedStr(_tbls []string, _row, _col string, _def string) string {
	nn := len(_tbls)
	if nn == 0 {
		return cfg.SelfStr(_row, _col, _def)
	}
	nn--
	if nn == 0 {
		return cfg.Str(_tbls[0], _row, _col, _def)
	}
	cfg1 := cfg.GetBlock(_tbls[:nn])
	if cfg1 == nil {
		return _def
	}
	return cfg1.Str(_tbls[nn], _row, _col, _def)
}

// Int is used to query an element of the in-memory representation of the config file, as type int.  It returns the specified default if the element is missing
func (cfg CfgBlock) Int(_tbl, _row, _col string, _def int) int {
	tbl, ok := cfg.tbls[_tbl]
	if !ok {
		fmt.Printf("did not find tbl (%s)\n", _tbl)
		return _def
	}
	row, ok := tbl.rows[_row]
	if !ok {
		fmt.Printf("did not find tbl (%s) has row (%s)\n", _tbl, _row)
		return _def
	}
	col, ok := row.cols[_col]
	if !ok {
		return _def
	}
	ival := int(_def)
	fmt.Sscanf(col, "%d", &ival)
	return ival
}

// SelfInt applies Int() on self
func (cfg CfgBlock) SelfInt(_row, _col string, _def int) int {
	row, ok := cfg.rows[_row]
	if !ok {
		fmt.Printf("did not find row (%s)\n", _row)
		return _def
	}
	if !ok {
		return _def
	}
	col, ok := row.cols[_col]
	if !ok {
		return _def
	}
	ival := int(_def)
	fmt.Sscanf(col, "%d", &ival)
	return ival
}

// NestedInt applies Int() on a nested block
func (cfg CfgBlock) NestedInt(_tbls []string, _row, _col string, _def int) int {
	nn := len(_tbls)
	if nn == 0 {
		return cfg.SelfInt(_row, _col, _def)
	}
	nn--
	if nn == 0 {
		return cfg.Int(_tbls[0], _row, _col, _def)
	}
	cfg1 := cfg.GetBlock(_tbls[:nn])
	if cfg1 == nil {
		return _def
	}
	return cfg1.Int(_tbls[nn], _row, _col, _def)
}

// Int64 is used to query an element of the in-memory representation of the config file, as type int64.  It returns the specified default if the element is missing
func (cfg CfgBlock) Int64(_tbl, _row, _col string, _def int64) int64 {
	tbl, ok := cfg.tbls[_tbl]
	if !ok {
		fmt.Printf("did not find tbl (%s)\n", _tbl)
		return _def
	}
	row, ok := tbl.rows[_row]
	if !ok {
		fmt.Printf("did not find tbl (%s) has row (%s)\n", _tbl, _row)
		return _def
	}
	col, ok := row.cols[_col]
	if !ok {
		return _def
	}
	ival := int(_def)
	fmt.Sscanf(col, "%d", &ival)
	return int64(ival)
}

// SelfInt64 applies Int64() on self
func (cfg CfgBlock) SelfInt64(_row, _col string, _def int64) int64 {
	row, ok := cfg.rows[_row]
	if !ok {
		fmt.Printf("did not find row (%s)\n", _row)
		return _def
	}
	if !ok {
		return _def
	}
	col, ok := row.cols[_col]
	if !ok {
		return _def
	}
	ival := int(_def)
	fmt.Sscanf(col, "%d", &ival)
	return int64(ival)
}

// NestedInt64 applies Int64() on a nested block
func (cfg CfgBlock) NestedInt64(_tbls []string, _row, _col string, _def int64) int64 {
	nn := len(_tbls)
	if nn == 0 {
		return cfg.SelfInt64(_row, _col, _def)
	}
	nn--
	if nn == 0 {
		return cfg.Int64(_tbls[0], _row, _col, _def)
	}
	cfg1 := cfg.GetBlock(_tbls[:nn])
	if cfg1 == nil {
		return _def
	}
	return cfg1.Int64(_tbls[nn], _row, _col, _def)
}

// Float64 is used to query an element of the in-memory representation of the config file, as type float64.  It returns the specified default if the element is missing
func (cfg CfgBlock) Float64(_tbl, _row, _col string, _def float64) float64 {
	tbl, ok := cfg.tbls[_tbl]
	if !ok {
		fmt.Printf("did not find tbl (%s)\n", _tbl)
		return _def
	}
	row, ok := tbl.rows[_row]
	if !ok {
		fmt.Printf("did not find tbl (%s) has row (%s)\n", _tbl, _row)
		return _def
	}
	col, ok := row.cols[_col]
	if !ok {
		return _def
	}
	ival := _def
	fmt.Sscanf(col, "%g", &ival)
	return float64(ival)
}

// SelfFloat64 applies Float64() on self
func (cfg CfgBlock) SelfFloat64(_row, _col string, _def float64) float64 {
	row, ok := cfg.rows[_row]
	if !ok {
		fmt.Printf("did not find row (%s)\n", _row)
		return _def
	}
	if !ok {
		return _def
	}
	col, ok := row.cols[_col]
	if !ok {
		return _def
	}
	ival := _def
	fmt.Sscanf(col, "%g", &ival)
	return float64(ival)
}

// NestedFloat64 applies Float64() on a nested block
func (cfg CfgBlock) NestedFloat64(_tbls []string, _row, _col string, _def float64) float64 {
	nn := len(_tbls)
	if nn == 0 {
		return cfg.SelfFloat64(_row, _col, _def)
	}
	nn--
	if nn == 0 {
		return cfg.Float64(_tbls[0], _row, _col, _def)
	}
	cfg1 := cfg.GetBlock(_tbls[:nn])
	if cfg1 == nil {
		return _def
	}
	return cfg1.Float64(_tbls[nn], _row, _col, _def)
}

// GetBlocks returns a list of names of all the blocks (aka blocks) within the current block
// Use it when you want to process an entire config file
func (cfg CfgBlock) GetBlocks() []string {
	blocks := make([]string, len(cfg.tbls))
	ii := 0
	for kk := range cfg.tbls {
		blocks[ii] = kk
		ii++
	}
	return blocks
}

// GetBlock	returns the block found by following down a block hierarchy
func (cfg CfgBlock) GetBlock(_blockPath []string) *CfgBlock {
	cfg1, ok := &cfg, true
	for _, tbl := range _blockPath {
		cfg1, ok = cfg1.tbls[tbl]
		if !ok {
			fmt.Println("GetBlock: path=", strings.Join(_blockPath, ":"), " failed")
			return nil
		}
	}
	return cfg1
}

// GetRows returns a list of names of all rows within a specific block (block)
func (cfg CfgBlock) GetRows(_tbl string) []string {
	rows := []string{}
	tbl, ok := cfg.tbls[_tbl]
	if ok {
		for rr := range tbl.rows {
			rows = append(rows, rr)
		}
	} else {
		fmt.Printf("did not find tbl (%s)\n", _tbl)
	}
	return rows
}

// GetCols returns a list of names of all columns within a specific rows of a specific block (block)
func (cfg CfgBlock) GetCols(_tbl, _row string) []string {
	cols := []string{}
	tbl, ok := cfg.tbls[_tbl]
	if ok {
		row, ok1 := tbl.rows[_row]
		if ok1 {
			for cc := range row.cols {
				cols = append(cols, cc)
			}
		} else {
			fmt.Printf("did not find row (%s) in tbl (%s)\n", _row, _tbl)
		}
	} else {
		fmt.Printf("did not find tbl (%s)\n", _tbl)
	}
	return cols
}

// RowExists is used to verify if a specific row exists within a specific block (block)
func (cfg CfgBlock) RowExists(block, row string) bool {
	tbl, ok := cfg.tbls[block]
	if ok == false {
		return false
	}
	_, ok = tbl.rows[row]
	if ok == false {
		return false
	}
	return true
}

// Split is shorthand for csv-splitting the output of qcfg packages Str func
func (cfg CfgBlock) Split(_tbl, _row, _col string, _def string) []string {
	return strings.Split(cfg.Str(_tbl, _row, _col, _def), ",")
}

// Expandlist is a shorthand method
// If box.row.col == "foo1,foo2,..."
// then return unique list of {box.row2.foo1, box.row2.foo2, ...}
func (cfg *CfgBlock) Expandlist(_block, _row, _col, _row2 string) []string {
	parts := []string{}
	switch {
	case len(_block) <= 0:
		return parts
	case len(_row) <= 0:
		return parts
	case len(_col) <= 0:
		return parts
	case len(_row2) <= 0:
		return parts
	}

	partsmap := map[string]bool{}
	for _, boxtype := range cfg.Split(_block, _row, _col, "") {
		for _, box := range cfg.Split(_block, _row2, boxtype, "") {
			partsmap[box] = true
		}
	}

	for kk := range partsmap {
		parts = append(parts, kk)
	}
	return parts
}

func expandUser(_fname string) string {
	switch {
	case len(_fname) < 2:
		return _fname
	case _fname[:2] != "~/":
		return _fname
	}
	usr, _ := user.Current()
	dir := usr.HomeDir
	return strings.Replace(_fname, "~/", dir+"/", 1)
}
