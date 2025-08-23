// Package noosexit implements a custom analyzer forbidding os.Exit in main.
package noosexit

import (
	"go/ast"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

// Analyzer forbids direct os.Exit calls inside main() of package main.
//
// This analyzer only flags direct calls *inside* main.main body.
var Analyzer = &analysis.Analyzer{
	Name: "noosexit",
	Doc:  "forbid direct os.Exit in main.main",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg == nil || pass.Pkg.Name() != "main" {
		return nil, nil
	}
	if strings.HasSuffix(pass.Pkg.Path(), "/cmd/staticlint") {
		return nil, nil
	}
	for _, f := range pass.Files {

		fn := pass.Fset.Position(f.Pos()).Filename
		if strings.Contains(fn, "/.cache/go-build/") || isGenerated(f) || importsTesting(f) {
			continue // игнорим testmain/сгенерённое
		}

		ast.Inspect(f, func(n ast.Node) bool {
			fd, ok := n.(*ast.FuncDecl)
			if !ok || fd.Recv != nil || fd.Name == nil || fd.Name.Name != "main" {
				return true
			}

			// We are in main.main body
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				// match os.Exit(...)
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok && sel.Sel != nil {
						if id.Name == "os" && sel.Sel.Name == "Exit" {
							pass.Reportf(call.Pos(), "do not call os.Exit inside main; delegate to run() and return code")
						}
					}
				}
				return true
			})
			return false // no need to inspect other functions deeply
		})
	}
	return nil, nil
}

func isGenerated(f *ast.File) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "Code generated") && strings.Contains(c.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}

func importsTesting(f *ast.File) bool {
	for _, im := range f.Imports {
		if p, _ := strconv.Unquote(im.Path.Value); p == "testing" || p == "testing/internal/testdeps" {
			return true
		}
	}
	return false
}
