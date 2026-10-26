package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupSites creates a sites directory for example.com and points the
// -path and -serve-static-files flags at it for the duration of the test.
func setupSites(t *testing.T, static bool) string {
	t.Helper()
	root := t.TempDir()
	sites := filepath.Join(root, "sites")
	files := map[string]string{
		"example.com/index.html":      "home {{ host }} {{ uri }}",
		"example.com/about.html":      "about page",
		"example.com/docs/index.html": "docs index",
		"example.com/raw.html":        "raw {{ host|upper }}",
		"example.com/style.css":       "body{}",
	}
	for name, content := range files {
		p := filepath.Join(sites, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	oldPath, oldStatic := *sitesPath, *serveStaticFiles
	*sitesPath, *serveStaticFiles = sites, static
	t.Cleanup(func() { *sitesPath, *serveStaticFiles = oldPath, oldStatic })
	return root
}

func get(host, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Host = host
	rec := httptest.NewRecorder()
	handler(rec, req)
	return rec
}

func TestHandlerRendersTemplates(t *testing.T) {
	setupSites(t, false)

	tests := []struct {
		host, target string
		wantCode     int
		wantBody     string
	}{
		{"example.com", "/", 200, "home example.com /"},
		{"example.com:8080", "/", 200, "home example.com /"},
		{"example.com", "/about", 200, "about page"},
		{"example.com", "/about.html", 200, "about page"},
		{"example.com", "/docs", 200, "docs index"},
		{"example.com", "/raw", 200, "raw EXAMPLE.COM"},
		{"example.com", "/missing", 404, ""},
		{"example.com", "/style.css", 404, ""}, // static files disabled
	}
	for _, tt := range tests {
		rec := get(tt.host, tt.target)
		if rec.Code != tt.wantCode {
			t.Errorf("%s%s: code = %d, want %d", tt.host, tt.target, rec.Code, tt.wantCode)
			continue
		}
		if tt.wantBody != "" && strings.TrimSpace(rec.Body.String()) != tt.wantBody {
			t.Errorf("%s%s: body = %q, want %q", tt.host, tt.target, rec.Body.String(), tt.wantBody)
		}
	}
}

func TestHandlerServesStaticFiles(t *testing.T) {
	setupSites(t, true)

	rec := get("example.com", "/style.css")
	if rec.Code != 200 || rec.Body.String() != "body{}" {
		t.Fatalf("code = %d, body = %q", rec.Code, rec.Body.String())
	}
}

func TestTemplateLookup(t *testing.T) {
	setupSites(t, false)

	tests := map[string]string{
		"/about":       filepath.Join(*sitesPath, "example.com", "about.html"),
		"/docs":        filepath.Join(*sitesPath, "example.com", "docs", "index.html"),
		"/style.css":   "",
		"/nonexistent": "",
	}
	for uri, want := range tests {
		if got := template("example.com", uri); got != want {
			t.Errorf("template(%q) = %q, want %q", uri, got, want)
		}
	}
}
