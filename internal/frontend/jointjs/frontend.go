package jointjs

import (
	_ "embed"
	"strings"
)

//go:embed index.html
var htmlTemplate string

//go:embed md5.js
var md5 string

func Render(data []byte) ([]byte, error) {
	filled := strings.ReplaceAll(htmlTemplate, "{{GRAPH_DATA}}", string(data))
	filled = strings.ReplaceAll(filled, "{{MD5}}", md5)

	return []byte(filled), nil
}
