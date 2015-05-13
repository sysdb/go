//
// Copyright (C) 2014-2015 Sebastian 'tokkee' Harl <sh@tokkee.org>
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
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/sysdb/go/proto"
	"github.com/sysdb/go/sysdb"
)

// An Identifier is a string that may not be quoted or escaped in a query.
type Identifier string

// The default format for date-time values.
var dtFormat = "2006-01-02 15:04:05"

func stringify(values ...interface{}) ([]interface{}, error) {
	str := make([]interface{}, len(values))
	for i, v := range values {
		switch val := v.(type) {
		case uint8, uint16, uint32, uint64, int8, int16, int32, int64, int:
			str[i] = fmt.Sprintf("%d", val)
		case float32, float64:
			str[i] = fmt.Sprintf("%e", val)
		case Identifier:
			str[i] = string(val)
		case string:
			str[i] = proto.EscapeString(val)
		case time.Time:
			str[i] = val.Format(dtFormat)
		default:
			return nil, fmt.Errorf("cannot embed value %v of type %T in query", v, v)
		}
	}
	return str, nil
}

// The fmt package does not expose these errors except through the formatted
// string. Let's just assume that this pattern never occurs in a real query
// string (or else, users will have to work around this by not using
// QueryString()).
var badArgRE = regexp.MustCompile(`%!?[A-Za-z]?\(.+`)

// QueryString formats a query string. The query q may include printf string
// verbs (%s) for each argument. The arguments may be of type Identifier,
// string, or time.Time and will be formatted to make them suitable for use in
// a query.
//
// This function tries to prevent injection attacks but it's not fool-proof.
// It will go away once the SysDB network protocol supports arguments to
// queries.
func QueryString(q string, args ...interface{}) (string, error) {
	args, err := stringify(args...)
	if err != nil {
		return "", err
	}

	str := fmt.Sprintf(q, args...)

	// Try to identify format string errors.
	if e := badArgRE.Find([]byte(str)); e != nil {
		return "", errors.New(string(e))
	}

	return str, nil
}

// Query executes a query on the server. It returns a sysdb object on success.
func (c *Client) Query(q string) (interface{}, error) {
	res, err := c.Call(&proto.Message{
		Type: proto.ConnectionQuery,
		Raw:  []byte(q),
	})
	if err != nil {
		return nil, err
	}
	if res.Type != proto.ConnectionData {
		return nil, fmt.Errorf("unexpected result type %d", res.Type)
	}

	t, err := res.DataType()
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	var obj interface{}
	switch t {
	case proto.HostList:
		var hosts []sysdb.Host
		err = proto.Unmarshal(res, &hosts)
		obj = hosts
	case proto.Host:
		var host sysdb.Host
		err = proto.Unmarshal(res, &host)
		obj = &host
	case proto.Timeseries:
		var ts sysdb.Timeseries
		err = proto.Unmarshal(res, &ts)
		obj = &ts
	default:
		return nil, fmt.Errorf("unsupported data type %d", t)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}
	return obj, nil
}

// vim: set tw=78 sw=4 sw=4 noexpandtab :
