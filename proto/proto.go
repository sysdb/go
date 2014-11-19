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

// Package proto provides helper functions for using the SysDB front-end
// protocol. That's the protocol used for communication between a client and a
// SysDB server instance.
package proto

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// Network byte order.
var nbo = binary.BigEndian

// A Status represents the type of a message. The message type describes the
// current status or state of a connection depending on the context.
type Status uint32

const (
	// ConnectionOK indicates that a command was successful.
	ConnectionOK = Status(0)
	// ConnectionError indicates that a command has failed.
	ConnectionError = Status(1)
	// ConnectionLog indicates an (asynchronous) log message.
	ConnectionLog = Status(2)

	// ConnectionData indicates a successful query returning data.
	ConnectionData = Status(100)
)

const (
	// ConnectionIdle is the internal state for idle connections.
	ConnectionIdle = Status(0)
	// ConnectionPing is the state requesting a connection check.
	ConnectionPing = Status(1)
	// ConnectionStartup is the state requesting the setup of a client
	// connection.
	ConnectionStartup = Status(2)

	// ConnectionQuery is the state requesting the execution of a query in the
	// server.
	ConnectionQuery = Status(3)
	// ConnectionFetch is the state requesting the execution of the 'FETCH'
	// command in the server.
	ConnectionFetch = Status(4)
	// ConnectionList is the state requesting the execution of the 'LIST'
	// command in the server.
	ConnectionList = Status(5)
	// ConnectionLookup is the state requesting the execution of the 'LOOKUP'
	// command in the server.
	ConnectionLookup = Status(6)
	// ConnectionTimeseries is the state requesting the execution of the
	// 'TIMESERIES' command in the server.
	ConnectionTimeseries = Status(7)

	// ConnectionMatcher is the internal state for parsing matchers.
	ConnectionMatcher = Status(100)
	// ConnectionExpr is the internal state for parsing expressions.
	ConnectionExpr = Status(101)
)

// The DataType describes the type of data in a ConnectionData message.
type DataType int

const (
	// A HostList can be unmarshaled to []sysdb.Host.
	HostList DataType = iota
	// A Host can be unmarshaled to sysdb.Host.
	Host
	// A Timeseries can be unmarshaled to sysdb.Timeseries.
	Timeseries
)

// A Message represents a raw message of the SysDB front-end protocol.
type Message struct {
	Type Status
	Raw  []byte
}

// Read reads a raw message encoded in the SysDB wire format from r. The
// function parses the header but the raw body of the message will still be
// encoded in the wire format.
//
// The reader has to be in blocking mode. Otherwise, the client and server
// will be out of sync after reading a partial message and cannot recover from
// that.
func Read(r io.Reader) (*Message, error) {
	var header [8]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	typ := nbo.Uint32(header[:4])
	l := nbo.Uint32(header[4:])
	msg := make([]byte, l)
	if _, err := io.ReadFull(r, msg); err != nil {
		return nil, err
	}

	return &Message{Status(typ), msg}, nil
}

// Write writes a raw message to w. The raw body of m has to be encoded in the
// SysDB wire format. The function adds the right header to the message.
//
// The writer has to be in blocking mode. Otherwise, the client and server
// will be out of sync after writing a partial message and cannot recover from
// that.
func Write(w io.Writer, m *Message) error {
	var header [8]byte
	nbo.PutUint32(header[:4], uint32(m.Type))
	nbo.PutUint32(header[4:], uint32(len(m.Raw)))

	if _, err := io.WriteString(w, string(header[:])); err != nil {
		return err
	}
	if _, err := io.WriteString(w, string(m.Raw)); err != nil {
		return err
	}
	return nil
}

// DataType determines the type of data in a ConnectionData message.
func (m Message) DataType() (DataType, error) {
	if m.Type != ConnectionData {
		return 0, fmt.Errorf("message is not of type DATA")
	}

	typ := nbo.Uint32(m.Raw[:4])
	switch Status(typ) {
	case ConnectionList, ConnectionLookup:
		return HostList, nil
	case ConnectionFetch:
		return Host, nil
	case ConnectionTimeseries:
		return Timeseries, nil
	}
	return 0, fmt.Errorf("unknown DATA type %d", typ)
}

// Unmarshal parses the raw body of m and stores the result in the value
// pointed to by v which has to match the type of the message and its data.
func Unmarshal(m *Message, v interface{}) error {
	if m.Type != ConnectionData {
		return fmt.Errorf("unmarshaling message of type %d not supported", m.Type)
	}
	if len(m.Raw) == 0 { // empty command
		return nil
	} else if len(m.Raw) < 4 {
		return fmt.Errorf("DATA message body too short")
	}
	return json.Unmarshal(m.Raw[4:], v)
}

// vim: set tw=78 sw=4 sw=4 noexpandtab :
