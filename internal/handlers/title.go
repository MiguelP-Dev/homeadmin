package handlers

import (
	"fmt"

	"github.com/homeadmin/internal/i18n"
)

// pageTitle builds the localized browser tab title for a page, appending the
// product suffix exactly like the previous hardcoded titles ("X — HomeAdmin").
func pageTitle(lang, key string) string {
	return fmt.Sprintf("%s — HomeAdmin", i18n.T(lang, key))
}
