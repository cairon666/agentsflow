package template

import (
	"fmt"
	"os"
)

// LoadFile reads and decodes a template file.
func LoadFile(path string) (Flow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Flow{}, fmt.Errorf("read template %q: %w", path, err)
	}
	flow, err := Decode(data)
	if err != nil {
		return Flow{}, fmt.Errorf("decode template %q: %w", path, err)
	}
	return flow, nil
}
