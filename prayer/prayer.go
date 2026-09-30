// Package prayer calculates Islamic prayer times astronomically, with no
// external API and no dependencies outside the standard library.
//
//	loc, _ := time.LoadLocation("Asia/Jakarta")
//	date := time.Date(2026, 9, 30, 0, 0, 0, 0, loc)
//	t, err := prayer.Calculate(date, prayer.Coordinates{Lat: -6.1754, Lng: 106.8272}, prayer.Kemenag)
//
// Results are in date.Location(). The solar position algorithm follows
// PrayTimes.org; the built-in Indonesian methods are calibrated against
// official schedules (see cmd/compare).
package prayer

import (
	"errors"
	"math"
	"time"
)

// Coordinates is the observer's location.
type Coordinates struct {
	Lat       float64 // latitude, negative for south
	Lng       float64 // longitude, positive for east
	Elevation float64 // meters above sea level (optional)
}

// Times holds one day's prayer times in the local time zone.
type Times struct {
	Imsak   time.Time
	Fajr    time.Time
	Sunrise time.Time
	Dhuha   time.Time
	Dhuhr   time.Time
	Asr     time.Time
	Maghrib time.Time
	Isha    time.Time
}

// ErrUndefined is returned at high latitudes when the sun never reaches the required angle.
var ErrUndefined = errors.New("prayer: times are undefined at this latitude")

// Calculate returns the prayer times for date's calendar day in date.Location().
func Calculate(date time.Time, c Coordinates, m Method) (Times, error) {
	y, mo, d := date.Date()
	loc := date.Location()
	midnight := time.Date(y, mo, d, 0, 0, 0, 0, loc)
	_, offset := time.Date(y, mo, d, 12, 0, 0, 0, loc).Zone()

	s := solar{
		jd:  julianDate(y, int(mo), d) - c.Lng/(15*24),
		lat: c.Lat,
	}

	// Initial estimates (hours), refined by iteration.
	fajr, sunrise, dhuha, dhuhr, asr, sunset, isha := 5.0, 6.0, 6.5, 12.0, 13.0, 18.0, 18.0
	riseAngle := m.RiseAngle
	if riseAngle == 0 {
		riseAngle = 0.833 + 0.0347*math.Sqrt(c.Elevation)
	}
	for range 2 {
		fajr = s.angleTime(m.FajrAngle, fajr, true)
		sunrise = s.angleTime(riseAngle, sunrise, true)
		dhuha = s.angleTime(-m.DhuhaAngle, dhuha, true)
		dhuhr = s.midDay(dhuhr)
		asr = s.asrTime(m.AsrFactor, asr)
		sunset = s.angleTime(riseAngle, sunset, false)
		isha = s.angleTime(m.IshaAngle, isha, false)
	}
	for _, v := range []float64{fajr, sunrise, dhuha, asr, sunset, isha} {
		if math.IsNaN(v) {
			return Times{}, ErrUndefined
		}
	}

	// Convert from local solar time to zone time.
	shift := float64(offset)/3600 - c.Lng/15
	ihtiyat := time.Duration(m.Ihtiyat) * time.Minute
	at := func(h float64) time.Time {
		return midnight.Add(time.Duration((h + shift) * float64(time.Hour)))
	}
	// Kemenag convention: prayer start times are rounded up; sunrise (the end
	// of Fajr) has ihtiyat subtracted and is rounded down.
	after := func(h float64, adj int) time.Time { return ceilMinute(at(h).Add(ihtiyat)).Add(minutes(adj)) }

	a := m.Adjust
	t := Times{
		Fajr:    after(fajr, a.Fajr),
		Sunrise: at(sunrise).Add(-ihtiyat).Truncate(time.Minute).Add(minutes(a.Sunrise)),
		Dhuha:   after(dhuha, a.Dhuha),
		Dhuhr:   after(dhuhr, a.Dhuhr),
		Asr:     after(asr, a.Asr),
		Maghrib: after(sunset, a.Maghrib),
		Isha:    after(isha, a.Isha),
	}
	t.Imsak = t.Fajr.Add(minutes(a.Imsak - m.ImsakMinutes))
	return t, nil
}

func minutes(n int) time.Duration { return time.Duration(n) * time.Minute }

func ceilMinute(t time.Time) time.Time {
	r := t.Truncate(time.Minute)
	if r.Before(t) {
		r = r.Add(time.Minute)
	}
	return r
}

// solar holds the astronomical context for one day at one location.
type solar struct {
	jd  float64 // Julian Date at local solar midnight
	lat float64
}

// position returns the sun's declination (degrees) and the equation of time (hours).
func (s solar) position(hour float64) (decl, eqt float64) {
	d := s.jd + hour/24 - 2451545.0
	g := fixAngle(357.529 + 0.98560028*d)
	q := fixAngle(280.459 + 0.98564736*d)
	l := fixAngle(q + 1.915*dsin(g) + 0.020*dsin(2*g))
	e := 23.439 - 0.00000036*d

	ra := darctan2(dcos(e)*dsin(l), dcos(l)) / 15
	eqt = q/15 - fixHour(ra)
	decl = darcsin(dsin(e) * dsin(l))
	return decl, eqt
}

// midDay is solar transit (sun on the meridian), in local solar hours.
func (s solar) midDay(hour float64) float64 {
	_, eqt := s.position(hour)
	return fixHour(12 - eqt)
}

// angleTime is when the sun is `angle` degrees below the horizon,
// before transit if ccw, after it otherwise.
func (s solar) angleTime(angle, hour float64, ccw bool) float64 {
	decl, _ := s.position(hour)
	noon := s.midDay(hour)
	cosH := (-dsin(angle) - dsin(decl)*dsin(s.lat)) / (dcos(decl) * dcos(s.lat))
	if cosH < -1 || cosH > 1 {
		return math.NaN()
	}
	t := darccos(cosH) / 15
	if ccw {
		return noon - t
	}
	return noon + t
}

// asrTime: shadow length = factor × object height + shadow length at transit.
func (s solar) asrTime(factor, hour float64) float64 {
	decl, _ := s.position(hour)
	angle := -darccot(factor + dtan(math.Abs(s.lat-decl)))
	return s.angleTime(angle, hour, false)
}

func julianDate(year, month, day int) float64 {
	if month <= 2 {
		year--
		month += 12
	}
	a := math.Floor(float64(year) / 100)
	b := 2 - a + math.Floor(a/4)
	return math.Floor(365.25*float64(year+4716)) + math.Floor(30.6001*float64(month+1)) + float64(day) + b - 1524.5
}

// Degree-based trigonometry helpers.
func rad(d float64) float64         { return d * math.Pi / 180 }
func deg(r float64) float64         { return r * 180 / math.Pi }
func dsin(d float64) float64        { return math.Sin(rad(d)) }
func dcos(d float64) float64        { return math.Cos(rad(d)) }
func dtan(d float64) float64        { return math.Tan(rad(d)) }
func darcsin(x float64) float64     { return deg(math.Asin(x)) }
func darccos(x float64) float64     { return deg(math.Acos(x)) }
func darctan2(y, x float64) float64 { return deg(math.Atan2(y, x)) }
func darccot(x float64) float64     { return deg(math.Atan(1 / x)) }
func fixAngle(a float64) float64    { return fix(a, 360) }
func fixHour(h float64) float64     { return fix(h, 24) }
func fix(a, b float64) float64 {
	a = a - b*math.Floor(a/b)
	if a < 0 {
		a += b
	}
	return a
}
