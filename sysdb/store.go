//
// Copyright (C) 2014 Sebastian 'tokkee' Harl <sh@tokkee.org>
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// ``AS IS'' AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED
// TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR
// PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDERS OR
// CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL,
// EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
// PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS;
// OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY,
// WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR
// OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF
// ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package sysdb

import (
	"fmt"
	"time"
)

// The SysDB JSON time format.
const jsonTime = `"2006-01-02 15:04:05 -0700"`

// A Duration represents the elapsed time between two instants as a
// nanoseconds count.
//
// It supports marshaling to and unmarshaling from the SysDB JSON format (a
// sequence of decimal numbers with a unit suffix).
type Duration time.Duration

const (
	Second = Duration(1000000000)
	Minute = 60 * Second
	Hour   = 60 * Minute
	Day    = 24 * Hour
	Month  = Duration(30436875 * 24 * 60 * 60 * 1000)
	Year   = Duration(3652425 * 24 * 60 * 60 * 100000)
)

// MarshalJSON implements the json.Marshaler interface. The time is a quoted
// string in the SysDB JSON format.
func (d Duration) MarshalJSON() ([]byte, error) {
	if d == 0 {
		return []byte(`"0s"`), nil
	}

	s := `"`
	secs := false
	for _, spec := range []struct {
		interval Duration
		suffix   string
	}{{Year, "Y"}, {Month, "M"}, {Day, "D"}, {Hour, "h"}, {Minute, "m"}, {Second, ""}} {
		if d >= spec.interval {
			s += fmt.Sprintf("%d%s", d/spec.interval, spec.suffix)
			d %= spec.interval
			if spec.interval == Second {
				secs = true
			}
		}
	}

	if d > 0 {
		s += fmt.Sprintf(".%09d", d)
		for i := len(s) - 1; i > 0; i-- {
			if s[i] != '0' {
				break
			}
			s = s[:i]
		}
		secs = true
	}
	if secs {
		s += "s"
	}
	s += `"`
	return []byte(s), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The duration is
// expected to be a quoted string in the SysDB JSON format.
func (d *Duration) UnmarshalJSON(data []byte) error {
	m := map[string]Duration{
		"Y": Year,
		"M": Month,
		"D": Day,
		"h": Hour,
		"m": Minute,
		"s": Second,
	}

	if data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("unquoted duration %q", string(data))
	}
	data = data[1 : len(data)-1]

	orig := string(data)
	var res Duration
	for len(data) != 0 {
		// consume digits
		n := 0
		dec := 0
		frac := false
		for n < len(data) && '0' <= data[n] && data[n] <= '9' {
			dec = dec*10 + int(data[n]-'0')
			n++
		}
		if n < len(data) && data[n] == '.' {
			frac = true
			n++

			// consume fraction
			m := 1000000000
			for n < len(data) && '0' <= data[n] && data[n] <= '9' {
				if m > 1 { // cut of to nanoseconds
					dec = dec*10 + int(data[n]-'0')
					m /= 10
				}
				n++
			}
			dec *= m
		}
		if n >= len(data) {
			return fmt.Errorf("missing unit in duration %q", orig)
		}
		if n == 0 {
			// we found something which is not a number
			return fmt.Errorf("invalid duration %q", orig)
		}

		// consume unit
		u := n
		for u < len(data) && data[u] != '.' && (data[u] < '0' || '9' < data[u]) {
			u++
		}

		unit := string(data[n:u])
		data = data[u:]

		// convert to Duration
		d, ok := m[unit]
		if !ok {
			return fmt.Errorf("invalid unit %q in duration %q", unit, orig)
		}

		if d == Second {
			if frac {
				d = 1
			}
		} else if frac {
			return fmt.Errorf("invalid fraction %q%s in duration %q", dec, unit, orig)
		}

		res += Duration(dec) * d
	}
	*d = res
	return nil
}

// String returns the duration formatted using a predefined format string.
func (d Duration) String() string { return time.Duration(d).String() }

// A Time represents an instant in time with nanosecond precision.
//
// It supports marshaling to and unmarshaling from the SysDB JSON format
// (YYYY-MM-DD hh:mm:ss +-zzzz).
type Time time.Time

func (t Time) MarshalJSON() ([]byte, error) {
	return []byte(time.Time(t).Format(jsonTime)), nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. The time is
// expected to be a quoted string in the SysDB JSON format.
func (t *Time) UnmarshalJSON(data []byte) error {
	parsed, err := time.Parse(jsonTime, string(data))
	if err == nil {
		*t = Time(parsed)
	}
	return err
}

// Equal reports whether t and u represent the same time instant.
func (t Time) Equal(u Time) bool {
	return time.Time(t).Equal(time.Time(u))
}

// String returns the time formatted using a predefined format string.
func (t Time) String() string { return time.Time(t).String() }

// An Attribute describes a host, metric, or service attribute.
type Attribute struct {
	Name           string   `json:"name"`
	Value          string   `json:"value"`
	LastUpdate     Time     `json:"last_update"`
	UpdateInterval Duration `json:"update_interval"`
	Backends       []string `json:"backends"`
}

// A Metric describes a metric known to SysDB.
type Metric struct {
	Name           string      `json:"name"`
	LastUpdate     Time        `json:"last_update"`
	UpdateInterval Duration    `json:"update_interval"`
	Backends       []string    `json:"backends"`
	Attributes     []Attribute `json:"attributes"`
}

// A Service describes a service object stored in the SysDB store.
type Service struct {
	Name           string      `json:"name"`
	LastUpdate     Time        `json:"last_update"`
	UpdateInterval Duration    `json:"update_interval"`
	Backends       []string    `json:"backends"`
	Attributes     []Attribute `json:"attributes"`
}

// A Host describes a host object stored in the SysDB store.
type Host struct {
	Name           string      `json:"name"`
	LastUpdate     Time        `json:"last_update"`
	UpdateInterval Duration    `json:"update_interval"`
	Backends       []string    `json:"backends"`
	Attributes     []Attribute `json:"attributes"`
	Metrics        []Metric    `json:"metrics"`
	Services       []Service   `json:"services"`
}

// A DataPoint describes a datum at a certain point of time.
type DataPoint struct {
	Timestamp Time    `json:"timestamp"`
	Value     float64 `json:"value,string"`
}

// A Timeseries describes a sequence of data-points.
type Timeseries struct {
	Start Time                   `json:"start"`
	End   Time                   `json:"end"`
	Data  map[string][]DataPoint `json:"data"`
}

// vim: set tw=78 sw=4 sw=4 noexpandtab :
