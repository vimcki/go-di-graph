package evaluator

import (
	"errors"
	"fmt"
	"go/ast"
	"log"
	"reflect"
	"strings"
	"unicode"

	"github.com/vimcki/go-di-graph/internal/deptree"
)

var buitins = []string{
	"append",
}

type dependency struct {
	name    string
	deps    []dependency
	flatten bool
	created string
}

type Evaluator struct {
	env     *Environment
	fnMap   map[string]*ast.FuncDecl
	globals map[string]dependency
}

type Environment struct {
	dep map[string]dependency
}

func NewEvaluator(fnMap map[string]*ast.FuncDecl, c map[string]interface{}) *Evaluator {
	dep := make(map[string]dependency)
	for k, v := range c {
		dep[k] = dependency{
			name:    fmt.Sprintf("%v", v),
			created: "const",
			flatten: false,
		}
	}
	return &Evaluator{
		env: &Environment{
			dep: dep,
		},
		fnMap:   fnMap,
		globals: dep,
	}
}

func (e *Evaluator) getFuncNode(name string) (*ast.FuncDecl, error) {
	node, ok := e.fnMap[name]
	if !ok {
		return nil, errors.New("unknown function: " + name)
	}
	return node, nil
}

func (e *Evaluator) Eval(node ast.Node) (deptree.Dependency, error) {
	switch t := node.(type) {
	case *ast.FuncDecl:
		dep, err := e.EvalFunc(t)
		if err != nil {
			return deptree.Dependency{}, err
		}
		if len(dep.deps) == 0 {
			deps := []dependency{}
			for _, env := range e.env.dep {
				if strings.Contains(env.name, ".") {
					name := strings.Split(env.name, ".")[1]
					if unicode.IsUpper(rune(name[0])) {
						deps = append(deps, env)
					}
				}
			}
			dep = dependency{
				name:    "Aggregate",
				deps:    deps,
				flatten: false,
				created: "Eval",
			}
		}

		printDep(dep, 0)
		return toCommon(dep), nil
	default:
		return deptree.Dependency{}, errors.New("unknown node type in eval")
	}
}

func (e *Evaluator) EvalFunc(fn *ast.FuncDecl) (dependency, error) {
	var deps dependency
	var err error
	for _, stmt := range fn.Body.List {
		deps, err = e.evalStatement(stmt)
		if err != nil {
			return dependency{}, err
		}
	}

	return deps, nil
}

func (e *Evaluator) evalExpr(expr ast.Expr) (dependency, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return e.evalIdent(t)
	case *ast.CallExpr:
		return e.evalCallExpr(t)
	case *ast.SelectorExpr:
		return e.evalSelectorExpr(t)
	case *ast.CompositeLit:
		return e.evalCompositeLiteral(t)
	case *ast.BasicLit:
		return dependency{flatten: true, created: "BasicLit"}, nil
	case *ast.KeyValueExpr:
		return e.evalKeyValueExpr(t)
	case *ast.FuncLit:
		return e.evalFuncLit(t)
	case *ast.UnaryExpr:
		return e.evalExpr(t.X)
	default:
		return dependency{}, errors.New("unknown expr type in eval, " + reflect.TypeOf(t).String())
	}
}

func (e *Evaluator) evalFuncLit(expr *ast.FuncLit) (dependency, error) {
	// TODO this can overwrite local variables, this needs nested environments
	var deps dependency
	var err error
	for _, arg := range expr.Type.Params.List {
		e.env.dep[arg.Names[0].Name] = dependency{
			created: "FuncLit",
			flatten: true,
		}
	}
	for _, stmt := range expr.Body.List {
		deps, err = e.evalStatement(stmt)
		if err != nil {
			return dependency{}, err
		}
	}
	for _, arg := range expr.Type.Params.List {
		delete(e.env.dep, arg.Names[0].Name)
	}

	return deps, nil
}

