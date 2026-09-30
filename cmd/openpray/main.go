// Command openpray prints the prayer times for one location.
//
//	go run ./cmd/openpray -lat -6.1754 -lng 106.8272 -tz Asia/Jakarta
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"time"
	_ "time/tzdata" // load time zones even on systems/images without tzdata (e.g. distroless)

	"github.com/katarsisxyz/openpray/prayer"
)

func main() {
	lat := flag.Float64("lat", -6.1754, "latitude")
	lng := flag.Float64("lng", 106.8272, "longitude")
	elev := flag.Float64("elev", 0, "elevation (meters)")
	tz := flag.String("tz", "Asia/Jakarta", "IANA time zone")
	date := flag.String("date", "", "date as YYYY-MM-DD (default today)")
	method := flag.String("method", "kemenag", strings.Join(prayer.MethodIDs(), " | "))
	flag.Parse()

	loc, err := time.LoadLocation(*tz)
	if err != nil {
		log.Fatal(err)
	}
	day := time.Now().In(loc)
	if *date != "" {
		if day, err = time.ParseInLocation(time.DateOnly, *date, loc); err != nil {
			log.Fatal(err)
		}
	}
	m, ok := prayer.MethodByID(*method)
	if !ok {
		log.Fatalf("unknown method: %s", *method)
	}

	t, err := prayer.Calculate(day, prayer.Coordinates{Lat: *lat, Lng: *lng, Elevation: *elev}, m)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s — %s (%.4f, %.4f)\n", day.Format(time.DateOnly), m.Name, *lat, *lng)
	for _, r := range []struct {
		name string
		t    time.Time
	}{
		{"Imsak", t.Imsak}, {"Fajr", t.Fajr}, {"Sunrise", t.Sunrise}, {"Dhuha", t.Dhuha},
		{"Dhuhr", t.Dhuhr}, {"Asr", t.Asr}, {"Maghrib", t.Maghrib}, {"Isha", t.Isha},
	} {
		fmt.Printf("  %-8s %s\n", r.name, r.t.Format("15:04"))
	}
}
