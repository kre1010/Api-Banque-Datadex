package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"api-banque-datadex/internal/favorites"
	"api-banque-datadex/internal/handlers"
	"api-banque-datadex/internal/plaid"
)

func getEnvOrDefault(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func main() {
	port := getEnvOrDefault("PORT", "8080")
	institutionID := getEnvOrDefault("PLAID_INSTITUTION_ID", "ins_109508")
	startDate := getEnvOrDefault("PLAID_START_DATE", "2026-01-01")
	endDate := getEnvOrDefault("PLAID_END_DATE", "2026-11-24")

	favStore, err := favorites.NewStore(filepath.Join("data", "favorites.json"))
	if err != nil {
		log.Fatalf("Erreur initialisation favoris : %v", err)
	}

	plaidClient, err := plaid.NewClient()
	if err != nil {
		log.Printf("Attention client Plaid : %v", err)
	}

	app := &handlers.AppHandler{
		PlaidClient:   plaidClient,
		FavStore:      favStore,
		TemplatesDir:  "templates",
		InstitutionID: institutionID,
		StartDate:     startDate,
		EndDate:       endDate,
	}

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	mux.HandleFunc("/", app.Home)
	mux.HandleFunc("/transactions", app.Transactions)
	mux.HandleFunc("/transactions/", app.TransactionDetail)
	mux.HandleFunc("/favorites", app.Favorites)
	mux.HandleFunc("/about", app.About)

	mux.HandleFunc("/favorites/add", app.AddFavorite)
	mux.HandleFunc("/favorites/remove", app.RemoveFavorite)

	addr := ":" + port
	log.Printf("Serveur DataDex démarré sur http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Erreur serveur : %v", err)
	}
}