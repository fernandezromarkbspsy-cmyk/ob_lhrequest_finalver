package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	dir := os.Getenv("FRONTEND_DIR")
	if dir == "" {
		dir = "frontend"
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
	mux.Handle("/", spaFileServer(http.Dir(dir)))

	addr := host + ":" + port
	log.Println("Frontend running on", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func spaFileServer(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			r.URL.Path = "/index.html"
		}
		fileServer.ServeHTTP(w, r)
	})
}