func (e *Evaluator) evalKeyValueExpr(expr *ast.KeyValueExpr) (dependency, error) {
	deps := []dependency{}
	valDep, err := e.evalExpr(expr.Value)
	if err != nil {
		return dependency{}, err
	}

	deps = append(deps, valDep)

	var name string
	switch key := expr.Key.(type) {
	case *ast.Ident:
		name = key.Name
	case *ast.BasicLit:
		name = key.Value
	case *ast.SelectorExpr:
		var dep *dependency
		name, dep, err = e.evalSelectorRecursively(key)
		if err != nil {
			return dependency{}, fmt.Errorf("failed to eval selector recursively: %w", err)
		}
		if dep != nil {
			deps = append(deps, *dep)
		}
	default:
		return dependency{}, errors.New("unknown key type in eval, " + reflect.TypeOf(key).String())
	}

	return dependency{
		name:    name,
		deps:    deps,
		created: "KeyValueExpr",
	}, nil
}

func (e *Evaluator) evalIdent(t *ast.Ident) (dependency, error) {
	if t.Name == "nil" {
		return dependency{flatten: true, created: "nil"}, nil
	}
	identDep, ok := e.env.dep[t.Name]
	if !ok {
		log.Println(t.Name)
		return dependency{
			name:    t.Name + " (unknown)",
			created: "Ident, unknown",
		}, nil
	}
	// if name has . it means its a function call
	// this is a bad hack
	if strings.Contains(identDep.name, ".") {
		return identDep, nil
	}
	identDep.flatten = true
	return identDep, nil
}

func (e *Evaluator) evalSelectorExpr(expr *ast.SelectorExpr) (dependency, error) {
	ident, err := findIdent(expr)
	if err != nil {
		return dependency{}, err
	}

	dep, ok := e.env.dep[ident]
	if ok {
		return dep, nil
	}

	selector, recDep, err := e.evalSelectorRecursively(expr)
	if err != nil {
		return dependency{}, err
	}

	deps := []dependency{}

	if recDep != nil {
		deps = append(deps, *recDep)
	}

	return dependency{
		name:    selector,
		created: "SelectorExpr",
		deps:    deps,
	}, nil
}

func findIdent(expr *ast.SelectorExpr) (string, error) {
	switch expr.X.(type) {
	case *ast.Ident:
		return expr.X.(*ast.Ident).Name, nil
	case *ast.SelectorExpr:
		return findIdent(expr.X.(*ast.SelectorExpr))
	case *ast.CallExpr:
		switch expr.X.(*ast.CallExpr).Fun.(type) {
		case *ast.SelectorExpr:
			return findIdent(expr.X.(*ast.CallExpr).Fun.(*ast.SelectorExpr))
		default:
			return "", fmt.Errorf("unknown fun type in find ident: %s", reflect.TypeOf(expr.X).String())
		}
	default:
		return "", fmt.Errorf("unknown selector expr type in find ident: %s", reflect.TypeOf(expr.X).String())
	}
}

func (e *Evaluator) evalSelectorRecursively(expr *ast.SelectorExpr) (string, *dependency, error) {
	var err error
	var selector string
	var dep *dependency
	switch expr.X.(type) {
	case *ast.Ident:
		selector = expr.X.(*ast.Ident).Name

	case *ast.SelectorExpr:
		selector, dep, err = e.evalSelectorRecursively(expr.X.(*ast.SelectorExpr))
		if err != nil {
			return "", nil, fmt.Errorf("failed to eval selector recursively: %w", err)
		}
	case *ast.CallExpr:
		callDep, err := e.evalCallExpr(expr.X.(*ast.CallExpr))
		if err != nil {
			return "", nil, fmt.Errorf("failed to eval call expr: %w", err)
		}

		var selectorDep *dependency
		switch expr.X.(*ast.CallExpr).Fun.(type) {
		case *ast.SelectorExpr:
			selector, selectorDep, err = e.evalSelectorRecursively(expr.X.(*ast.CallExpr).Fun.(*ast.SelectorExpr))
			if err != nil {
				return "", nil, fmt.Errorf("failed to eval selector recursively: %w", err)
			}

		default:
			return "", nil, fmt.Errorf("unknown fun type in eval selector: %s", reflect.TypeOf(expr.X).String())
		}

		if selectorDep == nil {
			dep = &callDep
		} else {
			dep = &dependency{
				deps:    []dependency{*selectorDep, callDep},
				created: "SelectorRecursively",
			}
		}

	default:
		return "", nil, fmt.Errorf("unknown selector expr type in eval selector: %s", reflect.TypeOf(expr.X).String())
	}
	return selector + "." + expr.Sel.Name, dep, nil
}

