package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"

	"example.com/flagged-commerce-cutover/internal/checkout"
)

func main() {
	workflow := checkout.Workflow{
		Flags:   checkout.FlagClient{APIKey: os.Getenv("INFRAI_API_KEY")},
		FlagKey: "commerce-checkout-v2",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /orders", func(w http.ResponseWriter, r *http.Request) {
		var order checkout.Order
		if err := json.NewDecoder(r.Body).Decode(&order); err != nil || order.ID == "" || order.CustomerID == "" || order.AmountCents <= 0 {
			http.Error(w, "invalid order", http.StatusBadRequest)
			return
		}
		result, err := workflow.Process(r.Context(), order)
		if err != nil {
			var rejected *checkout.InfraiError
			if errors.As(err, &rejected) && rejected.Status >= 400 && rejected.Status < 500 {
				http.Error(w, "flag decision rejected", http.StatusBadRequest)
				return
			}
			http.Error(w, "flag decision unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("checkout service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
