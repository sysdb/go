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

/*
Package client provides a SysDB client implementation.

The Connect function connects to a SysDB server as the specified user:

	c, err := client.Connect("unix:/var/run/sysdbd.sock", "username")
	if err != nil {
		// handle error
	}
	defer c.Close()

The github.com/sysdb/go/proto package provides support for handling requests
and responses. Use the Send and Receive functions to communicate with the
server:

	m := &proto.Message{
		Type: proto.ConnectionQuery,
		Raw:  []byte{"LOOKUP hosts MATCHING attribute.architecture = 'amd64';"},
	}
	if err := c.Send(m); err != nil {
		// handle error
	}
	m, err := c.Receive()
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
	"fmt"
	"net"
	"strings"

	"github.com/sysdb/go/proto"
)

// A Conn is a connection to a SysDB server instance.
//
// Multiple goroutines may invoke methods on a Conn simultaneously but since
// the SysDB protocol requires a strict ordering of request and response
// messages, the communication with the server will usually happen
// sequentially.
type Conn struct {
	c                   net.Conn
	network, addr, user string
}

func (c *Conn) connect() (err error) {
	if c.c, err = net.Dial(c.network, c.addr); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	m := &proto.Message{
		Type: proto.ConnectionStartup,
		Raw:  []byte(c.user),
	}
	if err := c.Send(m); err != nil {
		return err
	}

	m, err = c.Receive()
	if err != nil {
		return err
	}
	if m.Type == proto.ConnectionError {
		return fmt.Errorf("failed to startup session: %s", string(m.Raw))
	}
	if m.Type != proto.ConnectionOK {
		return fmt.Errorf("failed to startup session: unsupported")
	}
	return nil
}

// Connect sets up a client connection to a SysDB server instance at the
// specified address using the specified user.
//
// The address may be a UNIX domain socket, either prefixed with 'unix:' or
// specifying an absolute file-system path.
func Connect(addr, user string) (*Conn, error) {
	network := "tcp"
	if strings.HasPrefix(addr, "unix:") {
		network = "unix"
		addr = addr[len("unix:"):]
	} else if addr[0] == '/' {
		network = "unix"
	}

	c := &Conn{network: network, addr: addr, user: user}
	if err := c.connect(); err != nil {
		return nil, err
	}
	return c, nil
}

// Close closes the client connection.
//
// Any blocked Send or Receive operations will be unblocked and return errors.
func (c *Conn) Close() {
	if c.c == nil {
		return
	}
	c.c.Close()
	c.c = nil
}

// Send sends the specified raw message to the server.
//
// Send operations block until the full message could be written to the
// underlying sockets. This ensures that server and client don't get out of
// sync.
func (c *Conn) Send(m *proto.Message) error {
	var err error
	if c.c != nil {
		err = proto.Write(c.c, m)
		if err == nil {
			return nil
		}
		c.Close()
	}

	// Try to reconnect.
	if e := c.connect(); e == nil {
		return proto.Write(c.c, m)
	} else if err == nil {
		err = e
	}
	return err
}

// Receive waits for a reply from the server and returns the raw message.
//
// Receive operations block until a full message could be read from the
// underlying socket. This ensures that server and client don't get out of
// sync.
func (c *Conn) Receive() (*proto.Message, error) {
	var err error
	if c.c != nil {
		var m *proto.Message
		m, err = proto.Read(c.c)
		if err == nil {
			return m, err
		}
		c.Close()
	}

	// Try to reconnect.
	if e := c.connect(); e == nil {
		return proto.Read(c.c)
	} else if err == nil {
		err = e
	}
	return nil, err
}

// vim: set tw=78 sw=4 sw=4 noexpandtab :
