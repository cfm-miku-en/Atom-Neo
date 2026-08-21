package web

import (
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"Atom3/src/builtins"
)

type route struct {
	file string
	text string
}

var (
	routes    = make(map[string]route)
	handlers  = make(map[string]string)
	staticDir string
	notFound  string
	bindHost  string
	logging   bool
)

// The interpreter is single-threaded, so a handler and the request it can see
// are held together under one lock rather than left to race.
var (
	handlerLock sync.Mutex
	current     *http.Request
)

func requestField(read func(*http.Request) string) builtins.ValueFunc {
	return func(args []string) string {
		if current == nil {
			return ""
		}
		return read(current)
	}
}

// Go resolves extensions against the Windows registry, which regularly serves
// .js and .css as text/plain. These override that lookup.
var mimeTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".mjs":   "text/javascript; charset=utf-8",
	".json":  "application/json",
	".svg":   "image/svg+xml",
	".webp":  "image/webp",
	".avif":  "image/avif",
	".ico":   "image/x-icon",
	".woff":  "font/woff",
	".woff2": "font/woff2",
	".wasm":  "application/wasm",
	".xml":   "application/xml",
	".txt":   "text/plain; charset=utf-8",
}

func Register() {
	for ext, typ := range mimeTypes {
		mime.AddExtensionType(ext, typ)
	}

	builtins.RegisterModule(&builtins.Module{
		Name: "web",
		Funcs: map[string]builtins.Func{
			"static":   static,
			"route":    addRoute,
			"handle":   addHandler,
			"text":     addText,
			"notfound": setNotFound,
			"host":     setHost,
			"log":      setLogging,
			"listen":   listen,
		},
		Values: map[string]builtins.ValueFunc{
			"path":   requestField(func(r *http.Request) string { return r.URL.Path }),
			"method": requestField(func(r *http.Request) string { return r.Method }),
			"query": func(args []string) string {
				if current == nil || len(args) != 1 {
					return ""
				}
				return current.URL.Query().Get(args[0])
			},
		},
	})
}

func static(args []string) {
	if len(args) != 1 {
		fmt.Println("[Runtime Error]: web.static expects a directory")
		return
	}
	staticDir = args[0]
}

func addRoute(args []string) {
	if len(args) != 2 {
		fmt.Println("[Runtime Error]: web.route expects a path and a file")
		return
	}
	routes[args[0]] = route{file: args[1]}
}

// addHandler binds a path to an Atom function. The function receives the
// request path and returns the response body.
func addHandler(args []string) {
	if len(args) != 2 {
		fmt.Println("[Runtime Error]: web.handle expects a path and a function")
		return
	}
	handlers[args[0]] = args[1]
}

func addText(args []string) {
	if len(args) != 2 {
		fmt.Println("[Runtime Error]: web.text expects a path and some text")
		return
	}
	routes[args[0]] = route{text: args[1]}
}

func setNotFound(args []string) {
	if len(args) != 1 {
		fmt.Println("[Runtime Error]: web.notfound expects a file")
		return
	}
	notFound = args[0]
}

func setHost(args []string) {
	if len(args) != 1 {
		fmt.Println("[Runtime Error]: web.host expects an address")
		return
	}
	bindHost = args[0]
}

func setLogging(args []string) {
	if len(args) != 1 {
		fmt.Println("[Runtime Error]: web.log expects true or false")
		return
	}
	logging = args[0] == "true"
}

// resolve maps a request path to a file inside staticDir, falling back to
// index.html for directories and to name.html so /about serves about.html.
func resolve(urlPath string) (string, bool) {
	if staticDir == "" {
		return "", false
	}

	root, err := filepath.Abs(staticDir)
	if err != nil {
		return "", false
	}

	clean := filepath.Clean("/" + strings.TrimPrefix(urlPath, "/"))
	full, err := filepath.Abs(filepath.Join(root, clean))
	if err != nil || (full != root && !strings.HasPrefix(full, root+string(os.PathSeparator))) {
		return "", false
	}

	if info, err := os.Stat(full); err == nil {
		if !info.IsDir() {
			return full, true
		}
		index := filepath.Join(full, "index.html")
		if _, err := os.Stat(index); err == nil {
			return index, true
		}
		return "", false
	}

	if !strings.HasSuffix(urlPath, "/") {
		if _, err := os.Stat(full + ".html"); err == nil {
			return full + ".html", true
		}
	}
	return "", false
}

func cacheFor(w http.ResponseWriter, file string) {
	if strings.HasSuffix(file, ".html") {
		w.Header().Set("Cache-Control", "no-cache")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
}

func serveMissing(w http.ResponseWriter, r *http.Request) {
	if notFound != "" {
		if file, ok := resolve(notFound); ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusNotFound)
			body, err := os.ReadFile(file)
			if err == nil {
				w.Write(body)
				return
			}
		}
	}
	http.NotFound(w, r)
}

func handle(w http.ResponseWriter, r *http.Request) {
	if fn, ok := handlers[r.URL.Path]; ok {
		if builtins.Call == nil {
			http.Error(w, "handlers unavailable", http.StatusInternalServerError)
			return
		}

		handlerLock.Lock()
		current = r
		body, called := builtins.Call(fn, []string{r.URL.Path})
		current = nil
		handlerLock.Unlock()

		if !called {
			http.Error(w, "no such handler: "+fn, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, body)
		return
	}

	if rt, ok := routes[r.URL.Path]; ok {
		if rt.file != "" {
			if file, ok := resolve(rt.file); ok {
				cacheFor(w, file)
				http.ServeFile(w, r, file)
				return
			}
			serveMissing(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, rt.text)
		return
	}

	if file, ok := resolve(r.URL.Path); ok {
		cacheFor(w, file)
		http.ServeFile(w, r, file)
		return
	}

	serveMissing(w, r)
}

type recorder struct {
	http.ResponseWriter
	status int
}

func (r *recorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !logging {
			next(w, r)
			return
		}
		start := time.Now()
		rec := &recorder{ResponseWriter: w, status: http.StatusOK}
		next(rec, r)
		fmt.Printf("%s %s %d %s\n", r.Method, r.URL.Path, rec.status, time.Since(start).Round(time.Microsecond))
	}
}

func listen(args []string) {
	if len(args) != 1 {
		fmt.Println("[Runtime Error]: web.listen expects a port")
		return
	}
	port, err := strconv.Atoi(args[0])
	if err != nil {
		fmt.Printf("[Runtime Error]: '%s' is not a valid port\n", args[0])
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", withLogging(handle))

	addr := fmt.Sprintf("%s:%d", bindHost, port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	shown := bindHost
	if shown == "" {
		shown = "localhost"
	}
	fmt.Printf("serving on http://%s:%d\n", shown, port)

	if err := srv.ListenAndServe(); err != nil {
		fmt.Printf("[Runtime Error]: %v\n", err)
	}
}
