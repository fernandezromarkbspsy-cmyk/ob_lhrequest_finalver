package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func main() {
	dir := os.Getenv("FRONTEND_DIR")
	if dir == "" {
		dir = "frontend-react/dist"
	}

	port := os.Getenv("FRONTEND_PORT")
	if port == "" {
		port = "5173"
	}

	host := os.Getenv("FRONTEND_HOST")
	if host == "" {
		host = "127.0.0.1"
	}

	mux := http.NewServeMux()
	if proxy := apiProxy(); proxy != nil {
		mux.Handle("/api/", proxy)
		mux.Handle("/api", proxy)
		mux.Handle("/healthz", proxy)
	}
	mux.Handle("/", spaFileServer(http.Dir(dir)))

	addr := host + ":" + port
	log.Println("Frontend running on", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func apiProxy() http.Handler {
	target := os.Getenv("FRONTEND_API_URL")
	if target == "" {
		target = "http://127.0.0.1:8080"
	}
	targetURL, err := url.Parse(target)
	if err != nil {
		log.Println("API proxy disabled:", err)
		return nil
	}
	return httputil.NewSingleHostReverseProxy(targetURL)
}

func spaFileServer(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = "/index.html"
		}
		if strings.HasPrefix(r.URL.Path, "/static/") || strings.HasPrefix(r.URL.Path, "/truck_label/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(w, r)
	})
}
