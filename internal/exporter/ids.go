package exporter

import (
	"fmt"
	"regexp"
	"strings"
)

var nonIDChars = regexp.MustCompile(`[^a-z0-9]+`)

// NormalizeID converts a native name or filename into a portable kebab-case id.
func NormalizeID(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = nonIDChars.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	return normalized
}

// UniqueIDs allocates stable, unique ids.
type UniqueIDs struct {
	used map[string]struct{}
}

// NewUniqueIDs creates an id allocator.
func NewUniqueIDs() *UniqueIDs {
	return &UniqueIDs{used: map[string]struct{}{}}
}

// Next returns a normalized unique id. The fallback is normalized and used when raw is empty.
func (u *UniqueIDs) Next(raw, fallback string) string {
	base := NormalizeID(raw)
	if base == "" {
		base = NormalizeID(fallback)
	}
	if base == "" {
		base = "item"
	}
	if _, ok := u.used[base]; !ok {
		u.used[base] = struct{}{}
		return base
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s-%d", base, index)
		if _, ok := u.used[candidate]; ok {
			continue
		}
		u.used[candidate] = struct{}{}
		return candidate
	}
}
