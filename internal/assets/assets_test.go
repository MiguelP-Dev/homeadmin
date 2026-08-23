package assets_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"

	assets "github.com/homeadmin"
)

// staticApp builds a Fiber app wired exactly like cmd/server/main.go mounts
// embedded assets under /static.
func staticApp() *fiber.App {
	app := fiber.New()
	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   http.FS(assets.FS()),
		MaxAge: 31536000,
		Browse: false,
	}))
	return app
}

func TestFS_ServesEmbeddedAssetsUnderStatic(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantSubstr string
	}{
		{name: "stylesheet", path: "/static/css/output.css", wantStatus: http.StatusOK, wantSubstr: "tailwind"},
		{name: "script", path: "/static/js/htmx.min.js", wantStatus: http.StatusOK, wantSubstr: "htmx"},
		{name: "missing file", path: "/static/css/nope.css", wantStatus: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := staticApp().Test(httptest.NewRequest(http.MethodGet, tt.path, nil))
			if err != nil {
				t.Fatalf("request %s failed: %v", tt.path, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tt.wantStatus {
				t.Errorf("%s: status = %d, want %d", tt.path, resp.StatusCode, tt.wantStatus)
			}
			if tt.wantSubstr == "" {
				return
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("reading body: %v", err)
			}
			if !strings.Contains(string(body), tt.wantSubstr) {
				t.Errorf("%s: body does not contain %q", tt.path, tt.wantSubstr)
			}
		})
	}
}

func TestFS_SetsLongCacheHeader(t *testing.T) {
	resp, err := staticApp().Test(httptest.NewRequest(http.MethodGet, "/static/css/output.css", nil))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
	if got := resp.Header.Get("Cache-Control"); !strings.Contains(got, "max-age=31536000") {
		t.Errorf("Cache-Control = %q, want it to contain max-age=31536000", got)
	}
}

func TestVersion_IsTwelveCharHex(t *testing.T) {
	if len(assets.Version) != 12 {
		t.Errorf("Version length = %d, want 12", len(assets.Version))
	}
	for _, r := range assets.Version {
		isHex := (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
		if !isHex {
			t.Errorf("Version contains non-hex character %q", r)
		}
	}
}
