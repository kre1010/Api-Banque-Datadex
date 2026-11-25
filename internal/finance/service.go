package finance

import (
	"math"
	"sort"
	"strings"
)

func Search(list []Transaction, query string) []Transaction {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return list
	}

	var result []Transaction
	for _, t := range list {
		inName := strings.Contains(strings.ToLower(t.Name), q)
		inCategory := strings.Contains(strings.ToLower(t.Category), q)
		if inName || inCategory {
			result = append(result, t)
		}
	}
	return result
}

func Filter(list []Transaction, category, status string, minAmount float64) []Transaction {
	var result []Transaction

	for _, t := range list {
		if category != "" && !strings.EqualFold(t.Category, category) {
			continue
		}
		if status != "" && !strings.EqualFold(t.Status, status) {
			continue
		}
		if minAmount > 0 && t.Amount < minAmount {
			continue
		}

		result = append(result, t)
	}

	return result
}

func Paginate(list []Transaction, page, pageSize int) ([]Transaction, int, int, int) {
	if pageSize <= 0 {
		pageSize = 10
	}

	totalItems := len(list)
	if totalItems == 0 {
		return []Transaction{}, 0, 1, pageSize
	}

	totalPages := int(math.Ceil(float64(totalItems) / float64(pageSize)))
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if end > totalItems {
		end = totalItems
	}

	return list[start:end], totalItems, totalPages, page
}

func ProcessTransactions(all []Transaction, f Filters) PageResult {
	categories := ExtractCategories(all)

	step1 := Search(all, f.Query)
	
	step2 := Filter(step1, f.Category, f.Status, f.MinAmount)
	
	pagedItems, totalItems, totalPages, currentPage := Paginate(step2, f.Page, f.PageSize)

	return PageResult{
		Items:       pagedItems,
		TotalItems:  totalItems,
		TotalPages:  totalPages,
		CurrentPage: currentPage,
		PageSize:    f.PageSize,
		Categories:  categories,
	}
}

func ExtractCategories(list []Transaction) []string {
	seen := make(map[string]bool)
	var cats []string

	for _, t := range list {
		c := strings.TrimSpace(t.Category)
		if c != "" && !seen[c] {
			seen[c] = true
			cats = append(cats, c)
		}
	}
	sort.Strings(cats)
	return cats
}