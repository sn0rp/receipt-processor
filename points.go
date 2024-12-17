package main

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func calculatePoints(receipt *Receipt) (int64, error) {
	var points int64

	// Rule 1: One point for every alphanumeric character in the retailer name
	alphanumeric := regexp.MustCompile(`[a-zA-Z0-9]`)
	points += int64(len(alphanumeric.FindAllString(receipt.Retailer, -1)))

	// Rule 2: 50 points if the total is a round dollar amount
	total, err := strconv.ParseFloat(receipt.Total, 64)
	if err != nil {
		return 0, err
	}
	if total == float64(int64(total)) {
		points += 50
	}

	// Rule 3: 25 points if the total is a multiple of 0.25
	if math.Mod(total*100, 25) == 0 {
		points += 25
	}

	// Rule 4: 5 points for every two items
	points += int64(len(receipt.Items) / 2 * 5)

	// Rule 5: Points for item descriptions
	for _, item := range receipt.Items {
		trimmed := strings.TrimSpace(item.ShortDescription)
		if len(trimmed)%3 == 0 {
			price, err := strconv.ParseFloat(item.Price, 64)
			if err != nil {
				return 0, err
			}
			points += int64(math.Ceil(price * 0.2))
		}
	}

	// Rule 6: 6 points if the day in the purchase date is odd
	purchaseDate, err := time.Parse("2006-01-02", receipt.PurchaseDate)
	if err != nil {
		return 0, err
	}
	if purchaseDate.Day()%2 == 1 {
		points += 6
	}

	// Rule 7: 10 points if the time of purchase is after 2:00pm and before 4:00pm
	purchaseTime, err := time.Parse("15:04", receipt.PurchaseTime)
	if err != nil {
		return 0, err
	}
	hour := purchaseTime.Hour()
	minute := purchaseTime.Minute()

	if (hour == 14 && minute > 0) || hour == 15 {
		points += 10
	}

	return points, nil
}
