package render

import (
	"fmt"

	"github.com/pelletier/go-toml/v2"
)

// TOML renders TOML bytes.
func TOML(value any) ([]byte, error) {
	data, err := toml.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("render toml: %w", err)
	}
	return data, nil
}
