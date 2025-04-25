package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	mux.Handle("GET /admin/metrics", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "text/html")
		http.ServeFile(w, req, "./metrics.html")
		w.Write([]byte(fmt.Sprintf("<p>Chirpy has been visited %d times!</p>", apiCfg.fileserverHits.Load())))
	}))
	mux.Handle("POST /admin/reset", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		apiCfg.fileserverHits.Store(0)
	}))
	mux.Handle("POST /api/validate_chirp", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Body string `json:"body"`
		}
		decoder := json.NewDecoder(req.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			returnError(w, "Something went wrong", 500)
			return
		}
		log.Printf("len: %d", len(params.Body))
		if len(params.Body) > 140 {
			returnError(w, "Chirp is too long", 400)
			return
		}
		formattedBody := replaceWithStar(params.Body)

		returnJson(w, formattedBody)
		apiCfg.fileserverHits.Store(0)
	}))
	log.Printf("Server started on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(":8080", mux))
	// mux.Handle("/", apiHandler{})
}

func replaceWithStar(s string) string {
	words := strings.Split(s, " ")
	for i, el := range words {
		log.Printf("Idx %d, string: %s", i, el)
		lower := strings.ToLower(el)
		switch lower {
		case "kerfuffle", "sharbert", "fornax":
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func returnError(w http.ResponseWriter, error string, sc int) {
	type returnVals struct {
		Error string `json:"error"`
	}
	respBody := returnVals{
		Error: error,
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(sc)
	w.Write(dat)
	return
}

func returnJson(w http.ResponseWriter, msg string) {
	type returnVals struct {
		CleanedBody string `json:"cleaned_body"`
	}
	respBody := returnVals{
		CleanedBody: msg,
	}
	dat, err := json.Marshal(respBody)
	if err != nil {
		log.Printf("Error marshalling JSON: %s", err)
		w.WriteHeader(400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(dat)
	return
}
