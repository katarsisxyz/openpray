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

var (
	// ErrMethodNotFound is returned when the requested method does not exist.
	ErrMethodNotFound = errors.New("unknown method")
	// ErrCityNotFound is returned when the requested city does not exist.
	ErrCityNotFound = errors.New("unknown city")
	// ErrInvalidTimezone is returned when the time zone is not a known IANA name.
	ErrInvalidTimezone = errors.New("unknown tz")
)

// MethodRepository is the source of calculation methods.
type MethodRepository interface {
	List(ctx context.Context) ([]model.Method, error)
	FindByID(ctx context.Context, id string) (model.Method, error)
}

// CityRepository is the source of cities.
type CityRepository interface {
	FindByID(ctx context.Context, id string) (model.City, error)
	Search(ctx context.Context, query string, limit int) ([]model.City, error)
}

// Query is the place and method to calculate a schedule for. The place is
// either CityID, or Coord together with Timezone.
type Query struct {
	CityID   string
	Coord    prayer.Coordinates
	Timezone string // IANA name, e.g. "Asia/Jakarta"
	MethodID string
}

// PrayerService calculates prayer schedules.
type PrayerService struct {
	methods MethodRepository
	cities  CityRepository
	now     func() time.Time
}

// NewPrayerService returns a PrayerService.
func NewPrayerService(methods MethodRepository, cities CityRepository) *PrayerService {
	return &PrayerService{methods: methods, cities: cities, now: time.Now}
}

// Methods returns all available calculation methods.
func (s *PrayerService) Methods(ctx context.Context) ([]model.Method, error) {
	return s.methods.List(ctx)
}

// Daily calculates the schedule for date's calendar day (the time of day and
// location are ignored). A zero date means today at the resolved place.
func (s *PrayerService) Daily(ctx context.Context, q Query, date time.Time) (model.Schedule, error) {
	p, m, err := s.resolve(ctx, q)
	if err != nil {
		return model.Schedule{}, err
	}
	if date.IsZero() {
		date = s.now().In(p.Location)
	}
	y, mo, d := date.Date()
	day, err := calculate(time.Date(y, mo, d, 0, 0, 0, 0, p.Location), p, m)
	if err != nil {
		return model.Schedule{}, err
	}
	return model.Schedule{Place: p, Method: m, Days: []model.Day{day}}, nil
}

// Monthly calculates the schedule for every day of a month.
func (s *PrayerService) Monthly(ctx context.Context, q Query, year int, month time.Month) (model.Schedule, error) {
	p, m, err := s.resolve(ctx, q)
	if err != nil {
		return model.Schedule{}, err
	}
	first := time.Date(year, month, 1, 0, 0, 0, 0, p.Location)
	var days []model.Day
	for d := first; d.Month() == month; d = d.AddDate(0, 0, 1) {
		day, err := calculate(d, p, m)
		if err != nil {
			return model.Schedule{}, err
		}
		days = append(days, day)
	}
	return model.Schedule{Place: p, Method: m, Days: days}, nil
}

// resolve turns a query into a concrete place and method.
func (s *PrayerService) resolve(ctx context.Context, q Query) (model.Place, model.Method, error) {
	var p model.Place
	coord, tz := q.Coord, q.Timezone
	if q.CityID != "" {
		c, err := s.cities.FindByID(ctx, q.CityID)
		if errors.Is(err, model.ErrNotFound) {
			return p, model.Method{}, fmt.Errorf("%w: %q", ErrCityNotFound, q.CityID)
		}
		if err != nil {
			return p, model.Method{}, err
		}
		p.City = &c
		coord, tz = c.Coord, c.Timezone
	}
	loc, err := time.LoadLocation(tz)
	if err != nil || tz == "" {
		return p, model.Method{}, fmt.Errorf("%w: %q", ErrInvalidTimezone, tz)
	}
	p.Coord, p.Location = coord, loc

	id := q.MethodID
	if id == "" {
		id = DefaultMethodID
	}
	m, err := s.methods.FindByID(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return p, model.Method{}, fmt.Errorf("%w: %q", ErrMethodNotFound, id)
	}
	return p, m, err
}

func calculate(date time.Time, p model.Place, m model.Method) (model.Day, error) {
	t, err := prayer.Calculate(date, p.Coord, m.Method)
	if err != nil {
		return model.Day{}, err
	}
	return model.Day{Date: date, Times: t}, nil
}
