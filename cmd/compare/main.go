// Command compare checks the calculated times against an official schedule
// and prints how far off each prayer time is.
//
//	go run ./cmd/compare -source kemenag -year 2026 -months 1,4,7,10
//	go run ./cmd/compare -source muhammadiyah -year 2026 -months 9 -days 1,15
//
// Reference sources:
//   - kemenag: Kemenag schedules per city/regency, via api.myquran.com
//   - muhammadiyah: Muhammadiyah KHGT schedules per coordinate (khgt.muhammadiyah.or.id);
//     the endpoint only serves the current month
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/katarsisxyz/openpray/prayer"
)

type city struct {
	id, name, tz string // id: city ID on api.myquran.com
	coord        prayer.Coordinates
}

var cities = []city{
	{"0119", "Banda Aceh", "Asia/Jakarta", prayer.Coordinates{Lat: 5.5483, Lng: 95.3238}},
	{"1301", "Jakarta", "Asia/Jakarta", prayer.Coordinates{Lat: -6.1754, Lng: 106.8272}},
	{"1638", "Surabaya", "Asia/Jakarta", prayer.Coordinates{Lat: -7.2575, Lng: 112.7521}},
	{"2622", "Makassar", "Asia/Makassar", prayer.Coordinates{Lat: -5.1477, Lng: 119.4327}},
	{"3329", "Jayapura", "Asia/Jayapura", prayer.Coordinates{Lat: -2.5337, Lng: 140.7181}},
}

// refDay is one day of a reference schedule; times[field] = "HH:MM".
type refDay struct {
	date  string
	times map[string]string
}

type source struct {
	label  string
	method prayer.Method
	fields []string // times published by the source
	fetch  func(c city, year, month int, days []int) ([]refDay, error)
}

var sources = map[string]source{
	"kemenag": {
		label:  "Kemenag (api.myquran.com)",
		method: prayer.Kemenag,
		fields: []string{"imsak", "fajr", "sunrise", "dhuha", "dhuhr", "asr", "maghrib", "isha"},
		fetch:  fetchKemenag,
	},
	"muhammadiyah": {
		label:  "Muhammadiyah (KHGT)",
		method: prayer.Muhammadiyah,
		fields: []string{"fajr", "sunrise", "dhuhr", "asr", "maghrib", "isha"},
		fetch:  fetchKHGT,
	},
}

func main() {
	src := flag.String("source", "kemenag", "reference source: kemenag | muhammadiyah")
	year := flag.Int("year", 2026, "year")
	months := flag.String("months", "1,4,7,10", "comma-separated months")
	days := flag.String("days", "1,15", "days of the month to fetch (per-day sources such as muhammadiyah only)")
	rise := flag.Float64("rise", -1, "override the sunrise/sunset horizon angle (-1 = method default)")
	ihtiyat := flag.Int("ihtiyat", -1, "override ihtiyat in minutes (-1 = method default)")
	adjust := flag.String("adjust", "", "override minute adjustments, e.g. dhuhr=1,sunrise=-1")
	flag.Parse()

	s, ok := sources[*src]
	if !ok {
		log.Fatalf("unknown source: %q", *src)
	}
	method := s.method
	if *rise >= 0 {
		method.RiseAngle = *rise
	}
	if *ihtiyat >= 0 {
		method.Ihtiyat = *ihtiyat
	}
	if *adjust != "" {
		setAdjustments(&method.Adjust, *adjust)
	}
	dayList := parseInts(*days)

	var total, exact int
	for _, c := range cities {
		loc, err := time.LoadLocation(c.tz)
		if err != nil {
			log.Fatal(err)
		}
		// diffs[field][difference in minutes] = number of days
		diffs := map[string]map[int]int{}
		for _, mo := range parseInts(*months) {
			refs, err := s.fetch(c, *year, mo, dayList)
			if err != nil {
				log.Fatalf("%s %d-%02d: %v", c.name, *year, mo, err)
			}
			for _, ref := range refs {
				date, err := time.ParseInLocation(time.DateOnly, ref.date, loc)
				if err != nil {
					log.Fatal(err)
				}
				t, err := prayer.Calculate(date, c.coord, method)
				if err != nil {
					log.Fatal(err)
				}
				ours := map[string]time.Time{
					"imsak": t.Imsak, "fajr": t.Fajr, "sunrise": t.Sunrise, "dhuha": t.Dhuha,
					"dhuhr": t.Dhuhr, "asr": t.Asr, "maghrib": t.Maghrib, "isha": t.Isha,
				}
				for _, f := range s.fields {
					want, err := time.ParseInLocation("2006-01-02 15:04", ref.date+" "+ref.times[f], loc)
					if err != nil {
						log.Fatalf("%s %s %s: %v", c.name, ref.date, f, err)
					}
					d := int(ours[f].Sub(want).Minutes())
					if diffs[f] == nil {
						diffs[f] = map[int]int{}
					}
					diffs[f][d]++
					total++
					if d == 0 {
						exact++
					}
				}
			}
		}
		fmt.Printf("\n%s (ours − %s, minutes: days)\n", c.name, s.label)
		for _, f := range s.fields {
			fmt.Printf("  %-8s %s\n", f, histogram(diffs[f]))
		}
	}
	fmt.Printf("\nTotal: %d/%d exact matches (%.1f%%)\n", exact, total, 100*float64(exact)/float64(total))
}

