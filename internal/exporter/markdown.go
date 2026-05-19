package exporter

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// SplitFrontmatter parses optional Markdown YAML frontmatter.
func SplitFrontmatter(data []byte) (map[string]any, string, error) {
	text := string(data)
	lines := strings.SplitAfter(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return map[string]any{}, text, nil
	}

	start := len(lines[0])
	offset := start
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			frontmatter := text[start:offset]
			body := text[offset+len(line):]
			values := map[string]any{}
			if strings.TrimSpace(frontmatter) != "" {
				if err := yaml.Unmarshal([]byte(frontmatter), &values); err != nil {
					return nil, "", fmt.Errorf("parse frontmatter: %w", err)
				}
			}
			return values, body, nil
		}
		offset += len(line)
	}
	return nil, "", fmt.Errorf("frontmatter is missing closing delimiter")
}

// StringValue returns a trimmed string value from loosely typed decoded data.
func StringValue(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

// StringSliceValue returns a string slice from loosely typed decoded data.
func StringSliceValue(values map[string]any, key string) []string {
	value, ok := values[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, strings.TrimSpace(fmt.Sprint(item)))
		}
		return out
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{strings.TrimSpace(typed)}
	default:
		return nil
	}
}

// StringMapValue returns a string map from loosely typed decoded data.
func StringMapValue(values map[string]any, key string) map[string]string {
	value, ok := values[key]
	if !ok {
		return nil
	}
	typed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(typed))
	for itemKey, itemValue := range typed {
		out[itemKey] = strings.TrimSpace(fmt.Sprint(itemValue))
	}
	return out
}
