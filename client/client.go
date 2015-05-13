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

/*
Package client provides a SysDB client implementation.

The Connect function connects to a SysDB server as the specified user:

	c, err := client.Connect("unix:/var/run/sysdbd.sock", "username")
	if err != nil {
		// handle error
	}
	defer c.Close()

Then, it can issue requests to the server:

	major, minor, patch, extra, err := c.ServerVersion()
	if err != nil {
		// handle error
	}
	fmt.Printf("Connected to SysDB %d.%d.%d%s\n", major, minor, patch, extra)

or:

	res, err := c.Call(&proto.Message{Type: proto.ConnectionServerVersion})
	if err != nil {
		// handle error
	}
	fmt.Printf("%v\n", res)

or execute queries:

	q, err := client.QueryString("FETCH %s %s", client.Identifier(typ), name)
	if err != nil {
		// handle error
	}
	res, err := c.Query(q)
	if err != nil {
		// handle error
	}

	// res is one of the object types defined in the sysdb package.
	switch typ {
	case "host":
		host := res.(*sysdb.Host)
		// do something
		// ...
	}

Each client maintains multiple connections to a SysDB server allowing to
handle multiple requests in parallel. The SysDB server is able to handle that
easily making it a cheap approach. The low-level Dial function creates a
single connection to a SysDB server allowing to perform low-level operations:

	conn, err := client.Dial("unix:/var/run/sysdbd.sock", "username")
	if err != nil {
		// handle error
	}
	defer conn.Close()

The github.com/sysdb/go/proto package provides support for handling requests
and responses. Use the Send and Receive functions to communicate with the
server:

	m := &proto.Message{
		Type: proto.ConnectionQuery,
		Raw:  []byte{"LOOKUP hosts MATCHING attribute.architecture = 'amd64';"},
	}
	if err := conn.Send(m); err != nil {
		// handle error
	}
	m, err := conn.Receive()
	if err != nil {
		// handle error
	}
	if m.Type == proto.ConnectionError {
		// handle failed query
	}
	// ...
*/
package client

import (
	"encoding/binary"
	"fmt"
	"log"
	"runtime"

	"github.com/sysdb/go/proto"
)

// A Client is a client for SysDB.
//
// A client may be used from multiple goroutines in parallel.
type Client struct {
	conns chan *Conn
}

// Connect creates a new client connected to a SysDB server instance at the
// specified address using the specified user.
//
// The address may be a IP address or a UNIX domain socket, either prefixed
// with 'unix:' or specifying an absolute file-system path.
func Connect(addr, user string) (*Client, error) {
	c := &Client{conns: make(chan *Conn, 2*runtime.NumCPU())}

	for i := 0; i < cap(c.conns); i++ {
		conn, err := Dial(addr, user)
		if err != nil {
			return nil, err
		}
		c.conns <- conn
	}
	return c, nil
}

// Close closes a client connection. It may not be further used after calling
// this function.
//
// The function waits for all pending operations to finish.
func (c *Client) Close() {
	for i := 0; i < cap(c.conns); i++ {
		conn := <-c.conns
		conn.Close()
	}
	close(c.conns)
	c.conns = nil
}

// Call sends the specified request to the server and waits for its reply. It
// blocks until the full reply has been received.
func (c *Client) Call(req *proto.Message) (*proto.Message, error) {
	conn := <-c.conns
	defer func() { c.conns <- conn }()

	err := conn.Send(req)
	if err != nil {
		return nil, err
	}

	for {
		res, err := conn.Receive()
		switch {
		case err != nil:
			return nil, err
		case res.Type == proto.ConnectionError:
			return nil, fmt.Errorf("request failed: %s", string(res.Raw))
		case res.Type != proto.ConnectionLog:
			return res, err
		}

		if len(res.Raw) > 4 {
			log.Println(string(res.Raw[4:]))
		}
	}

	// Not reached; needed for Go1 compatibility.
	return nil, nil
}

// ServerVersion queries and returns the version of the remote server.
func (c *Client) ServerVersion() (major, minor, patch int, extra string, err error) {
	res, err := c.Call(&proto.Message{Type: proto.ConnectionServerVersion})
	if err != nil || res.Type != proto.ConnectionOK {
		if err == nil {
			err = fmt.Errorf("SERVER_VERSION command failed with status %d", res.Type)
		}
		return 0, 0, 0, "", err
	}
	if len(res.Raw) < 4 {
		return 0, 0, 0, "", fmt.Errorf("SERVER_VERSION reply is too short")
	}
	version := int(binary.BigEndian.Uint32(res.Raw[:4]))
	major = version / 10000
	minor = version/100 - 100*major
	patch = version - 10000*major - 100*minor
	if len(res.Raw) > 4 {
		extra = string(res.Raw[4:])
	}
	return major, minor, patch, extra, nil
}

// vim: set tw=78 sw=4 sw=4 noexpandtab :
