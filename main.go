package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) incMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset: utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(
		`
		<html>
			<body>
				<h1>Welcome, Chirpy Admin</h1>
				<p>Chirpy has been visited %d times!</p>
			</body>
		</html>
		`, cfg.fileserverHits.Load(),
	)))
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func main() {
	apiCfg := apiConfig{}
	handler := http.NewServeMux()
	handler.Handle("GET /app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	handler.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/api/healthz" {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	handler.HandleFunc("GET /admin/metrics", apiCfg.incMetrics)
	handler.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)
	handler.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		type reqParams struct {
			Body string `json:"body"`
		}

		var reqBody reqParams
		decoder := json.NewDecoder(req.Body)
		err := decoder.Decode(&reqBody)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Err bad request")
			return
		}
		if len(reqBody.Body) > 140 {
			respondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}

		cleanedBody := ""
		for _, word := range strings.Fields(reqBody.Body) {
			switch strings.ToLower(word) {
			case "kerfuffle":
				fallthrough
			case "sharbert":
				fallthrough
			case "fornax":
				if len(cleanedBody) > 0 {
					cleanedBody += " ****"
				} else {
					cleanedBody += "****"
				}
			default:
				if len(cleanedBody) > 0 {
					cleanedBody += " " + word
				} else {
					cleanedBody += word
				}
			}
		}

		respondWithSuccess(w, cleanedBody)
	})
	
	server := &http.Server{
		Addr: ":8080",
		Handler: handler,
	}
	fmt.Printf("Server is running and listening on http://localhost%s ...\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	type ErrorResponse struct {
		Error string `json:"error"`
	}

	w.Header().Set("Content-Type", "application/json")
	errResp := ErrorResponse{
		Error: msg,
	}
	errRespEncoded, err := json.Marshal(errResp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Error on marshaling error response: ` + msg + `"}`))
		return
	}

	w.WriteHeader(code)
	w.Write(errRespEncoded)
}

func respondWithSuccess(w http.ResponseWriter, msg string) {
	type resVal struct {
		CleanedBody string `json:"cleaned_body"`
	}

	resp := resVal{
		CleanedBody: msg,
	}
	respEncoded, err := json.Marshal(resp)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Error on marshaling success response")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respEncoded)
}
