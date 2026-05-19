package flow

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// EncodeSpec renders a template spec as YAML.
func EncodeSpec(spec Spec) ([]byte, error) {
	data, err := yaml.Marshal(spec)
	if err != nil {
		return nil, fmt.Errorf("render yaml: %w", err)
	}
	return data, nil
}
