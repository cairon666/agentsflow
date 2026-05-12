package binding

// Models maps model slot names to user-selected model IDs.
type Models map[string]string

// Resolve returns the selected model for slot, following fallback slots.
func (m Models) Resolve(slot string, fallbacks map[string]string) string {
	if model := m[slot]; model != "" {
		return model
	}
	seen := map[string]struct{}{slot: {}}
	for next := fallbacks[slot]; next != ""; next = fallbacks[next] {
		if _, ok := seen[next]; ok {
			return ""
		}
		seen[next] = struct{}{}
		if model := m[next]; model != "" {
			return model
		}
	}
	return ""
}
