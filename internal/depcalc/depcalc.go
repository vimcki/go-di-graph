package depcalc

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"github.com/vimcki/go-di-graph/internal/deptree"
	"github.com/vimcki/go-di-graph/internal/deptree/evaluator"
	"github.com/vimcki/go-di-graph/internal/globals"
)

type BuildInfo struct {
	Imports map[string]string
	Config  Config
	Result  Result
}

type Config struct {
	Package string
	Struct  string
}

type Result struct {
	Type string
}

func Depcalc(entryPoint, path string) (string, error) {
	packages, err := parsePackages(path)
	if err != nil {
		os.Exit(1)
	}

	var start *ast.FuncDecl

	c, err := globals.Get(packages)
	if err != nil {
		return "", fmt.Errorf("failed to get globals: %w", err)
	}

	for _, pkg := range packages {
		fnMap := make(map[string]*ast.FuncDecl)
		for _, file := range pkg.Files {
			ast.Inspect(file, func(node ast.Node) bool {
				switch t := node.(type) {
				case *ast.FuncDecl:
					if t.Name.Name != entryPoint {
						fnMap[t.Name.Name] = t
						return true
					}
					start = t
				}
				return true
			})
		}

		e := evaluator.NewEvaluator(fnMap, c)
		dep, err := e.Eval(start)
		if err != nil {
			return "", fmt.Errorf("failed to evaluate dependencies: %w", err)
		}
		printDep(dep, 0)
		bytes, err := json.Marshal(dep)
		if err != nil {
			return "", fmt.Errorf("failed to marshal dependencies: %w", err)
		}
		return string(bytes), nil
	}
	return "", errors.New("no packages found")
}

// parsePackages parses all packages in the specified directory.
func parsePackages(dirPath string) (map[string]*ast.Package, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dirPath, nil, 0)
	if err != nil {
		return nil, err
	}
	return pkgs, nil
}

func printDep(dep deptree.Dependency, indent int) {
	for _, dep := range dep.Deps {
		printDep(dep, indent+1)
	}
}
