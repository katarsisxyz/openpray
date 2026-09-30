package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/katarsisxyz/openpray/internal/model"
)

func TestCityRepository(t *testing.T) {
	r, err := NewCityRepository()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	all, _ := r.Search(ctx, "", 1000)
	if len(all) != 514 {
		t.Errorf("got %d cities, want 514", len(all))
	}

	c, err := r.FindByID(ctx, "35.78")
	if err != nil {
		t.Fatal(err)
	}
	if c.Name != "Kota Surabaya" || c.Province != "Jawa Timur" || c.Timezone != "Asia/Jakarta" {
		t.Errorf("35.78 = %+v", c)
	}
	if _, err := r.FindByID(ctx, "99.99"); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}

	// Time zones follow province borders, e.g. West Kalimantan is WIB, East is WITA.
	for id, tz := range map[string]string{"61.71": "Asia/Jakarta", "64.72": "Asia/Makassar", "91.71": "Asia/Jayapura"} {
		if c, _ := r.FindByID(ctx, id); c.Timezone != tz {
			t.Errorf("%s (%s) timezone = %s, want %s", id, c.Name, c.Timezone, tz)
		}
	}
}

func TestCitySearch(t *testing.T) {
	r, err := NewCityRepository()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	tests := []struct {
		query     string
		wantFirst string
		wantCount int // 0 = don't check
	}{
		{"surabaya", "Kota Surabaya", 1},
		{"SURABAYA", "Kota Surabaya", 1},
		{"bogor", "Kabupaten Bogor", 2},
		{"kota bogor", "Kota Bogor", 1},
		{"makas", "Kota Makassar", 0},          // prefix
		{"jakarta", "", 6},                     // 5 cities + Kepulauan Seribu, matched by province
		{"tidak ada kota seperti ini", "", -1}, // no results
	}
	for _, tt := range tests {
		got, _ := r.Search(ctx, tt.query, 20)
		if tt.wantCount == -1 {
			if len(got) != 0 {
				t.Errorf("%q: got %d results, want none", tt.query, len(got))
			}
			continue
		}
		if tt.wantCount > 0 && len(got) != tt.wantCount {
			t.Errorf("%q: got %d results, want %d", tt.query, len(got), tt.wantCount)
		}
		if tt.wantFirst != "" && (len(got) == 0 || got[0].Name != tt.wantFirst) {
			t.Errorf("%q: first = %v, want %s", tt.query, got, tt.wantFirst)
		}
	}

	if got, _ := r.Search(ctx, "", 5); len(got) != 5 {
		t.Errorf("limit 5: got %d", len(got))
	}
}
