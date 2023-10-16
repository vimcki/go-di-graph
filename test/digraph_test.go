package tests

import (
	"embed"
	"encoding/json"
	"io/fs"
	"os"
	"reflect"
	"testing"

	digraph "github.com/vimcki/go-di-graph"
	"github.com/vimcki/go-di-graph/internal/d2"
	"github.com/wI2L/jsondiff"
)

type Test struct {
	name       string
	entrypoint string
	fs         embed.FS
	path       string
}

//go:embed modules/test1/build
var test1 embed.FS

//go:embed modules/test_set_build/build
var test2 embed.FS

func TestDigraph(t *testing.T) {
	tests := []Test{
		{
			name:       "test1",
			entrypoint: "Build",
			fs:         test1,
			path:       "modules/test1/",
		},
		{
			name:       "test_set_build",
			entrypoint: "Build",
			fs:         test2,
			path:       "modules/test_set_build/",
		},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			realFS, err := fs.Sub(test.fs, test.path+"build")
			if err != nil {
				t.Fatal(err)
			}

			bytes, err := os.ReadFile(test.path + "config.json")
			if err != nil {
				t.Fatal(err)
			}

			var config interface{}

			err = json.Unmarshal(bytes, &config)
			if err != nil {
				t.Fatal(err)
			}

			dg, err := digraph.New(
				config,
				test.entrypoint,
				realFS,
				digraph.WithNoRender(),
				digraph.WithBlockingHandler(),
			)
			if err != nil {
				t.Fatal(err)
			}

			handler := dg.Handler()

			result, contentType := handler()

			if contentType != "application/json" {
				t.Log(dg.Dir())
				t.Log(string(result))
				t.Fatalf("expected application/json, got %s", contentType)
			}

			var got map[string]interface{}

			err = json.Unmarshal(result, &got)
			if err != nil {
				t.Fatal(err)
			}

			data, err := os.ReadFile(test.path + "result.json")
			if err != nil {
				t.Fatal(err)
			}

			var want map[string]interface{}

			err = json.Unmarshal(data, &want)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, want) {
				t.Log(dg.Dir())
				pretty, err := json.MarshalIndent(got, "", "  ")
				if err != nil {
					t.Fatal(err)
				}

				// t.Log(string(pretty))

				writeD2(t, pretty)

				printDiff(t, want, got)
				t.Fatalf("%v failed", test.name)
			}
		})
	}
}

func printDiff(t *testing.T, want, got map[string]interface{}) {
	patches, err := jsondiff.Compare(got, want)
	if err != nil {
		t.Fatalf("jsondiff error: %v", err)
	}
	for _, patch := range patches {
		t.Logf("%v", patch)
	}
}

func writeD2(t *testing.T, data []byte) {
	data, err := d2.Render(string(data))
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile("render.d2", []byte(data), 0o644)
	if err != nil {
		t.Fatal(err)
	}
}
