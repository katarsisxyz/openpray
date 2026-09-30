package prayer

import (
	"maps"
	"slices"
)

// Method holds the parameters of a prayer time calculation method.
type Method struct {
	Name         string
	FajrAngle    float64 // sun depression below the horizon for Fajr (degrees)
	IshaAngle    float64 // sun depression below the horizon for Isha (degrees)
	DhuhaAngle   float64 // sun altitude above the horizon for Dhuha (degrees)
	AsrFactor    float64 // shadow factor: 1 = Shafi'i/majority, 2 = Hanafi
	RiseAngle    float64 // horizon angle for sunrise/sunset; 0 = 0.833° + elevation correction
	Ihtiyat      int     // precautionary margin (minutes); subtracted from sunrise, added to other times
	ImsakMinutes int     // imsak = fajr - ImsakMinutes
	Adjust       Adjustments
}

// Adjustments are per-prayer minute corrections, applied after ihtiyat.
// Useful for matching the schedule of a specific institution or mosque.
type Adjustments struct {
	Imsak, Fajr, Sunrise, Dhuha, Dhuhr, Asr, Maghrib, Isha int
}

var (
	// Kemenag follows the Indonesian Ministry of Religious Affairs (Kemenag RI).
	// RiseAngle and Adjust are calibrated against official Kemenag schedules
	// (see cmd/compare): 92.8% exact matches, the rest within 1 minute.
	Kemenag = Method{
		Name:         "Kemenag RI",
		FajrAngle:    20,
		IshaAngle:    18,
		DhuhaAngle:   4.5,
		AsrFactor:    1,
		RiseAngle:    1.0,
		Ihtiyat:      2,
		ImsakMinutes: 10,
		Adjust:       Adjustments{Dhuhr: 1},
	}

	// Muhammadiyah has used a Fajr angle of 18° since the 2020 Tarjih conference.
	// Ihtiyat, RiseAngle and Adjust are calibrated against Muhammadiyah's KHGT
	// schedules (see cmd/compare): 96.6% exact matches, the rest within 1 minute.
	// KHGT does not publish imsak or dhuha; those follow the Kemenag convention.
	Muhammadiyah = Method{
		Name:         "Muhammadiyah",
		FajrAngle:    18,
		IshaAngle:    18,
		DhuhaAngle:   4.5,
		AsrFactor:    1,
		RiseAngle:    1.08,
		Ihtiyat:      1,
		ImsakMinutes: 10,
		Adjust:       Adjustments{Sunrise: -1},
	}

	// MWL is the Muslim World League method, without ihtiyat.
	MWL = Method{
		Name:         "Muslim World League",
		FajrAngle:    18,
		IshaAngle:    17,
		DhuhaAngle:   4.5,
		AsrFactor:    1,
		ImsakMinutes: 10,
	}
)

var methods = map[string]Method{
	"kemenag":      Kemenag,
	"muhammadiyah": Muhammadiyah,
	"mwl":          MWL,
}

// MethodByID returns a built-in method by ID, e.g. "kemenag".
// For custom parameters, construct a Method value directly.
func MethodByID(id string) (Method, bool) {
	m, ok := methods[id]
	return m, ok
}

// MethodIDs returns the IDs of all built-in methods, sorted.
func MethodIDs() []string {
	return slices.Sorted(maps.Keys(methods))
}
