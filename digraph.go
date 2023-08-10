package digraph

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"

	"github.com/vimcki/go-di-graph/internal/d2"
	"github.com/vimcki/go-di-graph/internal/depcalc"
	"github.com/vimcki/go-di-graph/internal/encoder"
	"github.com/vimcki/go-di-graph/internal/enhancer"
	"github.com/vimcki/go-di-graph/internal/flatten"
)

type Digraph struct {
	config     string
	dir        string
	graph      []byte
	entrypoint string
	buildFS    fs.FS
	err        error
	lock       sync.RWMutex
}

func New(config interface{}, entrypoint string, buildFS fs.FS) *Digraph {
	cfgString, err := encoder.Marshal(config)
	if err != nil {
		panic(err)
	}
	return &Digraph{config: string(cfgString), entrypoint: entrypoint, buildFS: buildFS}
}

func (d *Digraph) Close() error {
	return os.RemoveAll(d.dir)
}

func (d *Digraph) Handler() func() ([]byte, string) {
	go func() {
		err := d.buildGraph()
		if err != nil {
			d.lock.Lock()

			defer d.lock.Unlock()

			d.err = err
		}
	}()

	return d.handle
}

func (d *Digraph) buildGraph() error {
	_, err := exec.LookPath("d2")
	if err != nil {
		return fmt.Errorf("d2 binary is not available: %w", err)
	}

	dir, err := os.MkdirTemp("/tmp/", "digraph")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	d.dir = dir

	err = d.unpackBuildFS()
	if err != nil {
		return fmt.Errorf("failed to unpack build fs: %w", err)
	}

	graph, err := d.build()
	if err != nil {
		return fmt.Errorf("failed to build graph: %w", err)
	}

	d.lock.Lock()
	defer d.lock.Unlock()

	d.graph = graph

	return nil
}

func (d *Digraph) handle() ([]byte, string) {
	d.lock.RLock()
	defer d.lock.RUnlock()

	if d.err != nil {
		return []byte(d.err.Error()), "text/plain"
	}

	graph := d.graph

	if graph == nil {
		return []byte("building..."), "text/plain"
	}

	return graph, "image/svg+xml"
}

func (d *Digraph) build() ([]byte, error) {
	err := os.Mkdir(d.dir+"/flat", 0o755)
	if err != nil {
		return nil, fmt.Errorf("failed to create build dir: %w", err)
	}

	configPath := path.Join(d.dir, "/config.json")

	err = os.WriteFile(configPath, []byte(d.config), 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write config file: %w", err)
	}

	flatten.Flatten(d.dir, "build", "flat", d.entrypoint, configPath)

	deptree, err := depcalc.Depcalc(d.entrypoint, path.Join(d.dir, "/flat"))
	if err != nil {
		return nil, fmt.Errorf("failed to calculate dependencies: %w", err)
	}

	err = os.WriteFile(path.Join(d.dir, "/deptree.json"), []byte(deptree), 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write deptree file: %w", err)
	}

	enhanced, err := enhancer.Enhance(configPath, deptree)
	if err != nil {
		return nil, fmt.Errorf("failed to enhance tree: %w", err)
	}

	err = os.WriteFile(path.Join(d.dir, "/enhanced.json"), []byte(enhanced), 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write enhanced file: %w", err)
	}

	d2Graph, err := d2.Render(enhanced)
	if err != nil {
		return nil, fmt.Errorf("failed to render d2 graph: %w", err)
	}

	renderPath := path.Join(d.dir, "/render.d2")

	err = os.WriteFile(renderPath, d2Graph, 0o644)
	if err != nil {
		return nil, fmt.Errorf("failed to write d2 file: %w", err)
	}

	svgPath := path.Join(d.dir, "/render.svg")

	cmd := exec.Command("d2", "--layout=elk", renderPath, svgPath)

	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to run d2: %w", err)
	}

	bytes, err := os.ReadFile(svgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return bytes, nil
}

func (d *Digraph) unpackBuildFS() error {
	err := os.Mkdir(d.dir+"/build", 0o755)
	if err != nil {
		return fmt.Errorf("failed to create build dir: %w", err)
	}

	err = fs.WalkDir(d.buildFS, ".", func(path string, e fs.DirEntry, _ error) error {
		if !isLevelDeep(path) {
			return nil
		}
		if e.IsDir() {
			return nil
		}
		file, err := d.buildFS.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}

		bytes := make([]byte, 1024)

		i, err := file.Read(bytes)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}

		bytes = bytes[:i]

		err = os.WriteFile(d.dir+"/build/"+path, bytes, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to walk build fs: %w", err)
	}

	return nil
}

func isLevelDeep(path string) bool {
	dir, _ := filepath.Split(path)
	return dir == ""
}
