package render

import (
	"encoding/json"
	"fmt"
)

// JSON renders deterministic indented JSON.
func JSON(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("render json: %w", err)
	}
	return append(data, '\n'), nil
}
