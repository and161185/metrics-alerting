// Package main provides the staticlint multichecker for this project.
//
// Build:
//
//	go build -o staticlint ./cmd/staticlint
//
// Usage:
//
//	./staticlint [packages]
//
// Examples:
//
//	./staticlint ./...
//	./staticlint $(go list ./... | grep -v '/cmd/staticlint$') // исключить сам multichecker
//
// Analyzers:
//
// Std passes (golang.org/x/tools/go/analysis/passes):
//   asmdecl        — verifies consistency between asm and Go declarations.
//   assign         — detects suspicious or useless assignments.
//   atomic         — checks correct usage of sync/atomic.
//   bools          — flags common mistakes in boolean expressions.
//   buildtag       — validates build tags.
//   cgocall        — warns about unsafe patterns in cgo calls/pointers.
//   composite      — detects errors in composite literals.
//   copylock       — reports copying of values containing mutexes (sync.Mutex, etc.).
//   errorsas       — checks proper use of errors.As.
//   framepointer   — checks frame pointer usage in builds/assembly.
//   httpresponse   — flags common mistakes when working with net/http responses.
//   ifaceassert    — detects unsafe type assertions.
//   loopclosure    — warns about loop variables captured by closures.
//   lostcancel     — finds missing cancel() calls for context.WithCancel/Timeout/Deadline.
//   nilfunc        — detects calls through nil function values.
//   printf         — checks printf-like calls for argument consistency.
//   shadow         — reports variable shadowing.
//   shift          — detects incorrect shifts (sign, size, overflow).
//   sigchanyzer    — detects misuse of os.Signal channels.
//   stdmethods     — checks signatures of standard interfaces (error, Stringer, etc.).
//   stringintconv  — flags suspicious string↔int conversions.
//   structtag      — checks format and validity of struct tags.
//   tests          — flags common mistakes in testing code.
//   unmarshal      — reports issues when unmarshalling into struct fields.
//   unreachable    — detects unreachable code.
//   unsafeptr      — warns about unsafe.Pointer conversions.
//   unusedresult   — detects ignored results of calls where results must be used.
//
// Staticcheck (honnef.co/go/tools) — all SA* (bug-finding rules).
//   Example: SA4006 (unnecessary assignment), SA1019 (deprecated API), SA5008 (invalid struct tags).
//
// Other staticcheck class:
//   ST1000 (stylecheck) — requires a package comment in the package.
//
// Public analyzers:
//   bodyclose (github.com/timakin/bodyclose) — ensures http.Response.Body is closed.
//   nilerr    (github.com/gostaticanalysis/nilerr) — in an `if err != nil` branch, returning a nil error.
//
// Custom:
//   noosexit — forbids direct os.Exit inside main.main; skips go-build cache files and generated code (Code generated …).

package main

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"

	"github.com/and161185/metrics-alerting/internal/analyzers/noosexit"
	"github.com/gostaticanalysis/nilerr"
	"github.com/timakin/bodyclose/passes/bodyclose"

	// std passes
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
)

func collect() []*analysis.Analyzer {
	list := []*analysis.Analyzer{
		// std passes
		asmdecl.Analyzer, assign.Analyzer, atomic.Analyzer, bools.Analyzer, buildtag.Analyzer,
		cgocall.Analyzer, composite.Analyzer, copylock.Analyzer, errorsas.Analyzer, framepointer.Analyzer,
		httpresponse.Analyzer, ifaceassert.Analyzer, loopclosure.Analyzer, lostcancel.Analyzer, nilfunc.Analyzer,
		printf.Analyzer, shift.Analyzer, sigchanyzer.Analyzer, stdmethods.Analyzer, stringintconv.Analyzer,
		structtag.Analyzer, tests.Analyzer, unmarshal.Analyzer, unreachable.Analyzer, unsafeptr.Analyzer,
		unusedresult.Analyzer, shadow.Analyzer,

		// custom
		noosexit.Analyzer,
	}

	// add all SA* analyzers
	for _, a := range staticcheck.Analyzers {
		if len(a.Analyzer.Name) >= 2 && a.Analyzer.Name[:2] == "SA" {
			list = append(list, a.Analyzer)
		}
	}

	list = append(list, stylecheck.Analyzers[0].Analyzer) // ST1000 (package comment)
	list = append(list, bodyclose.Analyzer)
	list = append(list, nilerr.Analyzer)

	return list
}

func main() {
	multichecker.Main(collect()...)
}
