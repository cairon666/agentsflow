package flow

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// DecodeSpec parses a template YAML document with strict known fields.
func DecodeSpec(data []byte) (Spec, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var spec Spec
	if err := dec.Decode(&spec); err != nil {
		return Spec{}, fmt.Errorf("parse yaml: %w", err)
	}
	return spec, nil
}

// LoadSpecFile reads and decodes a template spec file.
func LoadSpecFile(path string) (Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Spec{}, fmt.Errorf("read template %q: %w", path, err)
	}
	spec, err := DecodeSpec(data)
	if err != nil {
		return Spec{}, fmt.Errorf("decode template %q: %w", path, err)
	}
	return spec, nil
}
