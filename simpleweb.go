package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/flosch/pongo2/v6"
	"github.com/gorilla/mux"
)

var (
	sitesPath        = flag.String("path", "./sites", "site path")
	serveStaticFiles = flag.Bool("serve-static-files", false, "Controls whether we serve static assets")
	addr             = flag.String("addr", ":80", "Address and port to listen on")
)

func main() {
	flag.Parse()

	router := mux.NewRouter()
	router.Methods("GET").HandlerFunc(handler)
	err := http.ListenAndServe(*addr, router)
	if err != nil {
		fmt.Println(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	// r.Host doesn't have a port if the default port (80) is used
	// so an error is thrown but we can just use r.Host because it
	// doesn't include the port info.
	if err != nil {
		host = r.Host
	}
	// The Host header is client controlled and becomes a directory name, so
	// reject anything that could escape the sites directory (e.g. "..").
	if !validHost(host) {
		http.Error(w, "invalid host", http.StatusBadRequest)
		return
	}
	// Use the cleaned path, not RequestURI: RequestURI includes the query
	// string (so "/about?x=1" never matched about.html) and is not cleaned.
	uri := path.Clean("/" + r.URL.Path)

	templateFile := template(host, uri)
	if templateFile == "" {
		if *serveStaticFiles {
			http.FileServer(http.Dir(filepath.Join(*sitesPath, host))).ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	} else {
		ctx := pongo2.Context{"host": host, "uri": uri, "time": time.Now()}
		tmpl, err := pongo2.FromFile(templateFile)
		if err != nil {
			log.Printf("parsing %s: %v", templateFile, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		// ExecuteWriter renders into a buffer first, so nothing has been
		// written yet if it fails.
		if err := tmpl.ExecuteWriter(ctx, w); err != nil {
			log.Printf("rendering %s: %v", templateFile, err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
	}
}

// validHost reports whether host is safe to use as a directory name below
// the sites path.
func validHost(host string) bool {
	if host == "" || host == "." || host == ".." || strings.HasPrefix(host, ".") {
		return false
	}
	return !strings.ContainsAny(host, "/\\\x00")
}

func template(host, uri string) string {
	file := path.Join(*sitesPath, host, uri)
	if templateExists(file) {
		return file
	}
	file = path.Join(*sitesPath, host, uri+".html")
	if templateExists(file) {
		return file
	}
	file = path.Join(*sitesPath, host, uri, "index.html")
	if templateExists(file) {
		return file
	}
	return ""
}

func templateExists(file string) bool {
	fi, err := os.Stat(file)
	if err == nil && !fi.IsDir() && strings.HasSuffix(file, ".html") {
		return true
	}
	return false
}
