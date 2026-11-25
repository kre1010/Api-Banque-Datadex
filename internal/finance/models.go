package finance

type Transaction struct {
	ID       string
	Name     string
	Category string
	Status   string
	Amount   float64
	Date     string
}

type Filters struct {
	Query       string  // FT1 : recherche texte (ex. nom ou catégorie)
	Category    string  // FT2 - Filtre 1 : catégorie exacte ou choisie
	Status      string  // FT2 - Filtre 2 : "En attente" ou "Validée"
	MinAmount   float64 // FT2 - Filtre 3 : montant minimum
	Page        int     // FT3 : numéro de page (1, 2, 3...)
	PageSize    int     // FT3 : nombre d'éléments par page (ex. 10, 20, 30)
}

type PageResult struct {
	Items       []Transaction
	TotalItems  int
	TotalPages  int
	CurrentPage int
	PageSize    int
	Categories  []string
}