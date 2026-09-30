package repository

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/csv"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/katarsisxyz/openpray/internal/model"
	"github.com/katarsisxyz/openpray/prayer"
)

//go:generate go run ../../tools/gencities -o data/cities.csv

// citiesCSV lists Indonesia's 514 regencies and cities (Kepmendagri codes),
// derived from github.com/cahyadsn/wilayah (MIT, see data/LICENSE-wilayah).
//
//go:embed data/cities.csv
var citiesCSV []byte

// CityRepository serves the embedded list of Indonesian regencies and cities.
type CityRepository struct {
	cities []model.City
	byID   map[string]int
}

// NewCityRepository parses the embedded city data.
func NewCityRepository() (*CityRepository, error) {
	rows, err := csv.NewReader(bytes.NewReader(citiesCSV)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse cities.csv: %w", err)
	}
	r := &CityRepository{byID: make(map[string]int, len(rows))}
	for i, row := range rows[1:] { // skip header
		var nums [3]float64
		for j, s := range row[4:7] {
			if nums[j], err = strconv.ParseFloat(s, 64); err != nil {
				return nil, fmt.Errorf("cities.csv line %d: %w", i+2, err)
			}
		}
		r.byID[row[0]] = len(r.cities)
		r.cities = append(r.cities, model.City{
			ID:       row[0],
			Name:     row[1],
			Capital:  row[2],
			Province: row[3],
			Coord:    prayer.Coordinates{Lat: nums[0], Lng: nums[1], Elevation: nums[2]},
			Timezone: row[7],
		})
	}
	return r, nil
}

// FindByID looks up a city by its Kemendagri code.
func (r *CityRepository) FindByID(_ context.Context, id string) (model.City, error) {
	i, ok := r.byID[id]
	if !ok {
		return model.City{}, fmt.Errorf("city %q: %w", id, model.ErrNotFound)
	}
	return r.cities[i], nil
}

// Search returns up to limit cities whose name, capital or province contains
// every word of query (case-insensitive). Exact name matches come first, then
// prefix matches; ties keep Kemendagri code order. An empty query lists all cities.
func (r *CityRepository) Search(_ context.Context, query string, limit int) ([]model.City, error) {
	words := strings.Fields(strings.ToLower(query))
	q := strings.Join(words, " ")

	type hit struct {
		city model.City
		rank int
	}
	var hits []hit
	for _, c := range r.cities {
		haystack := strings.ToLower(c.Name + " " + c.Capital + " " + c.Province)
		if !containsAll(haystack, words) {
			continue
		}
		name := shortName(c.Name)
		rank := 2
		switch {
		case q == "" || name == q:
			rank = 0
		case strings.HasPrefix(name, q):
			rank = 1
		}
		hits = append(hits, hit{c, rank})
	}
	slices.SortStableFunc(hits, func(a, b hit) int { return a.rank - b.rank })

	out := make([]model.City, 0, min(limit, len(hits)))
	for _, h := range hits[:min(limit, len(hits))] {
		out = append(out, h.city)
	}
	return out, nil
}

func containsAll(s string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(s, w) {
			return false
		}
	}
	return true
}

// shortName strips the administrative prefix: "Kota Surabaya" -> "surabaya".
func shortName(name string) string {
	name = strings.ToLower(name)
	for _, p := range []string{"kabupaten administrasi ", "kota administrasi ", "kabupaten ", "kota "} {
		if s, ok := strings.CutPrefix(name, p); ok {
			return s
		}
	}
	return name
}
