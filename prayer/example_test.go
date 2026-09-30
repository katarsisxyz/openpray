package prayer_test

import (
	"fmt"
	"time"

	"github.com/katarsisxyz/openpray/prayer"
)

func ExampleCalculate() {
	loc, _ := time.LoadLocation("Asia/Jakarta")
	date := time.Date(2026, 9, 30, 0, 0, 0, 0, loc)
	monas := prayer.Coordinates{Lat: -6.1754, Lng: 106.8272}

	t, err := prayer.Calculate(date, monas, prayer.Kemenag)
	if err != nil {
		panic(err)
	}
	fmt.Println("Fajr   ", t.Fajr.Format("15:04"))
	fmt.Println("Maghrib", t.Maghrib.Format("15:04"))
	// Output:
	// Fajr    04:24
	// Maghrib 17:50
}

// Parameters are plain struct fields, e.g. Kemenag with Hanafi Asr.
func ExampleMethod_custom() {
	m := prayer.Kemenag
	m.Name = "Kemenag (Hanafi Asr)"
	m.AsrFactor = 2

	loc, _ := time.LoadLocation("Asia/Makassar")
	date := time.Date(2026, 9, 30, 0, 0, 0, 0, loc)
	t, _ := prayer.Calculate(date, prayer.Coordinates{Lat: -5.1477, Lng: 119.4327}, m)
	fmt.Println(m.Name, t.Asr.Format("15:04"))
}

func ExampleMethodByID() {
	for _, id := range prayer.MethodIDs() {
		m, _ := prayer.MethodByID(id)
		fmt.Printf("%s: Fajr %g°, Isha %g°\n", id, m.FajrAngle, m.IshaAngle)
	}
	// Output:
	// kemenag: Fajr 20°, Isha 18°
	// muhammadiyah: Fajr 18°, Isha 18°
	// mwl: Fajr 18°, Isha 17°
}
