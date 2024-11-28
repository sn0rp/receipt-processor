package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Server struct {
	config *Config
	store  *MemoryStore
	router *mux.Router
}

func NewServer(cfg *Config) *Server {
	s := &Server{
		config: cfg,
		store:  NewMemoryStore(),
		router: mux.NewRouter(),
	}

	s.router.Use(enableCORS)
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/receipts/process", s.ProcessReceipt).Methods("POST", "OPTIONS")
	s.router.HandleFunc("/receipts/{id}/points", s.GetPoints).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/receipts", s.ListReceipts).Methods("GET", "OPTIONS")
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	return http.ListenAndServe(addr, s.router)
}

func (s *Server) ProcessReceipt(w http.ResponseWriter, r *http.Request) {
	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Invalid receipt format",
		})
		return
	}

	if err := validateReceipt(&receipt); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	receipt.ID = uuid.New().String()
	points, err := calculatePoints(&receipt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to calculate points",
		})
		return
	}
	receipt.Points = points

	if err := s.store.SaveReceipt(&receipt); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to save receipt",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		ID     string `json:"id"`
		Points int64  `json:"points"`
	}{
		ID:     receipt.ID,
		Points: points,
	})
}

func (s *Server) GetPoints(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	receipt, err := s.store.GetReceipt(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Receipt not found",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Points int64 `json:"points"`
	}{
		Points: receipt.Points,
	})
}

func (s *Server) ListReceipts(w http.ResponseWriter, r *http.Request) {
	receipts, err := s.store.ListReceipts()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(receipts)
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func validateReceipt(r *Receipt) error {
	if !regexp.MustCompile(`^[\w\s\-&]+$`).MatchString(r.Retailer) {
		return fmt.Errorf("invalid retailer format")
	}

	if _, err := time.Parse("2006-01-02", r.PurchaseDate); err != nil {
		return fmt.Errorf("invalid purchase date format")
	}

	if _, err := time.Parse("15:04", r.PurchaseTime); err != nil {
		return fmt.Errorf("invalid purchase time format")
	}

	if !regexp.MustCompile(`^\d+\.\d{2}$`).MatchString(r.Total) {
		return fmt.Errorf("invalid total format")
	}

	if len(r.Items) < 1 {
		return fmt.Errorf("at least one item is required")
	}

	for _, item := range r.Items {
		if !regexp.MustCompile(`^[\w\s\-]+$`).MatchString(item.ShortDescription) {
			return fmt.Errorf("invalid item description format")
		}
		if !regexp.MustCompile(`^\d+\.\d{2}$`).MatchString(item.Price) {
			return fmt.Errorf("invalid item price format")
		}
	}

	return nil
}
