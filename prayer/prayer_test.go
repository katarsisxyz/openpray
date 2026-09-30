package prayer

import (
	"testing"
	"time"
)

// Reference: Kemenag schedule for KOTA JAKARTA (api.myquran.com, id 1301).
func TestCalculateKemenagJakarta(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	jakarta := Coordinates{Lat: -6.1754, Lng: 106.8272} // Monas

	tests := []struct {
		date string
		want [8]string // imsak, fajr, sunrise, dhuha, dhuhr, asr, maghrib, isha
	}{
		{"2026-09-30", [8]string{"04:14", "04:24", "05:35", "06:02", "11:46", "14:52", "17:50", "18:59"}},
		{"2026-09-01", [8]string{"04:28", "04:38", "05:50", "06:17", "11:56", "15:14", "17:56", "19:05"}},
	}
	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			date, _ := time.ParseInLocation(time.DateOnly, tt.date, loc)
			got, err := Calculate(date, jakarta, Kemenag)
			if err != nil {
				t.Fatal(err)
			}
			names := [8]string{"imsak", "fajr", "sunrise", "dhuha", "dhuhr", "asr", "maghrib", "isha"}
			times := [8]time.Time{got.Imsak, got.Fajr, got.Sunrise, got.Dhuha, got.Dhuhr, got.Asr, got.Maghrib, got.Isha}
			for i, want := range tt.want {
				if s := times[i].Format("15:04"); s != want {
					t.Errorf("%s = %s, want %s", names[i], s, want)
				}
			}
		})
	}
}

// Reference: Muhammadiyah KHGT schedule (khgt.muhammadiyah.or.id/prayer) at Monas.
// KHGT does not publish imsak or dhuha.
func TestCalculateMuhammadiyahJakarta(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	date := time.Date(2026, 9, 30, 0, 0, 0, 0, loc)
	got, err := Calculate(date, Coordinates{Lat: -6.1754, Lng: 106.8272}, Muhammadiyah)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"fajr": "04:31", "sunrise": "05:35", "dhuhr": "11:44", "asr": "14:51", "maghrib": "17:50", "isha": "18:58"}
	times := map[string]time.Time{"fajr": got.Fajr, "sunrise": got.Sunrise, "dhuhr": got.Dhuhr, "asr": got.Asr, "maghrib": got.Maghrib, "isha": got.Isha}
	for name, w := range want {
		if s := times[name].Format("15:04"); s != w {
			t.Errorf("%s = %s, want %s", name, s, w)
		}
	}
}

func TestAdjustments(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	date := time.Date(2026, 9, 30, 0, 0, 0, 0, loc)
	c := Coordinates{Lat: -6.1754, Lng: 106.8272}

	m := MWL
	base, _ := Calculate(date, c, m)
	m.Adjust = Adjustments{Imsak: -5, Fajr: 3, Sunrise: -2, Maghrib: 1}
	got, _ := Calculate(date, c, m)

	for name, tt := range map[string]struct {
		got, base time.Time
		want      time.Duration
	}{
		"imsak":   {got.Imsak, base.Imsak, -2 * time.Minute}, // follows fajr (+3), then -5
		"fajr":    {got.Fajr, base.Fajr, 3 * time.Minute},
		"sunrise": {got.Sunrise, base.Sunrise, -2 * time.Minute},
		"maghrib": {got.Maghrib, base.Maghrib, time.Minute},
		"isha":    {got.Isha, base.Isha, 0},
	} {
		if d := tt.got.Sub(tt.base); d != tt.want {
			t.Errorf("%s shifted by %v, want %v", name, d, tt.want)
		}
	}
}

func TestCalculateHighLatitude(t *testing.T) {
	// Tromsø in summer: the sun never goes 18° below the horizon.
	date := time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC)
	if _, err := Calculate(date, Coordinates{Lat: 69.65, Lng: 18.96}, Kemenag); err != ErrUndefined {
		t.Fatalf("err = %v, want ErrUndefined", err)
	}
}
