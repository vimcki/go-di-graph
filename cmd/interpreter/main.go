package main

import (
	"fmt"
	"strings"
	"testing/fstest"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

var testFilesystem = fstest.MapFS{
	"main.go": &fstest.MapFile{
		Data: []byte(`package main

import (
	"fmt"
	"github.com/project/internal/build"
	"github.com/project/internal/config"
)

func main() {
		cfg := config.Config{
			Strategy: "test",
		}

		
		processor, err := build.Build(cfg)
		if err != nil {
			panic(err)
		}
		fmt.Println(processor)
}
`),
	},
	"_pkg/src/github.com/project/internal/config/config.go": &fstest.MapFile{
		Data: []byte(`
package config

import (
	"fmt"
)

type Config struct {
	Strategy string
}
`),
	},
	"_pkg/src/github.com/project/internal/build/build.go": &fstest.MapFile{
		Data: []byte(`
package build

import (
	"fmt"
	"github.com/project/internal/impl"
	"github.com/project/internal/config"
)

type Processor interface {
	Process() (string, error)
}

func Build(cfg config.Config) (impl.Dependency, error) {
	return impl.New(cfg.Strategy), nil
}
`),
	},
	"_pkg/src/github.com/project/internal/impl/processor.go": &fstest.MapFile{
		Data: []byte(`
package impl

type Dependency struct {
	Name         string
	Deps         []Dependency
	Flatten      bool
	Created      string
	ImportedFrom string
}

func New(args ...interface{}) Dependency {
	var deps []Dependency

	for _, arg := range args {
		switch argT := arg.(type) {
		case string:
			deps = append(deps, Dependency{
					Name: argT,
			})

		case Dependency:
			deps = append(deps, argT)
		}
	}

	return Dependency{
		Name:         "impl.New",
		Deps:         deps,
		Flatten:      false,
		ImportedFrom: "github.com/project/internal/impl",
	}
}
`),
	},
}

const buildHarnessTemplate = `
	func buildHarness() []byte {
		cfgRepr := {{CFG}}

		var cfg config.Config

		if err := json.Unmarshal([]byte(cfgRepr), &cfg); err != nil {
			panic(err)
		}

		components, err := build.Build(cfg)
		if err != nil {
			panic(err)
		}

		fmt.Println(components)

		bytes, err := json.Marshal(components)
		if err != nil {
			panic(err)
		}

		return bytes
	}
`

type Config struct {
	Strategy string
}

func main() {
	i := interp.New(interp.Options{
		GoPath:               "./_pkg",
		SourcecodeFilesystem: testFilesystem,
	})

	cfg := `"{\"Strategy\": \"test\"}"`

	buildHarness := strings.Replace(buildHarnessTemplate, "{{CFG}}", cfg, 1)

	if err := i.Use(stdlib.Symbols); err != nil {
		panic(err)
	}

	imports := `
import (
	"fmt"
	"encoding/json"

	"github.com/project/internal/build"
	"github.com/project/internal/config"
	"github.com/project/internal/impl"
	)
	`

	_, err := i.Eval(imports)
	if err != nil {
		panic(err)
	}

	_, err = i.Eval(buildHarness)
	if err != nil {
		panic(err)
	}

	v, err := i.Eval("buildHarness()")
	if err != nil {
		panic(err)
	}

	bytes, ok := v.Interface().([]byte)
	if !ok {
		panic("not ok")
	}

	fmt.Println(string(bytes))
}
