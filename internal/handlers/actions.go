package handlers

import (
	"net/http"
	"strings"
)

func (h *AppHandler) AddFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.renderError(w, http.StatusMethodNotAllowed, "Méthode non autorisée.")
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		h.renderError(w, http.StatusBadRequest, "ID de transaction manquant pour l'ajout aux favoris.")
		return
	}

	txList, err := h.getTransactions()
	if err != nil {
		h.renderError(w, http.StatusBadGateway, "Impossible de récupérer la transaction depuis l'API : "+err.Error())
		return
	}

	for _, t := range txList {
		if t.ID == id {
			if err := h.FavStore.Add(t); err != nil {
				h.renderError(w, http.StatusInternalServerError, "Erreur lors de l'enregistrement du favori.")
				return
			}
			http.Redirect(w, r, "/favorites", http.StatusSeeOther)
			return
		}
	}

	h.renderError(w, http.StatusNotFound, "Transaction introuvable.")
}

func (h *AppHandler) RemoveFavorite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.renderError(w, http.StatusMethodNotAllowed, "Méthode non autorisée.")
		return
	}

	id := strings.TrimSpace(r.FormValue("id"))
	if id == "" {
		h.renderError(w, http.StatusBadRequest, "ID de transaction manquant pour la suppression.")
		return
	}

	if err := h.FavStore.Remove(id); err != nil {
		h.renderError(w, http.StatusInternalServerError, "Erreur lors de la suppression du favori.")
		return
	}

	http.Redirect(w, r, "/favorites", http.StatusSeeOther)
}