// fetchKemenag fetches a whole month for one city/regency.
func fetchKemenag(c city, year, month int, _ []int) ([]refDay, error) {
	var r struct {
		Status bool `json:"status"`
		Data   struct {
			Jadwal []map[string]string `json:"jadwal"`
		} `json:"data"`
	}
	url := fmt.Sprintf("https://api.myquran.com/v2/sholat/jadwal/%s/%d/%02d", c.id, year, month)
	key := fmt.Sprintf("myquran-%s-%d-%02d", c.id, year, month)
	err := cachedGet(key, url, &r, func() error {
		if !r.Status || len(r.Data.Jadwal) == 0 {
			return errors.New("empty schedule")
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	refs := make([]refDay, len(r.Data.Jadwal))
	for i, j := range r.Data.Jadwal {
		refs[i] = refDay{date: j["date"], times: map[string]string{
			"imsak": j["imsak"], "fajr": j["subuh"], "sunrise": j["terbit"], "dhuha": j["dhuha"],
			"dhuhr": j["dzuhur"], "asr": j["ashar"], "maghrib": j["maghrib"], "isha": j["isya"],
		}}
	}
	return refs, nil
}

// fetchKHGT fetches single days for exactly the same coordinates we calculate for.
func fetchKHGT(c city, year, month int, days []int) ([]refDay, error) {
	var refs []refDay
	for _, d := range days {
		// Skip days the month doesn't have (e.g. September 31).
		if t := time.Date(year, time.Month(month), d, 0, 0, 0, 0, time.UTC); t.Month() != time.Month(month) {
			continue
		}
		date := fmt.Sprintf("%d-%02d-%02d", year, month, d)
		var r struct {
			Times map[string]string `json:"times"`
		}
		url := fmt.Sprintf("https://khgt.muhammadiyah.or.id/prayer?lat=%g&long=%g&date=%s", c.coord.Lat, c.coord.Lng, date)
		key := fmt.Sprintf("khgt-%g-%g-%s", c.coord.Lat, c.coord.Lng, date)
		err := cachedGet(key, url, &r, func() error {
			if r.Times["subuh"] == "" {
				return errors.New("empty times")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		t := r.Times
		refs = append(refs, refDay{date: date, times: map[string]string{
			"fajr": t["subuh"], "sunrise": t["terbit"], "dhuhr": t["zuhur"],
			"asr": t["ashar"], "maghrib": t["maghrib"], "isha": t["isya"],
		}})
	}
	return refs, nil
}

// cachedGet fetches JSON from url into out, with an on-disk cache, a polite
// delay between requests, and retries on invalid responses (e.g. rate limits).
func cachedGet(key, url string, out any, validate func() error) error {
	dir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	cache := filepath.Join(dir, "openpray", key+".json")
	if body, err := os.ReadFile(cache); err == nil {
		return json.Unmarshal(body, out)
	}

	var body []byte
	for attempt := 1; ; attempt++ {
		time.Sleep(300 * time.Millisecond)
		body, err = get(url)
		if err == nil {
			if err = json.Unmarshal(body, out); err == nil {
				err = validate()
			}
		}
		if err == nil {
			break
		}
		if attempt == 5 {
			return fmt.Errorf("%s: %w (%.80s)", url, err, body)
		}
		time.Sleep(time.Duration(attempt) * 2 * time.Second)
	}
	if err := os.MkdirAll(filepath.Dir(cache), 0o755); err != nil {
		return err
	}
	return os.WriteFile(cache, body, 0o644)
}

func get(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// setAdjustments overrides the fields named in s ("dhuhr=1,sunrise=-1"), leaving the rest.
func setAdjustments(a *prayer.Adjustments, s string) {
	fields := map[string]*int{
		"imsak": &a.Imsak, "fajr": &a.Fajr, "sunrise": &a.Sunrise, "dhuha": &a.Dhuha,
		"dhuhr": &a.Dhuhr, "asr": &a.Asr, "maghrib": &a.Maghrib, "isha": &a.Isha,
	}
	for kv := range strings.SplitSeq(s, ",") {
		k, v, _ := strings.Cut(kv, "=")
		p, ok := fields[strings.TrimSpace(k)]
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if !ok || err != nil {
			log.Fatalf("invalid adjustment: %q", kv)
		}
		*p = n
	}
}

func parseInts(s string) []int {
	var out []int
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p == "" {
			continue
		}
		v, err := strconv.Atoi(p)
		if err != nil {
			log.Fatalf("not a number: %q", p)
		}
		out = append(out, v)
	}
	return out
}

func histogram(h map[int]int) string {
	keys := make([]int, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%+d:%d", k, h[k])
	}
	return strings.Join(parts, "  ")
}
