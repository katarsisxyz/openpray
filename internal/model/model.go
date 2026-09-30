// Package model contains domain types shared across layers.
package model

import (
	"errors"
	"time"

	"github.com/katarsisxyz/openpray/prayer"
)

// ErrNotFound is returned by repositories when a record does not exist.
var ErrNotFound = errors.New("not found")

// Method is a calculation method together with its ID.
type Method struct {
	ID string
	prayer.Method
}

// City is an Indonesian regency (kabupaten) or city (kota).
type City struct {
	ID       string // Kemendagri code, e.g. "35.78"
	Name     string // e.g. "Kota Surabaya"
	Capital  string // administrative center, e.g. "Surabaya"
	Province string
	Coord    prayer.Coordinates // location of the capital
	Timezone string             // IANA zone, e.g. "Asia/Jakarta"
}

// Place is a resolved location to calculate prayer times for.
type Place struct {
	Coord    prayer.Coordinates
	Location *time.Location
	City     *City // set when the place was given as a city
}

// Day holds the prayer times for one date.
type Day struct {
	Date  time.Time
	Times prayer.Times
}

// Schedule holds one or more days, the place and the method used to calculate them.
type Schedule struct {
	Place  Place
	Method Method
	Days   []Day
}
