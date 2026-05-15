package render

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Frontmatter renders Markdown YAML frontmatter from string values and lists.
func Frontmatter(values map[string]any) string {
	var b strings.Builder
	b.WriteString("---\n")
	data, err := yaml.Marshal(values)
	if err != nil {
		fmt.Fprintf(&b, "error: %q\n", err.Error())
	} else {
		b.Write(data)
	}
	b.WriteString("---\n\n")
	return b.String()
}
