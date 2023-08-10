package main

import (
	"flag"
	"os"

	"github.com/vimcki/go-di-graph/internal/enhancer"
)

func main() {
	configPath := flag.String("config_path", "", "Path to config file")
	treePath := flag.String("tree_path", "", "Path to tree file")

	flag.Parse()

	data, err := os.ReadFile(*treePath)
	if err != nil {
		os.Exit(1)
	}

	rusult, err := enhancer.Enhance(*configPath, string(data))
	if err != nil {
		os.Exit(1)
	}

	os.Stdout.Write([]byte(rusult))
}
