package d2

import "github.com/vimcki/go-di-graph/internal/d2/tree"

func Render(graph string) ([]byte, error) {
	return tree.Render(graph)
}
