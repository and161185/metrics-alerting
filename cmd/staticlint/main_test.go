package main

import (
	"testing"

	"golang.org/x/tools/go/analysis/passes/printf"
)

func Test_collect_AddsAnalyzers(t *testing.T) {
	as := collect()
	if len(as) == 0 {
		t.Fatal("collect returned empty")
	}

	_ = printf.Analyzer
}
