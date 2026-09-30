package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/katarsisxyz/openpray/internal/model"
)

// CityService looks up Indonesian regencies and cities.
type CityService struct {
	cities CityRepository
}

// NewCityService returns a CityService.
func NewCityService(cities CityRepository) *CityService {
	return &CityService{cities: cities}
}

// Search returns up to limit cities matching query; an empty query lists all.
func (s *CityService) Search(ctx context.Context, query string, limit int) ([]model.City, error) {
	return s.cities.Search(ctx, query, limit)
}

// Get returns one city by its Kemendagri code.
func (s *CityService) Get(ctx context.Context, id string) (model.City, error) {
	c, err := s.cities.FindByID(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return model.City{}, fmt.Errorf("%w: %q", ErrCityNotFound, id)
	}
	return c, err
}
