package main

import (
	"fmt"
	"log"
	"net/http"
)

type apiHandler struct{}

func (apiHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	fmt.Fprintf(w, "Welcome to the home page!")
}

func main() {
	port := ":8080"
	mux := http.NewServeMux()
	mux.HandleFunc("/app", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./index.html")
	})
	mux.HandleFunc("/app/assets", func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./index.html")
		w.Write([]byte(""))
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	//http.FileServer(http.Dir("."))
	log.Printf("Server started on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(":8080", mux))
	// mux.Handle("/", apiHandler{})
}
