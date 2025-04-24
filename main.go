package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	fmt.Fprintf(w, "Welcome to the home page!")
}

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, req)
	})
}

func main() {
	port := ":8080"
	apiCfg := apiConfig{}
	mux := http.NewServeMux()
	mux.Handle("/app", apiCfg.middlewareMetricsInc(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./index.html")
	})))
	mux.Handle("/app/assets/", apiCfg.middlewareMetricsInc(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./index.html")
		w.Write([]byte(""))
	})))
	mux.Handle("GET /api/healthz", apiCfg.middlewareMetricsInc(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})))
	mux.Handle("GET /api/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Hits: %d", apiCfg.fileserverHits.Load())))
	}))
	mux.Handle("POST /api/reset", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		apiCfg.fileserverHits.Store(0)
	}))
	log.Printf("Server started on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(":8080", mux))
	// mux.Handle("/", apiHandler{})
}
