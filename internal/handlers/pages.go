package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"api-banque-datadex/internal/favorites"
	"api-banque-datadex/internal/finance"
	"api-banque-datadex/internal/plaid"
)

type AppHandler struct {
	PlaidClient   *plaid.Client
	FavStore      *favorites.Store
	TemplatesDir  string
	InstitutionID string
	StartDate     string
	EndDate       string

	mu            sync.RWMutex
	accessToken   string
	cachedTx      []finance.Transaction
	hasLoadedOnce bool
}

type PageData struct {
	Title        string
	ActivePage   string
	ErrorMsg     string
	StatusCode   int
	Transactions []finance.Transaction
	Transaction  *finance.Transaction
	Favorites    []finance.Transaction
	Result       finance.PageResult
	Filters      finance.Filters
	TotalCount   int
	FavCount     int
}

func (h *AppHandler) renderTemplate(w http.ResponseWriter, status int, pageName string, data PageData) {
	layoutPath := filepath.Join(h.TemplatesDir, "base.html")
	pagePath := filepath.Join(h.TemplatesDir, pageName)

	funcMap := template.FuncMap{
		"plus":  func(a, b int) int { return a + b },
		"minus": func(a, b int) int { return a - b },
	}

	tmpl, err := template.New("base").Funcs(funcMap).ParseFiles(layoutPath, pagePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Erreur de template : %v", err)
		return
	}

	w.WriteHeader(status)
	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		fmt.Fprintf(w, "Erreur d'exécution du template : %v", err)
	}
}

func (h *AppHandler) renderError(w http.ResponseWriter, status int, message string) {
	h.renderTemplate(w, status, "error.html", PageData{
		Title:      "Erreur - DataDex",
		ActivePage: "error",
		StatusCode: status,
		ErrorMsg:   message,
	})
}

func toFinanceTransaction(pt plaid.Transaction) finance.Transaction {
	category := "Non classée"
	if len(pt.Category) > 0 && strings.TrimSpace(pt.Category[0]) != "" {
		category = pt.Category[0]
	}

	name := strings.TrimSpace(pt.MerchantName)
	if name == "" {
		name = strings.TrimSpace(pt.Name)
	}
	if name == "" {
		name = "Transaction sans nom"
	}

	status := "Validée"
	if pt.Pending {
		status = "En attente"
	}

	return finance.Transaction{
		ID:       pt.TransactionID,
		Name:     name,
		Category: category,
		Status:   status,
		Amount:   pt.Amount,
		Date:     pt.Date,
	}
}

func (h *AppHandler) getTransactions() ([]finance.Transaction, error) {
	h.mu.RLock()
	if h.hasLoadedOnce && len(h.cachedTx) > 0 {
		items := make([]finance.Transaction, len(h.cachedTx))
		copy(items, h.cachedTx)
		h.mu.RUnlock()
		return items, nil
	}
	token := h.accessToken
	h.mu.RUnlock()

	if h.PlaidClient == nil {
		return nil, fmt.Errorf("client Plaid non initialisé")
	}

	if token == "" {
		instID := h.InstitutionID
		if instID == "" {
			instID = "ins_109508"
		}

		pubToken, err := h.PlaidClient.CreatePublicToken(instID, []string{"transactions"})
		if err != nil {
			return nil, fmt.Errorf("création public_token impossible : %w", err)
		}

		accToken, err := h.PlaidClient.ExchangePublicToken(pubToken)
		if err != nil {
			return nil, fmt.Errorf("échange access_token impossible : %w", err)
		}
		token = accToken
	}

	rawTx, err := h.PlaidClient.GetTransactions(token, h.StartDate, h.EndDate)
	if err != nil {
		return nil, fmt.Errorf("récupération des transactions impossible : %w", err)
	}

	converted := make([]finance.Transaction, 0, len(rawTx))
	for _, pt := range rawTx {
		converted = append(converted, toFinanceTransaction(pt))
	}

	h.mu.Lock()
	h.accessToken = token
	h.cachedTx = converted
	h.hasLoadedOnce = true
	h.mu.Unlock()

	return converted, nil
}

// GET /
func (h *AppHandler) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		h.renderError(w, http.StatusNotFound, "La page demandée n'existe pas.")
		return
	}

	txList, err := h.getTransactions()
	if err != nil {
		h.renderError(w, http.StatusBadGateway, "Impossible de charger les données bancaires pour le moment : "+err.Error())
		return
	}

	favs, _ := h.FavStore.GetAll()

	h.renderTemplate(w, http.StatusOK, "home.html", PageData{
		Title:        "Accueil - DataDex Banque",
		ActivePage:   "home",
		Transactions: txList,
		TotalCount:   len(txList),
		FavCount:     len(favs),
	})
}

func (h *AppHandler) Transactions(w http.ResponseWriter, r *http.Request) {
	txList, err := h.getTransactions()
	if err != nil {
		h.renderError(w, http.StatusBadGateway, "Erreur lors de la récupération des transactions : "+err.Error())
		return
	}

	q := r.URL.Query()
	page, err := strconv.Atoi(q.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(q.Get("limit"))
	if err != nil || (pageSize != 10 && pageSize != 20 && pageSize != 30) {
		pageSize = 10
	}

	var minAmount float64
	if rawMin := strings.TrimSpace(q.Get("min_amount")); rawMin != "" {
		parsedMin, err := strconv.ParseFloat(rawMin, 64)
		if err != nil || parsedMin < 0 {
			h.renderError(w, http.StatusBadRequest, "Le montant minimum saisi est invalide.")
			return
		}
		minAmount = parsedMin
	}

	filters := finance.Filters{
		Query:     q.Get("q"),
		Category:  q.Get("category"),
		Status:    q.Get("status"),
		MinAmount: minAmount,
		Page:      page,
		PageSize:  pageSize,
	}

	res := finance.ProcessTransactions(txList, filters)

	h.renderTemplate(w, http.StatusOK, "transactions.html", PageData{
		Title:      "Transactions - DataDex Banque",
		ActivePage: "transactions",
		Filters:    filters,
		Result:     res,
		TotalCount: len(txList),
	})
}

func (h *AppHandler) TransactionDetail(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/transactions/")
	id = strings.TrimSpace(id)
	if id == "" {
		h.renderError(w, http.StatusBadRequest, "Identifiant de transaction manquant.")
		return
	}

	txList, err := h.getTransactions()
	if err != nil {
		h.renderError(w, http.StatusBadGateway, "Impossible de charger les détails de la transaction : "+err.Error())
		return
	}

	for _, t := range txList {
		if t.ID == id {
			selected := t
			h.renderTemplate(w, http.StatusOK, "detail.html", PageData{
				Title:       selected.Name + " - Détails",
				ActivePage:  "transactions",
				Transaction: &selected,
			})
			return
		}
	}

	h.renderError(w, http.StatusNotFound, "Aucune transaction trouvée avec cet identifiant.")
}

func (h *AppHandler) Favorites(w http.ResponseWriter, r *http.Request) {
	favs, err := h.FavStore.GetAll()
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "Impossible de lire le fichier des favoris.")
		return
	}

	h.renderTemplate(w, http.StatusOK, "favorites.html", PageData{
		Title:      "Favoris - DataDex Banque",
		ActivePage: "favorites",
		Favorites:  favs,
		FavCount:   len(favs),
	})
}

func (h *AppHandler) About(w http.ResponseWriter, r *http.Request) {
	h.renderTemplate(w, http.StatusOK, "about.html", PageData{
		Title:      "À propos - DataDex Banque",
		ActivePage: "about",
	})
}