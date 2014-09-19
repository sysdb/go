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
	"testing"
	"time"
)

func TestMarshalDuration(t *testing.T) {
	for _, test := range []struct {
		d        Duration
		expected string
	}{
		{Duration(0), `"0s"`},
		{Duration(123), `".000000123s"`},
		{Duration(1000123000), `"1.000123s"`},
		{Duration(47940228000000000), `"1Y6M7D"`},
		{Second, `"1s"`},
		{Minute, `"1m"`},
		{Hour, `"1h"`},
		{Day, `"1D"`},
		{Month, `"1M"`},
		{Year, `"1Y"`},
		{Year + Day + Minute, `"1Y1D1m"`},
	} {
		got, err := test.d.MarshalJSON()
		if err != nil || string(got) != test.expected {
			t.Errorf("%s.MarshalJSON() = %s, %v; want %s, <nil>",
				test.d, string(got), err, test.expected)
		}
	}
}

func TestUnmarshalDuration(t *testing.T) {
	for _, test := range []struct {
		data     string
		expected Duration
		err      bool
	}{
		{"0s", 0, true},  // unquoted
		{`"0"`, 0, true}, // missing unit
		{`"0.0"`, 0, true},
		{`".0"`, 0, true},
		{`"s"`, 0, true},   // missing decimal
		{`"1y"`, 0, true},  // invalid unit
		{`"abc"`, 0, true}, // all invalid
		{`"0s"`, 0, false},
		{`"1.0s"`, Second, false},
		{`".5s"`, 500000000, false},
		{`"1.000123s"`, 1000123000, false},
		{`"1.0001234s"`, 1000123400, false},
		{`"1.00012345s"`, 1000123450, false},
		{`"1.000123456s"`, 1000123456, false},
		{`"1.0001234567s"`, 1000123456, false},
		{`"1.000123000123s"`, 1000123000, false},
		{`"1Y6M7D"`, 47940228000000000, false},
		{`"1s"`, Second, false},
		{`"1m"`, Minute, false},
		{`"1h"`, Hour, false},
		{`"1D"`, Day, false},
		{`"1M"`, Month, false},
		{`"1Y"`, Year, false},
	} {
		var d Duration
		err := d.UnmarshalJSON([]byte(test.data))
		if (err != nil) != test.err || d != test.expected {
			e := "<nil>"
			if test.err {
				e = "<err>"
			}
			t.Errorf("UnmarshalJSON(%s) = %v (%s); want %s (%s)",
				test.data, err, d, e, test.expected)
		}
	}
}

func TestMarshalTime(t *testing.T) {
	tm := Time(time.Date(2014, 9, 18, 23, 42, 12, 123, time.UTC))
	expected := `"2014-09-18 23:42:12 +0000"`
	got, err := tm.MarshalJSON()
	if err != nil || string(got) != expected {
		t.Errorf("%s.MarshalJSON() = %s, %v; %s, <nil>", tm, got, err, expected)
	}
}

func TestUnmarshalTime(t *testing.T) {
	for _, test := range []struct {
		data     string
		expected Time
		err      bool
	}{
		{
			`"2014-09-18 23:42:12 +0000"`,
			Time(time.Date(2014, 9, 18, 23, 42, 12, 0, time.UTC)),
			false,
		},
		{
			`2014-09-18 23:42:12 +0000`,
			Time{},
			true,
		},
		{
			`"2014-09-18T23:42:12Z00:00"`,
			Time{},
			true,
		},
	} {
		var tm Time
		err := tm.UnmarshalJSON([]byte(test.data))
		if (err != nil) != test.err || !tm.Equal(test.expected) {
			e := "<nil>"
			if test.err {
				e = "<err>"
			}
			t.Errorf("UnmarshalJSON(%s) = %v (%s); want %s (%s)",
				test.data, err, tm, e, test.expected)
		}
	}
}

// vim: set tw=78 sw=4 sw=4 noexpandtab :
