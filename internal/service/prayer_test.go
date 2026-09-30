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

func newTestService(t *testing.T) (*PrayerService, Query) {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewPrayerService(fakeMethodRepo{[]model.Method{{ID: "kemenag", Method: prayer.Kemenag}}})
	return svc, Query{Coord: prayer.Coordinates{Lat: -6.1754, Lng: 106.8272}, Location: loc}
}

func TestDailyUsesDefaultMethod(t *testing.T) {
	svc, q := newTestService(t)
	s, err := svc.Daily(context.Background(), q, time.Date(2026, 9, 30, 15, 0, 0, 0, q.Location))
	if err != nil {
		t.Fatal(err)
	}
	if s.Method.ID != DefaultMethodID || len(s.Days) != 1 {
		t.Fatalf("schedule = %+v", s)
	}
	if got := s.Days[0].Times.Dhuhr.Format("15:04"); got != "11:46" {
		t.Errorf("dhuhr = %s, want 11:46", got)
	}
}

func TestMonthlyDayCount(t *testing.T) {
	svc, q := newTestService(t)
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

func TestUnknownMethod(t *testing.T) {
	svc, q := newTestService(t)
	q.MethodID = "foo"
	if _, err := svc.Daily(context.Background(), q, time.Now()); !errors.Is(err, ErrMethodNotFound) {
		t.Fatalf("err = %v, want ErrMethodNotFound", err)
	}
}
