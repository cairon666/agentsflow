package flow

import (
	"github.com/cairon666/agentsflow/internal/diagnostic"
)

// LoadResult contains a normalized flow and diagnostics from template validation.
type LoadResult struct {
	Flow        Flow
	Diagnostics []diagnostic.Diagnostic
}

// LoadFile loads, validates, and normalizes a flow template file.
func LoadFile(path string) (LoadResult, error) {
	spec, err := LoadSpecFile(path)
	if err != nil {
		return LoadResult{}, err
	}
	return LoadResult{
		Flow:        Normalize(spec),
		Diagnostics: ValidateSpec(spec),
	}, nil
}
