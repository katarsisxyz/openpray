// Package service contains the prayer times business logic.
package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/katarsisxyz/openpray/internal/model"
	"github.com/katarsisxyz/openpray/prayer"
)

// DefaultMethodID is used when Query.MethodID is empty.
const DefaultMethodID = "kemenag"

// ErrMethodNotFound is returned when the requested method does not exist.
var ErrMethodNotFound = errors.New("unknown method")

// MethodRepository is the source of calculation methods.
type MethodRepository interface {
	List(ctx context.Context) ([]model.Method, error)
	FindByID(ctx context.Context, id string) (model.Method, error)
}

// Query is the location and method to calculate a schedule for.
type Query struct {
	Coord    prayer.Coordinates
	Location *time.Location
	MethodID string
}

// PrayerService calculates prayer schedules.
type PrayerService struct {
	methods MethodRepository
}

// NewPrayerService returns a PrayerService.
func NewPrayerService(methods MethodRepository) *PrayerService {
	return &PrayerService{methods: methods}
}

// Methods returns all available calculation methods.
func (s *PrayerService) Methods(ctx context.Context) ([]model.Method, error) {
	return s.methods.List(ctx)
}

// Daily calculates the schedule for date's calendar day (the time of day is ignored).
func (s *PrayerService) Daily(ctx context.Context, q Query, date time.Time) (model.Schedule, error) {
	m, err := s.method(ctx, q.MethodID)
	if err != nil {
		return model.Schedule{}, err
	}
	y, mo, d := date.Date()
	day, err := calculate(time.Date(y, mo, d, 0, 0, 0, 0, q.Location), q, m)
	if err != nil {
		return model.Schedule{}, err
	}
	return model.Schedule{Method: m, Days: []model.Day{day}}, nil
}

// Monthly calculates the schedule for every day of a month.
func (s *PrayerService) Monthly(ctx context.Context, q Query, year int, month time.Month) (model.Schedule, error) {
	m, err := s.method(ctx, q.MethodID)
	if err != nil {
		return model.Schedule{}, err
	}
	first := time.Date(year, month, 1, 0, 0, 0, 0, q.Location)
	var days []model.Day
	for d := first; d.Month() == month; d = d.AddDate(0, 0, 1) {
		day, err := calculate(d, q, m)
		if err != nil {
			return model.Schedule{}, err
		}
		days = append(days, day)
	}
	return model.Schedule{Method: m, Days: days}, nil
}

func (s *PrayerService) method(ctx context.Context, id string) (model.Method, error) {
	if id == "" {
		id = DefaultMethodID
	}
	m, err := s.methods.FindByID(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return model.Method{}, fmt.Errorf("%w: %q", ErrMethodNotFound, id)
	}
	return m, err
}

func calculate(date time.Time, q Query, m model.Method) (model.Day, error) {
	t, err := prayer.Calculate(date, q.Coord, m.Method)
	if err != nil {
		return model.Day{}, err
	}
	return model.Day{Date: date, Times: t}, nil
}
