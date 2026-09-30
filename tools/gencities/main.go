// Command gencities regenerates internal/repository/data/cities.csv from the
// cahyadsn/wilayah dataset (MIT, Kepmendagri administrative codes), pinned to
// a specific commit so the output is reproducible.
//
//	go generate ./internal/repository
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

const (
	commit = "cb05064af1ccc6c6ae5ee33e498ad69a62034a0a"
	source = "https://raw.githubusercontent.com/cahyadsn/wilayah/" + commit + "/db/wilayah_level_1_2.sql"
)

// row matches the leading columns of one VALUES tuple:
// ('kode','nama','ibukota', lat, lng, elv, tz, ...
var row = regexp.MustCompile(`\('(\d{2}(?:\.\d{2})?)','([^']*)','([^']*)',\s*([-\d.eE]+),\s*([-\d.eE]+),\s*([-\d.]+),\s*(\d+),`)

// zones maps the dataset's UTC offset (hours) to the IANA zone used across Indonesia.
var zones = map[string]string{"7": "Asia/Jakarta", "8": "Asia/Makassar", "9": "Asia/Jayapura"}

func main() {
	out := flag.String("o", "data/cities.csv", "output CSV path")
	flag.Parse()

	resp, err := http.Get(source)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("GET %s: %s", source, resp.Status)
	}
	sql, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	provinces := map[string]string{}
	var records [][]string
	for _, m := range row.FindAllStringSubmatch(string(sql), -1) {
		code, name, capital, lat, lng, elv, tz := m[1], m[2], m[3], m[4], m[5], m[6], m[7]
		if len(code) == 2 {
			provinces[code] = name
			continue
		}
		zone, ok := zones[tz]
		if !ok {
			log.Fatalf("%s: unexpected tz %q", code, tz)
		}
		records = append(records, []string{code, name, capital, "", round(lat), round(lng), elv, zone})
	}
	for _, r := range records {
		p, ok := provinces[r[0][:2]]
		if !ok {
			log.Fatalf("%s: unknown province", r[0])
		}
		r[3] = p
	}
	if len(records) == 0 {
		log.Fatal("no regencies found; has the dataset format changed?")
	}

	f, err := os.Create(*out)
	if err != nil {
		log.Fatal(err)
	}
	w := csv.NewWriter(f)
	w.Write([]string{"id", "name", "capital", "province", "lat", "lng", "elevation", "timezone"})
	w.WriteAll(records)
	if err := w.Error(); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %d cities from %d provinces to %s\n", len(records), len(provinces), *out)
}

// round keeps 6 decimals (~0.1 m), plenty for prayer times.
func round(s string) string {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatal(err)
	}
	return strconv.FormatFloat(v, 'f', 6, 64)
}
