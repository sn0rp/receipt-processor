package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(cfg *Config) (*PostgresStore, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error connecting to the database: %v", err)
	}

	return &PostgresStore{db: db}, nil
}

func (s *PostgresStore) SaveReceipt(receipt *Receipt) error {
	query := `
		INSERT INTO receipts (
			id, retailer, purchase_date, purchase_time, total, points, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback()

	if receipt.ID == "" {
		receipt.ID = uuid.New().String()
	}
	receipt.CreatedAt = time.Now()

	_, err = tx.Exec(
		query,
		receipt.ID,
		receipt.Retailer,
		receipt.PurchaseDate,
		receipt.PurchaseTime,
		receipt.Total,
		receipt.Points,
		receipt.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("error saving receipt: %v", err)
	}

	// Insert items
	for _, item := range receipt.Items {
		_, err = tx.Exec(`
			INSERT INTO items (
				receipt_id, short_description, price
			) VALUES ($1, $2, $3)
		`, receipt.ID, item.ShortDescription, item.Price)
		if err != nil {
			return fmt.Errorf("error saving item: %v", err)
		}
	}

	return tx.Commit()
}

func (s *PostgresStore) GetReceipt(id string) (*Receipt, error) {
	receipt := &Receipt{}
	err := s.db.QueryRow(`
		SELECT id, retailer, purchase_date, purchase_time, total, points, created_at
		FROM receipts WHERE id = $1
	`, id).Scan(
		&receipt.ID,
		&receipt.Retailer,
		&receipt.PurchaseDate,
		&receipt.PurchaseTime,
		&receipt.Total,
		&receipt.Points,
		&receipt.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("receipt not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("error getting receipt: %v", err)
	}

	// Get items
	rows, err := s.db.Query(`
		SELECT short_description, price
		FROM items WHERE receipt_id = $1
	`, id)
	if err != nil {
		return nil, fmt.Errorf("error getting items: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.ShortDescription, &item.Price); err != nil {
			return nil, fmt.Errorf("error scanning item: %v", err)
		}
		receipt.Items = append(receipt.Items, item)
	}

	return receipt, nil
}

func (s *PostgresStore) ListReceipts() ([]*Receipt, error) {
	rows, err := s.db.Query(`
		SELECT id, retailer, purchase_date, purchase_time, total, points, created_at
		FROM receipts ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("error listing receipts: %v", err)
	}
	defer rows.Close()

	var receipts []*Receipt
	for rows.Next() {
		receipt := &Receipt{}
		err := rows.Scan(
			&receipt.ID,
			&receipt.Retailer,
			&receipt.PurchaseDate,
			&receipt.PurchaseTime,
			&receipt.Total,
			&receipt.Points,
			&receipt.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning receipt: %v", err)
		}

		// Get items for each receipt
		itemRows, err := s.db.Query(`
			SELECT short_description, price
			FROM items WHERE receipt_id = $1
		`, receipt.ID)
		if err != nil {
			return nil, fmt.Errorf("error getting items: %v", err)
		}
		defer itemRows.Close()

		for itemRows.Next() {
			var item Item
			if err := itemRows.Scan(&item.ShortDescription, &item.Price); err != nil {
				return nil, fmt.Errorf("error scanning item: %v", err)
			}
			receipt.Items = append(receipt.Items, item)
		}

		receipts = append(receipts, receipt)
	}

	return receipts, nil
}