func (e *Evaluator) evalCompositeLiteral(expr *ast.CompositeLit) (dependency, error) {
	if len(expr.Elts) == 0 {
		return dependency{flatten: true, created: "CompositeLit, empty"}, nil
	}

	first := expr.Elts[0]
	switch first.(type) {
	case *ast.KeyValueExpr:
		return e.evalCompositeLiteralMap(expr)
	default:
		return e.evalCompositeLiteralSlice(expr)
	}
}

func (e *Evaluator) evalCompositeLiteralMap(expr *ast.CompositeLit) (dependency, error) {
	var deps []dependency
	for _, elt := range expr.Elts {
		dep, err := e.evalExpr(elt)
		if err != nil {
			return dependency{}, err
		}
		deps = append(deps, dep)
	}
	return dependency{
		deps:    deps,
		flatten: false,
		created: "CompositeLitMap",
	}, nil
}

func (e *Evaluator) evalCompositeLiteralSlice(expr *ast.CompositeLit) (dependency, error) {
	var deps []dependency
	for _, elt := range expr.Elts {
		dep, err := e.evalExpr(elt)
		if err != nil {
			return dependency{}, err
		}
		deps = append(deps, dep)
	}
	return dependency{
		deps:    deps,
		flatten: true,
		created: "CompositeLitSlice",
	}, nil
}

func (e *Evaluator) evalCallExpr(callExpr *ast.CallExpr) (dependency, error) {
	var name string
	var flatten bool
	switch t := callExpr.Fun.(type) {
	case *ast.Ident:
		if !in(t.Name, buitins) {
			args := make(map[string]dependency)
			node, err := e.getFuncNode(t.Name)
			if err != nil {
				return dependency{}, err
			}
			nodeArgs := node.Type.Params.List
			for i, arg := range callExpr.Args {
				dep, err := e.evalExpr(arg)
				if err != nil {
					return dependency{}, err
				}
				ident := nodeArgs[i].Names[0].Name
				log.Println("ident", ident)
				args[ident] = dep
			}
			evaluator := NewEvaluatorFrom(e, args)
			dep, err := evaluator.EvalFunc(node)
			if err != nil {
				return dependency{}, err
			}
			dep.flatten = true
			return dep, nil
		}
		name = t.Name
		flatten = false
	case *ast.SelectorExpr:
		fnName := t.Sel.Name
		switch t.X.(type) {
		case *ast.Ident:
			name = t.X.(*ast.Ident).Name + "." + fnName
		case *ast.CallExpr:
			return e.evalCallExpr(t.X.(*ast.CallExpr))
		default:
			return dependency{}, errors.New("unknown call expr type in eval selector expr, " + reflect.TypeOf(t).String())
		}
		flatten = false
	default:
		return dependency{}, errors.New("unknown call expr type in eval, " + reflect.TypeOf(t).String())
	}

	var deps []dependency
	for _, arg := range callExpr.Args {
		dep, err := e.evalExpr(arg)
		if err != nil {
			return dependency{}, err
		}
		deps = append(deps, dep)
	}
	dep := dependency{
		name:    name,
		deps:    deps,
		flatten: flatten,
		created: "CallExpr",
	}
	return dep, nil
}

func NewEvaluatorFrom(e *Evaluator, args map[string]dependency) *Evaluator {
	dep := make(map[string]dependency)
	for k, v := range e.globals {
		dep[k] = v
	}
	for k, v := range args {
		dep[k] = v
	}
	return &Evaluator{
		env: &Environment{
			dep: dep,
		},
		fnMap:   e.fnMap,
		globals: e.globals,
	}
}

