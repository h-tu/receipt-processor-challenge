package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Receipt struct {
	Retailer     string `json:"retailer"`
	PurchaseDate string `json:"purchaseDate"`
	PurchaseTime string `json:"purchaseTime"`
	Items        []Item `json:"items"`
	Total        string `json:"total"`
}

type Item struct {
	ShortDescription string `json:"shortDescription"`
	Price            string `json:"price"`
}

var (
	retailerPattern  = regexp.MustCompile(`^[\w\s\-&]+$`)
	shortDescPattern = regexp.MustCompile(`^[\w\s\-]+$`)
	pricePattern     = regexp.MustCompile(`^\d+\.\d{2}$`)
	dateLayout       = "2006-01-02"
	timeLayout       = "15:04"
)

var store = struct {
	sync.Mutex
	data map[string]int
}{data: make(map[string]int)}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/receipts/process", processReceiptHandler)
	mux.HandleFunc("/receipts/", getPointsHandler)

	log.Println("Starting server on http://localhost:8080...")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func processReceiptHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}

	var receipt Receipt
	if err := json.NewDecoder(r.Body).Decode(&receipt); err != nil {
		http.Error(w, "The receipt is invalid. Please verify input.", http.StatusBadRequest)
		return
	}

	if !isValidReceipt(receipt) {
		http.Error(w, "The receipt is invalid. Please verify input.", http.StatusBadRequest)
		return
	}

	points := calculatePoints(receipt)
	id := generateID()

	store.Lock()
	store.data[id] = points
	store.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func getPointsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 || parts[3] != "points" || parts[2] == "" {
		http.NotFound(w, r)
		return
	}

	store.Lock()
	points, ok := store.data[parts[2]]
	store.Unlock()

	if !ok {
		http.Error(w, "No receipt found for that ID.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"points": points})
}

func isValidReceipt(receipt Receipt) bool {
	if !retailerPattern.MatchString(receipt.Retailer) || !pricePattern.MatchString(receipt.Total) {
		return false
	}
	if _, err := time.Parse(dateLayout, receipt.PurchaseDate); err != nil {
		return false
	}
	if _, err := time.Parse(timeLayout, receipt.PurchaseTime); err != nil {
		return false
	}
	if len(receipt.Items) < 1 {
		return false
	}
	for _, item := range receipt.Items {
		if !shortDescPattern.MatchString(item.ShortDescription) || !pricePattern.MatchString(item.Price) {
			return false
		}
	}
	return true
}

func calculatePoints(receipt Receipt) int {
	points := 0
	for _, ch := range receipt.Retailer {
		if (ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') {
			points++
		}
	}

	totalCents, _ := strconv.ParseInt(strings.ReplaceAll(receipt.Total, ".", ""), 10, 64)
	if totalCents%100 == 0 {
		points += 50
	}
	if totalCents%25 == 0 {
		points += 25
	}

	points += (len(receipt.Items) / 2) * 5

	for _, item := range receipt.Items {
		desc := strings.TrimSpace(item.ShortDescription)
		if len(desc)%3 == 0 {
			if priceVal, err := strconv.ParseFloat(item.Price, 64); err == nil {
				points += int(math.Ceil(priceVal * 0.2))
			}
		}
	}

	if date, err := time.Parse(dateLayout, receipt.PurchaseDate); err == nil && date.Day()%2 == 1 {
		points += 6
	}

	if t, err := time.Parse(timeLayout, receipt.PurchaseTime); err == nil {
		minutes := t.Hour()*60 + t.Minute()
		if minutes > 14*60 && minutes < 16*60 {
			points += 10
		}
	}

	return points
}

func generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
