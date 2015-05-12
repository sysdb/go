//
// Copyright (C) 2015 Sebastian 'tokkee' Harl <sh@tokkee.org>
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

package client

import (
	"testing"
	"time"
)

func TestQueryString(t *testing.T) {
	ts, _ := time.Parse("2006-01-02 15:04:05", "2006-01-02 15:04:05")
	for _, test := range []struct {
		q       string
		args    []interface{}
		want    string
		wantErr bool
	}{
		{"some %s; foo %s", []interface{}{"thing", "bar"}, "some 'thing'; foo 'bar'", false},
		{"s=%s", []interface{}{"'a"}, "s='''a'", false},
		{"t=%s", []interface{}{ts}, "t=2006-01-02 15:04:05", false},
		{"i=%s; f=%s", []interface{}{1234, 47.11}, "i=1234; f=4.711000e+01", false},
		{"t=%d", []interface{}{ts}, "", true},
		{"some %s; foo %s", []interface{}{"a", "b", "c"}, "", true},
		{"some %s; foo %s", []interface{}{"a"}, "", true},
		{"s=%s", []interface{}{`multi
line
text`}, "s='multi\nline\ntext'", false},
		{"s=%d", []interface{}{`multi
line
error`}, "", true},
	} {
		s, err := QueryString(test.q, test.args...)
		if s != test.want || (err != nil) != test.wantErr {
			e := "<nil>"
			if test.wantErr {
				e = "<err>"
			}
			t.Errorf("QueryString(%q, %v) = %q, %v; want %q, %s", test.q, test.args, s, err, test.want, e)
		}
	}
}

// vim: set tw=78 sw=4 sw=4 noexpandtab :
