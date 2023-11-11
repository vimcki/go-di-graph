package frontend

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed index.html
var htmlTemplate string

type dependency struct {
	ID    int
	Hash  string
	Name  string       `json:"name,omitempty"`
	Deps  []dependency `json:"deps,omitempty"`
	Value interface{}  `json:"value,omitempty"`
}

func Render(data []byte) ([]byte, error) {
	var root dependency
	if err := json.Unmarshal([]byte(data), &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph: %w", err)
	}

	return []byte(htmlTemplate), nil
}