func (e *Evaluator) evalStatement(stmt ast.Stmt) (dependency, error) {
	switch t := stmt.(type) {
	case *ast.ReturnStmt:
		return e.evalExpr(t.Results[0])
	case *ast.AssignStmt:
		var deps []dependency
		for _, expr := range t.Rhs {
			dep, err := e.evalExpr(expr)
			if err != nil {
				return dependency{}, err
			}
			deps = append(deps, dep)
		}
		var err error
		for _, ident := range t.Lhs {
			name := ""
			switch i := ident.(type) {
			case *ast.Ident:
				name = ident.(*ast.Ident).Name
			case *ast.SelectorExpr:
				var dep *dependency
				name, dep, err = e.evalSelectorRecursively(i)
				if err != nil {
					return dependency{}, fmt.Errorf("failed to eval selector recursively: %w", err)
				}
				deps = append(deps, *dep)
			default:
				return dependency{}, errors.New("unknown assign stmt type in eval, " + reflect.TypeOf(i).String())
			}

			e.env.dep[name] = dependency{
				name:    name,
				deps:    deps,
				created: "AssignStmt",
			}
		}
		return dependency{
			created: "AssignStmt, return empty dep",
		}, nil
	case *ast.DeclStmt:
		return e.evalDeclStmt(t)
	case *ast.BlockStmt:
		return e.evalBlockStmt(t)
	case *ast.ExprStmt:
		return e.evalExpr(t.X)
	case *ast.RangeStmt:
		return e.evalRangeStmt(t)
	default:
		return dependency{}, errors.New("unknown statement type in eval, " + reflect.TypeOf(t).String())
	}
}

func (e *Evaluator) evalDeclStmt(stmt *ast.DeclStmt) (dependency, error) {
	name := stmt.Decl.(*ast.GenDecl).Specs[0].(*ast.ValueSpec).Names[0].Name
	e.env.dep[name] = dependency{
		flatten: true,
		created: "DeclStmt",
	}

	return dependency{
		created: "DeclStmt, return empty dep",
	}, nil
}

func (e *Evaluator) evalBlockStmt(stmt *ast.BlockStmt) (dependency, error) {
	for _, stmt := range stmt.List {
		_, err := e.evalStatement(stmt)
		if err != nil {
			return dependency{}, err
		}
	}
	return dependency{
		created: "BlockStmt, return empty dep",
	}, nil
}

func (e *Evaluator) evalRangeStmt(stmt *ast.RangeStmt) (dependency, error) {
	key := stmt.Key.(*ast.Ident).Name
	val := stmt.Value.(*ast.Ident).Name
	dep, err := e.evalExpr(stmt.X)
	if err != nil {
		return dependency{}, err
	}
	e.env.dep[key] = dep
	e.env.dep[val] = dep
	return e.evalBlockStmt(stmt.Body)
}

func in(s string, ss []string) bool {
	for _, sss := range ss {
		if sss == s {
			return true
		}
	}
	return false
}

func toCommon(dep dependency) deptree.Dependency {
	var deps []deptree.Dependency
	for _, d := range dep.deps {
		if d.flatten {
			deps = append(deps, skipFlatten(d)...)
			continue
		}
		deps = append(deps, toCommon(d))
	}
	return deptree.Dependency{
		Name: dep.name,
		Deps: deps,
		Desc: dep.created,
	}
}

func skipFlatten(dep dependency) []deptree.Dependency {
	var deps []deptree.Dependency
	for _, dd := range dep.deps {
		if dd.flatten {
			deps = append(deps, skipFlatten(dd)...)
			continue
		}
		deps = append(deps, toCommon(dd))
	}
	return deps
}

func printDep(dep dependency, level int) {
	// log.Printf("%s%s:%v - %s\n", strings.Repeat(" ", level), dep.name, dep.flatten, dep.created)
	for _, d := range dep.deps {
		printDep(d, level+1)
	}
}
