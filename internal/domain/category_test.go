package domain

import "testing"

func TestParseCategory(t *testing.T) {
	tests := []struct {
		in   string
		want Category
	}{
		{"еда", CategoryFood},
		{"Еда", CategoryFood},
		{"food", CategoryFood},
		{"жильё", CategoryHousing},
		{"жилье", CategoryHousing},
		{"транспорт", CategoryTransport},
		{"другое", CategoryOther},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseCategory(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %s want %s", got, tt.want)
			}
		})
	}
}

func TestParseCategoryInvalid(t *testing.T) {
	_, err := ParseCategory("крипта")
	if err == nil {
		t.Fatal("expected error")
	}
}
