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

// Day holds the prayer times for one date.
type Day struct {
	Date  time.Time
	Times prayer.Times
}

// Schedule holds one or more days and the method used to calculate them.
type Schedule struct {
	Method Method
	Days   []Day
}
