package d2

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Dependency struct {
	ID    int
	Name  string       `json:"name,omitempty"`
	Deps  []Dependency `json:"deps,omitempty"`
	Value interface{}  `json:"value,omitempty"`
}

func generateGraphD2(dep Dependency, nodes map[int]bool, result *[]string) {
	if !nodes[dep.ID] {
		name := dep.Name
		if name == "" {
			name = "aggregate"
		}
		fixedName := strings.ReplaceAll(name, "\"", "\\\"")
		if dep.Value != nil {
			rawVal := fmt.Sprintf("%v", dep.Value)
			val := strings.ReplaceAll(rawVal, "\"", "\\\"")
			*result = append(*result, fmt.Sprintf("%d: \"%v\\n%s\"\n", dep.ID, val, fixedName))
		} else {
			*result = append(*result, fmt.Sprintf("%d: \"%s\"\n", dep.ID, fixedName))
		}
		nodes[dep.ID] = true
	}
	// Recursively traverse the dependencies and print the edges
	for _, childDep := range dep.Deps {
		generateGraphD2(childDep, nodes, result)
		*result = append(*result, fmt.Sprintf("%d -> %d", dep.ID, childDep.ID))
	}
}

func fillIDs(dep *Dependency, id *int) {
	dep.ID = *id
	*id++
	for i := range dep.Deps {
		fillIDs(&dep.Deps[i], id)
	}
}

func Render(graph string) ([]byte, error) {
	var root Dependency
	if err := json.Unmarshal([]byte(graph), &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph: %w", err)
	}

	fillIDs(&root, new(int))

	nodes := make(map[int]bool)

	var result []string

	generateGraphD2(root, nodes, &result)

	return []byte(strings.Join(result, "\n")), nil
}
