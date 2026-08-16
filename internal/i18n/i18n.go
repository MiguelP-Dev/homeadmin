// Package i18n provides a small stdlib-only translation layer with English
// and Spanish dictionaries plus locale-aware display helpers.
package i18n

import "fmt"

// dict returns the dictionary for lang. Any language other than "es"
// resolves to English (design D1/D5: default "en").
func dict(lang string) map[string]string {
	if lang == "es" {
		return es
	}
	return en
}

// T returns the translated string for key in lang. A missing key returns the
// key itself so translation gaps fail loud in tests and in the UI (design D2).
func T(lang, key string) string {
	if v, ok := dict(lang)[key]; ok {
		return v
	}
	return key
}

// Tf returns the translated string for key in lang with fmt-style arguments
// interpolated into the template (REQ-I18N-2). A missing key returns the key
// itself; without args the value is returned verbatim.
func Tf(lang, key string, args ...any) string {
	v, ok := dict(lang)[key]
	if !ok {
		return key
	}
	if len(args) == 0 {
		return v
	}
	return fmt.Sprintf(v, args...)
}
