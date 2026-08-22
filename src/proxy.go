package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

type proxyOpts struct {
	routes    map[string]string
	fallback  string
	listen    string
	tls       bool
	acceptTOS bool
	certDir   string
}

func parseProxyArgs(args []string) proxyOpts {
	o := proxyOpts{routes: map[string]string{}, listen: "9144", certDir: "atom-certs"}

	for i := 0; i < len(args); i++ {
		value, hasValue := "", i+1 < len(args)
		if hasValue {
			value = args[i+1]
		}

		switch args[i] {
		case "--to":
			if hasValue {
				o.fallback = value
				i++
			}
		case "--route":
			if hasValue {
				if host, target, found := strings.Cut(value, "="); found {
					o.routes[strings.ToLower(strings.TrimSpace(host))] = strings.TrimSpace(target)
				} else {
					fmt.Printf("[Proxy Error]: --route wants host=target, got '%s'\n", value)
				}
				i++
			}
		case "--listen":
			if hasValue {
				o.listen = value
				i++
			}
		case "--certs":
			if hasValue {
				o.certDir = value
				i++
			}
		case "--tls":
			o.tls = true
		case "--accept-tos":
			o.acceptTOS = true
		}
	}
	return o
}

func proxyUsage() {
	fmt.Println("Usage: atom proxy --to host:port [--listen port]")
	fmt.Println("       atom proxy --route example.com=localhost:9143 [--route api.example.com=localhost:9000]")
	fmt.Println("       atom proxy --route example.com=localhost:9143 --tls --accept-tos")
	fmt.Println()
	fmt.Println("--route sends a domain to its own target; --to catches anything unmatched.")
	fmt.Println("--tls serves https on 443 with certificates from Let's Encrypt and")
	fmt.Println("redirects port 80. It needs at least one --route and --accept-tos.")
}

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "https://"+r.Host+r.URL.RequestURI(), http.StatusMovedPermanently)
}

func newProxy(target *url.URL, scheme string) *httputil.ReverseProxy {
	rp := httputil.NewSingleHostReverseProxy(target)

	// ReverseProxy already appends X-Forwarded-For; the origin also needs to
	// know which name and scheme the client actually asked for.
	inner := rp.Director
	rp.Director = func(r *http.Request) {
		host := r.Host

		// Whatever the client claimed is dropped first. ReverseProxy appends
		// the address it actually came from, so keeping the old value would
		// let anyone put any address at the front of the chain.
		r.Header.Del("X-Forwarded-For")

		inner(r)
		r.Header.Set("X-Forwarded-Host", host)
		r.Header.Set("X-Forwarded-Proto", scheme)
	}

	rp.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		fmt.Printf("[Proxy Error]: %v\n", err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}
	return rp
}

func targetURL(target string) (*url.URL, error) {
	if !strings.Contains(target, "://") {
		target = "http://" + target
	}
	return url.Parse(target)
}

type router struct {
	byHost   map[string]http.Handler
	fallback http.Handler
}

func (rt *router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := strings.ToLower(r.Host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	if h, ok := rt.byHost[host]; ok {
		h.ServeHTTP(w, r)
		return
	}
	if rt.fallback != nil {
		rt.fallback.ServeHTTP(w, r)
		return
	}
	http.Error(w, "no route for "+host, http.StatusNotFound)
}

func buildRouter(o proxyOpts, scheme string) (*router, []string, bool) {
	rt := &router{byHost: map[string]http.Handler{}}
	var hosts []string

	for host, target := range o.routes {
		u, err := targetURL(target)
		if err != nil {
			fmt.Printf("[Proxy Error]: bad target '%s' for %s: %v\n", target, host, err)
			return nil, nil, false
		}
		rt.byHost[host] = newProxy(u, scheme)
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)

	if o.fallback != "" {
		u, err := targetURL(o.fallback)
		if err != nil {
			fmt.Printf("[Proxy Error]: bad target '%s': %v\n", o.fallback, err)
			return nil, nil, false
		}
		rt.fallback = newProxy(u, scheme)
	}
	return rt, hosts, true
}

func server(addr string, h http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
}

func proxy(args []string) {
	o := parseProxyArgs(args)

	if o.fallback == "" && len(o.routes) == 0 {
		proxyUsage()
		return
	}

	if !o.tls {
		rt, hosts, ok := buildRouter(o, "http")
		if !ok {
			return
		}

		addr := ":" + o.listen
		for _, h := range hosts {
			fmt.Printf("%s -> %s\n", h, o.routes[h])
		}
		if o.fallback != "" {
			fmt.Printf("anything else -> %s\n", o.fallback)
		}
		fmt.Printf("proxying on http://localhost%s\n", addr)

		if err := server(addr, rt).ListenAndServe(); err != nil {
			fmt.Printf("[Proxy Error]: %v\n", err)
		}
		return
	}

	// Without a host whitelist any name pointed at this box could trigger a
	// certificate request, which burns Let's Encrypt rate limits.
	if len(o.routes) == 0 {
		fmt.Println("[Proxy Error]: --tls needs at least one --route to say which domains to serve")
		return
	}

	// Issuing a certificate accepts Let's Encrypt's subscriber agreement, so
	// that is the operator's call to make rather than something done silently.
	if !o.acceptTOS {
		fmt.Println("[Proxy Error]: --tls requests certificates from Let's Encrypt, which")
		fmt.Println("accepts their subscriber agreement on your behalf. Pass --accept-tos")
		fmt.Println("to confirm: https://letsencrypt.org/repository/")
		return
	}

	rt, hosts, ok := buildRouter(o, "https")
	if !ok {
		return
	}

	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(hosts...),
		Cache:      autocert.DirCache(o.certDir),
	}

	go func() {
		if err := http.ListenAndServe(":80", manager.HTTPHandler(http.HandlerFunc(redirectToHTTPS))); err != nil {
			fmt.Printf("[Proxy Error]: port 80: %v\n", err)
		}
	}()

	for _, h := range hosts {
		fmt.Printf("https://%s -> %s\n", h, o.routes[h])
	}
	fmt.Printf("certificates cached in %s\n", o.certDir)

	srv := server(":443", rt)
	srv.TLSConfig = manager.TLSConfig()

	if err := srv.ListenAndServeTLS("", ""); err != nil {
		fmt.Printf("[Proxy Error]: %v\n", err)
	}
}
