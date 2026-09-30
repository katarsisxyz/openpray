// Package repository provides data access.
package repository

import (
	"context"
	"fmt"
	"slices"

	"github.com/katarsisxyz/openpray/internal/model"
	"github.com/katarsisxyz/openpray/prayer"
)

// MethodRepository stores calculation methods in memory.
// It can be swapped for a database-backed implementation without touching the service.
type MethodRepository struct {
	methods []model.Method
}

// NewMethodRepository returns a repository seeded with the prayer package's built-in methods.
func NewMethodRepository() *MethodRepository {
	var methods []model.Method
	for _, id := range prayer.MethodIDs() {
		m, _ := prayer.MethodByID(id)
		methods = append(methods, model.Method{ID: id, Method: m})
	}
	return &MethodRepository{methods: methods}
}

// List returns all methods, sorted by ID.
func (r *MethodRepository) List(_ context.Context) ([]model.Method, error) {
	return slices.Clone(r.methods), nil
}

// FindByID looks up a method by ID.
func (r *MethodRepository) FindByID(_ context.Context, id string) (model.Method, error) {
	for _, m := range r.methods {
		if m.ID == id {
			return m, nil
		}
	}
	return model.Method{}, fmt.Errorf("method %q: %w", id, model.ErrNotFound)
}
