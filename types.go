package main

import "time"

type Receipt struct {
	ID           string    `json:"id"`
	Retailer     string    `json:"retailer" validate:"required,regexp=^[\\w\\s\\-&]+$"`
	PurchaseDate string    `json:"purchaseDate" validate:"required,datetime=2006-01-02"`
	PurchaseTime string    `json:"purchaseTime" validate:"required,datetime=15:04"`
	Items        []Item    `json:"items" validate:"required,min=1"`
	Total        string    `json:"total" validate:"required,regexp=^\\d+\\.\\d{2}$"`
	CreatedAt    time.Time `json:"-"`
	Points       int64     `json:"points,omitempty"`
}

type Item struct {
	ShortDescription string `json:"shortDescription" validate:"required,regexp=^[\\w\\s\\-]+$"`
	Price            string `json:"price" validate:"required,regexp=^\\d+\\.\\d{2}$"`
}
