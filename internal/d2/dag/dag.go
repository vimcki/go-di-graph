package dag

import (
	"encoding/json"
	"fmt"
	"strings"
)

var builtins = []string{
	"append",
	"",
}

type dependency struct {
	ID    int
	Name  string       `json:"name,omitempty"`
	Deps  []dependency `json:"deps,omitempty"`
	Value interface{}  `json:"value,omitempty"`
}

func Render(graph string) ([]byte, error) {
	var root dependency
	if err := json.Unmarshal([]byte(graph), &root); err != nil {
		return nil, fmt.Errorf("failed to unmarshal graph: %w", err)
	}

	rootID := 1

	fillIDs(&root, &rootID, make(map[string]int))

	var result []string

	generateGraphD2(root, make(map[int]bool), &result)

	filterDuplicates(&result)

	return []byte(strings.Join(result, "\n")), nil
}

func fillIDs(dep *dependency, id *int, nameToID map[string]int) {
	oldID, ok := nameToID[dep.Name]
	if ok && !in(builtins, dep.Name) && !isLiteral(dep.Name) {
		dep.ID = oldID
	} else {
		dep.ID = *id
		nameToID[dep.Name] = *id
		*id++
	}
	for i := range dep.Deps {
		fillIDs(&dep.Deps[i], id, nameToID)
	}
}

func generateGraphD2(dep dependency, nodes map[int]bool, result *[]string) {
	if dep.ID == 0 {
		// skip repeated nodes
		return
	}
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

func filterDuplicates(result *[]string) {
	seen := make(map[string]bool)
	var filtered []string
	for _, line := range *result {
		if !seen[line] {
			seen[line] = true
			filtered = append(filtered, line)
		}
	}
	*result = filtered
}

func in(list []string, elem string) bool {
	for _, e := range list {
		if e == elem {
			return true
		}
	}
	return false
}

func isLiteral(x string) bool {
	if strings.HasPrefix(x, "\"") && strings.HasSuffix(x, "\"") {
		return true
	}
	return false
}
