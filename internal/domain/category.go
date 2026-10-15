package domain

import (
	"fmt"
	"strings"
)

type Category string

const (
	CategoryFood          Category = "food"
	CategoryTransport     Category = "transport"
	CategoryHousing       Category = "housing"
	CategoryEntertainment Category = "entertainment"
	CategoryHealth        Category = "health"
	CategoryShopping      Category = "shopping"
	CategoryOther         Category = "other"
)

var AllCategories = []Category{
	CategoryFood,
	CategoryTransport,
	CategoryHousing,
	CategoryEntertainment,
	CategoryHealth,
	CategoryShopping,
	CategoryOther,
}

var categoryLabels = map[Category]string{
	CategoryFood:          "еда",
	CategoryTransport:     "транспорт",
	CategoryHousing:       "жильё",
	CategoryEntertainment: "развлечения",
	CategoryHealth:        "здоровье",
	CategoryShopping:      "покупки",
	CategoryOther:         "другое",
}

var labelToCategory = map[string]Category{
	"еда":          CategoryFood,
	"транспорт":    CategoryTransport,
	"жильё":        CategoryHousing,
	"жилье":        CategoryHousing,
	"развлечения":  CategoryEntertainment,
	"здоровье":     CategoryHealth,
	"покупки":      CategoryShopping,
	"другое":       CategoryOther,
	"food":         CategoryFood,
	"transport":    CategoryTransport,
	"housing":      CategoryHousing,
	"entertainment": CategoryEntertainment,
	"health":       CategoryHealth,
	"shopping":     CategoryShopping,
	"other":        CategoryOther,
}

func (c Category) Label() string {
	if label, ok := categoryLabels[c]; ok {
		return label
	}
	return string(c)
}

func (c Category) Valid() bool {
	_, ok := categoryLabels[c]
	return ok
}

func ParseCategory(s string) (Category, error) {
	key := strings.ToLower(strings.TrimSpace(s))
	if c, ok := labelToCategory[key]; ok {
		return c, nil
	}
	c := Category(key)
	if c.Valid() {
		return c, nil
	}
	return "", fmt.Errorf("%w: unknown category %q", ErrValidation, s)
}

func CategoryLabelsList() string {
	parts := make([]string, 0, len(AllCategories))
	for _, c := range AllCategories {
		parts = append(parts, fmt.Sprintf("%s (%s)", c.Label(), c))
	}
	return strings.Join(parts, ", ")
}
