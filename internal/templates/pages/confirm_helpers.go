package pages

import "github.com/a-h/templ"

// confirmAttrs builds the translated onclick confirmation dialog for delete
// forms. Spread the result into the element tag with
// `{ confirmAttrs(msg)... }` so templ renders a real onclick attribute;
// assigning it via `attributes={ ... }` would stringify the map instead of
// spreading it.
func confirmAttrs(msg string) templ.Attributes {
	return templ.Attributes(map[string]any{"onclick": "return confirm('" + msg + "')"})
}
