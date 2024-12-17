package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
)

type Server struct {
	config *Config
	store  *PostgresStore
	router *mux.Router
}

func NewServer(cfg *Config) (*Server, error) {
	if err := runMigrations(cfg); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %v", err)
	}

	store, err := NewPostgresStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	s := &Server{
		config: cfg,
		store:  store,
		router: mux.NewRouter(),
	}

	s.router.Use(enableCORS)
	s.setupRoutes()
	return s, nil
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

	// Check for duplicates
	isDuplicate, err := s.isDuplicateReceipt(&receipt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Failed to check for duplicate receipt",
		})
		return
	}
	if isDuplicate {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "This receipt has already been processed",
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

func (s *Server) isDuplicateReceipt(receipt *Receipt) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1 FROM receipts r
			WHERE r.retailer = $1
			AND r.purchase_date = $2
			AND r.purchase_time = $3
			AND r.total = $4
			AND (
				SELECT string_agg(short_description || ':' || price, ',' ORDER BY short_description)
				FROM items
				WHERE items.receipt_id = r.id
			) = (
				SELECT string_agg(d || ':' || p, ',' ORDER BY d)
				FROM unnest($5::text[], $6::text[]) AS t(d, p)
			)
		)
	`

	// Prepare the items arrays
	itemDescs := make([]string, len(receipt.Items))
	itemPrices := make([]string, len(receipt.Items))
	for i, item := range receipt.Items {
		itemDescs[i] = item.ShortDescription
		itemPrices[i] = item.Price
	}

	err := s.store.db.QueryRow(
		query,
		receipt.Retailer,
		receipt.PurchaseDate,
		receipt.PurchaseTime,
		receipt.Total,
		pq.Array(itemDescs),
		pq.Array(itemPrices),
	).Scan(&exists)

	return exists, err
}
