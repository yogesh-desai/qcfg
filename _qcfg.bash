#!/bin/bash
$GOROOT/bin/gofmt qcfg.go > qcfg.fmt
$GOROOT/bin/go test

