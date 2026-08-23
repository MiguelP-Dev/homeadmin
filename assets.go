// Package assets embeds the application's static files so the binary serves
// them regardless of the working directory it is launched from.
package assets

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"sync"
)

//go:embed static
var embedded embed.FS

var staticFS = sync.OnceValue(func() fs.FS {
	sub, err := fs.Sub(embedded, "static")
	if err != nil {
		// Unreachable: "static" always exists because go:embed requires it.
		panic(fmt.Sprintf("assets: sub-FS static/: %v", err))
	}
	return sub
})

// FS returns the embedded asset tree rooted at the static directory.
func FS() fs.FS {
	return staticFS()
}

// Version is a short content hash of the cacheable assets. Reference it in
// asset URLs (?v=<Version>) so deploys bust browser caches.
var Version = computeVersion()

func computeVersion() string {
	h := sha256.New()
	for _, name := range []string{"css/output.css", "js/htmx.min.js"} {
		f, err := FS().Open(name)
		if err != nil {
			panic(fmt.Sprintf("assets: opening %s for version hash: %v", name, err))
		}
		if _, err := io.Copy(h, f); err != nil {
			f.Close()
			panic(fmt.Sprintf("assets: hashing %s: %v", name, err))
		}
		if err := f.Close(); err != nil {
			panic(fmt.Sprintf("assets: closing %s: %v", name, err))
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}
