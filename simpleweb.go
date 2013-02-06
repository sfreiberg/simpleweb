package main

import (
	"code.google.com/p/gorilla/mux"
	"github.com/flosch/pongo"

	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	sitesPath        = flag.String("path", "./sites", "site path")
	serveStaticFiles = flag.Bool("serve-static-files", false, "Controls whether we serve static assets")
	addr             = flag.String("addr", ":80", "Address and port to listen on")
)

func init() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
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
	templateFile := template(host, r.URL.RequestURI())
	if templateFile == "" {
		if *serveStaticFiles {
			http.FileServer(http.Dir(filepath.Join(*sitesPath, host))).ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	} else {
		ctx := pongo.Context{"host": host, "uri": r.URL.RequestURI(), "time": time.Now()}
		tmpl, err := pongo.FromFile(templateFile, nil)
		err = tmpl.ExecuteRW(w, &ctx)
		if err != nil {
			io.WriteString(w, err.Error())
		}
	}
}

func template(host, uri string) string {
	if uri == "/" {
		return path.Join(*sitesPath, host, "index.html")
	}
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
