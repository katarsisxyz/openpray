package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/katarsisxyz/openpray/internal/model"
	"github.com/katarsisxyz/openpray/prayer"
)

type fakeMethodRepo struct{ methods []model.Method }

func (f fakeMethodRepo) List(context.Context) ([]model.Method, error) { return f.methods, nil }

func (f fakeMethodRepo) FindByID(_ context.Context, id string) (model.Method, error) {
	for _, m := range f.methods {
		if m.ID == id {
			return m, nil
		}
	}
	return model.Method{}, model.ErrNotFound
}

type fakeCityRepo struct{ cities []model.City }

func (f fakeCityRepo) FindByID(_ context.Context, id string) (model.City, error) {
	for _, c := range f.cities {
		if c.ID == id {
			return c, nil
		}
	}
	return model.City{}, model.ErrNotFound
}

func (f fakeCityRepo) Search(context.Context, string, int) ([]model.City, error) {
	return f.cities, nil
}

var monas = prayer.Coordinates{Lat: -6.1754, Lng: 106.8272}

func newTestService() *PrayerService {
	return NewPrayerService(
		fakeMethodRepo{[]model.Method{{ID: "kemenag", Method: prayer.Kemenag}}},
		fakeCityRepo{[]model.City{{ID: "31.71", Name: "Kota Administrasi Jakarta Pusat", Coord: monas, Timezone: "Asia/Jakarta"}}},
	)
}

func TestDailyUsesDefaultMethod(t *testing.T) {
	svc := newTestService()
	q := Query{Coord: monas, Timezone: "Asia/Jakarta"}
	s, err := svc.Daily(context.Background(), q, time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if s.Method.ID != DefaultMethodID || len(s.Days) != 1 || s.Place.City != nil {
		t.Fatalf("schedule = %+v", s)
	}
	if got := s.Days[0].Times.Dhuhr.Format("15:04"); got != "11:46" {
		t.Errorf("dhuhr = %s, want 11:46", got)
	}
}

func TestDailyByCity(t *testing.T) {
	svc := newTestService()
	s, err := svc.Daily(context.Background(), Query{CityID: "31.71"}, time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if s.Place.City == nil || s.Place.City.ID != "31.71" || s.Place.Location.String() != "Asia/Jakarta" {
		t.Fatalf("place = %+v", s.Place)
	}
	if got := s.Days[0].Times.Dhuhr.Format("15:04"); got != "11:46" {
		t.Errorf("dhuhr = %s, want 11:46", got)
	}
}

func TestDailyDefaultsToTodayAtPlace(t *testing.T) {
	svc := newTestService()
	// 2026-09-30 20:00 UTC is already 2026-10-01 in Jakarta (UTC+7).
	svc.now = func() time.Time { return time.Date(2026, 9, 30, 20, 0, 0, 0, time.UTC) }
	s, err := svc.Daily(context.Background(), Query{CityID: "31.71"}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Days[0].Date.Format(time.DateOnly); got != "2026-10-01" {
		t.Errorf("date = %s, want 2026-10-01", got)
	}
}

func TestMonthlyDayCount(t *testing.T) {
	svc := newTestService()
	q := Query{Coord: monas, Timezone: "Asia/Jakarta"}
	for _, tt := range []struct {
		year  int
		month time.Month
		want  int
	}{{2026, time.February, 28}, {2028, time.February, 29}, {2026, time.September, 30}, {2026, time.December, 31}} {
		s, err := svc.Monthly(context.Background(), q, tt.year, tt.month)
		if err != nil {
			t.Fatal(err)
		}
		if len(s.Days) != tt.want {
			t.Errorf("%d-%02d: %d days, want %d", tt.year, tt.month, len(s.Days), tt.want)
		}
	}
}

func TestResolveErrors(t *testing.T) {
	svc := newTestService()
	for _, tt := range []struct {
		name string
		q    Query
		want error
	}{
		{"unknown method", Query{CityID: "31.71", MethodID: "foo"}, ErrMethodNotFound},
		{"unknown city", Query{CityID: "99.99"}, ErrCityNotFound},
		{"unknown tz", Query{Coord: monas, Timezone: "Mars/Base"}, ErrInvalidTimezone},
		{"empty tz", Query{Coord: monas}, ErrInvalidTimezone},
	} {
		if _, err := svc.Daily(context.Background(), tt.q, time.Time{}); !errors.Is(err, tt.want) {
			t.Errorf("%s: err = %v, want %v", tt.name, err, tt.want)
		}
	}
}
