package main

import (
	"Chirpy/internal/auth"
	"Chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
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
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
		return
	}

	chirpyDb := database.New(db)
	port := ":8080"
	apiCfg := apiConfig{}
	mux := http.NewServeMux()
	mux.Handle("/app", apiCfg.middlewareMetricsInc(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		http.ServeFile(w, req, "./index.html")
	})))

	mux.Handle("POST /api/users", apiCfg.middlewareMetricsInc(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		decoder := json.NewDecoder(req.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			returnError(w, "Something went wrong", 500)
			return
		}
		hashedPassword, err := auth.HashPassword(params.Password)
		if err != nil {
			returnError(w, "Something went wrong", 500)
			return
		}

		args := database.CreateUserParams{
			Email:          params.Email,
			HashedPassword: hashedPassword,
		}
		user, err := chirpyDb.CreateUser(req.Context(), args)
		if err != nil {
			log.Fatalf("Error creating new user: %v", err)
			returnError(w, "Error creating new user", 500)
			return
		}
		type userResp struct {
			ID        string `json:"id"`
			Email     string `json:"email"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}
		resp := userResp{
			ID:        user.ID.String(),
			Email:     user.Email,
			CreatedAt: user.CreatedAt.String(),
			UpdatedAt: user.UpdatedAt.String(),
		}
		jsonData, err := json.Marshal(resp)

		if err != nil {
			returnError(w, "Error parsing a json", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(jsonData)
	})))

	mux.Handle("POST /api/login", apiCfg.middlewareMetricsInc(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {

		var err error
		type parameters struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		decoder := json.NewDecoder(req.Body)
		params := parameters{}
		err = decoder.Decode(&params)
		if err != nil {
			returnError(w, "Something went wrong", 500)
			return
		}
		user, err := chirpyDb.GetUserByEmail(req.Context(), params.Email)
		if err != nil {
			returnError(w, "Error while getting user from db", 500)
			return
		}
		log.Printf("hahsed pass %s, email %s", user.HashedPassword, user.Email)

		err = auth.CheckPasswordHash(user.HashedPassword, params.Password)
		if err != nil {
			returnError(w, "Unautharized", 401)
			return
		}

		type userResp struct {
			ID        string `json:"id"`
			Email     string `json:"email"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}
		resp := userResp{
			ID:        user.ID.String(),
			Email:     user.Email,
			CreatedAt: user.CreatedAt.String(),
			UpdatedAt: user.UpdatedAt.String(),
		}
		jsonData, err := json.Marshal(resp)

		if err != nil {
			returnError(w, "Error parsing a json", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonData)
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
		chirpyDb.DeleteAllChirps(req.Context())
		chirpyDb.DeleteAllUsers(req.Context())
		w.WriteHeader(http.StatusOK)
		apiCfg.fileserverHits.Store(0)
	}))

	mux.Handle("POST /api/chirps", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		type parameters struct {
			Body   string `json:"body"`
			UserID string `json:"user_id"`
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
		userID, err := uuid.Parse(params.UserID)
		if err != nil {
			returnError(w, "Error parsing uuid", 500)
			return
		}

		arg := database.CreateChirpParams{
			Body:   formattedBody,
			UserID: userID,
		}
		chirp, err := chirpyDb.CreateChirp(req.Context(), arg)
		if err != nil {
			returnError(w, "Error creating chirp", 500)
			return
		}

		type userResp struct {
			Body   string `json:"body"`
			UserId string `json:"user_id"`
			Id     string `json:"id"`
		}
		resp := userResp{
			Body:   formattedBody,
			UserId: chirp.UserID.String(),
			Id:     chirp.ID.String(),
		}
		jsonData, err := json.Marshal(resp)

		if err != nil {
			returnError(w, "Error parsing a json", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(jsonData)
		apiCfg.fileserverHits.Store(0)
	}))

	mux.Handle("GET /api/chirps", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		chirps, err := chirpyDb.GetAllChrips(req.Context())
		if err != nil {
			returnError(w, "Error creating chirp", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(chirps)
		if err != nil {
			returnError(w, "Error encoding chirps", 500)
			return
		}
		apiCfg.fileserverHits.Store(0)
	}))
	mux.Handle("GET /api/chirps/{id}", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		idStr := req.PathValue("id")
		id, err := uuid.Parse(idStr)
		if err != nil {
			returnError(w, "Error while parsing id str to uuid", 500)
			return
		}
		chirp, err := chirpyDb.GetChripById(req.Context(), id)
		if err != nil {
			returnError(w, "Error creating chirp", 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(chirp)
		if err != nil {
			returnError(w, "Error encoding chirps", 500)
			return
		}
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
