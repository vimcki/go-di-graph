package enhancer

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/maja42/goval"
)

type Dependency struct {
	Name  string        `json:"name"`
	Deps  []*Dependency `json:"deps,omitempty"`
	Value interface{}   `json:"value,omitempty"`
}

func Enhance(configPath, treeData string) (string, error) {
	// Load config
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	var config map[string]interface{}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return "", fmt.Errorf("failed to parse config file: %w", err)
	}

	var tree Dependency

	err = json.Unmarshal([]byte(treeData), &tree)
	if err != nil {
		return "", fmt.Errorf("failed to parse tree file: %w", err)
	}

	err = enhanceTree(&tree, config)
	if err != nil {
		return "", fmt.Errorf("failed to enhance tree: %w", err)
	}

	bytes, err := json.Marshal(tree)
	if err != nil {
		return "", fmt.Errorf("failed to marshal tree: %w", err)
	}

	return string(bytes), nil
}

func enhanceTree(dep *Dependency, config map[string]interface{}) error {
	if strings.HasPrefix(dep.Name, "cfg.") {
		ctx := map[string]interface{}{
			"cfg": config,
		}
		evaluator := goval.NewEvaluator()

		var err error

		dep.Value, err = evaluator.Evaluate(dep.Name, ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to evaluate expression '%s': %w", dep.Name, err)
		}
	}

	for _, depChild := range dep.Deps {
		err := enhanceTree(depChild, config)
		if err != nil {
			return err
		}
	}

	return nil
}
