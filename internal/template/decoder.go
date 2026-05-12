package template

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Decode parses a template YAML document with strict known fields.
func Decode(data []byte) (Flow, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var flow Flow
	if err := dec.Decode(&flow); err != nil {
		return Flow{}, fmt.Errorf("parse yaml: %w", err)
	}
	return flow, nil
}
