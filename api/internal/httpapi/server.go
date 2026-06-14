package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"splitwise-assignment/api/internal/importer"
	"splitwise-assignment/api/internal/store"
)

type Server struct {
	store *store.Memory
	mux   *http.ServeMux
}

func New(memory *store.Memory) *Server {
	server := &Server{store: memory, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.mux.HandleFunc("POST /api/imports", s.importCSV)
	s.mux.HandleFunc("POST /api/login", s.login)
	s.mux.HandleFunc("POST /api/register", s.register)
	s.mux.HandleFunc("GET /api/imports/latest", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.Report())
	})
	s.mux.HandleFunc("DELETE /api/imports/latest", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.ClearImport())
	})
	s.mux.HandleFunc("PATCH /api/imports/latest/anomalies", s.reviewAnomaly)
	s.mux.HandleFunc("GET /api/groups/default/members", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.Members())
	})
	s.mux.HandleFunc("GET /api/groups/default/expenses", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.Expenses())
	})
	s.mux.HandleFunc("GET /api/groups/default/balances", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.store.Balances())
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid login payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(payload.Email) == "" || strings.TrimSpace(payload.Password) == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"name":  "Demo Reviewer",
		"email": payload.Email,
		"token": "demo-session",
	})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid register payload", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(payload.Name)
	email := strings.TrimSpace(payload.Email)
	if name == "" || email == "" || strings.TrimSpace(payload.Password) == "" {
		http.Error(w, "name, email and password are required", http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{
		"name":  name,
		"email": email,
		"token": "demo-session",
	})
}

func (s *Server) reviewAnomaly(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		RowNumber int    `json:"rowNumber"`
		Code      string `json:"code"`
		Decision  string `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid review payload", http.StatusBadRequest)
		return
	}
	report, err := s.store.ReviewAnomaly(payload.RowNumber, payload.Code, payload.Decision)
	if err != nil {
		http.Error(w, fmt.Sprintf("review failed: %s", err), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) importCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var reader = r.Body
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
		if err := r.ParseMultipartForm(20 << 20); err != nil {
			http.Error(w, "invalid multipart body", http.StatusBadRequest)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()
		reader = file
	}

	report, err := importer.Parse(reader)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.store.ReplaceImport(report)
	writeJSON(w, http.StatusCreated, report)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
