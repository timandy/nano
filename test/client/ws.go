package client

import (
	"io"
	"net"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lonng/nano/protocal/packet"
)

func NewWsClient() *Client {
	return &Client{
		dialFunc: func(addr string) (net.Conn, error) {
			conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
			if err != nil {
				return nil, err
			}
			return newWSConn(conn)
		},
		die:               make(chan struct{}),
		codec:             packet.NewDecoder(),
		chSend:            make(chan []byte, 64),
		mid:               1,
		pushCallbacks:     map[string]Callback{},
		responseCallbacks: map[uint64]Callback{},
	}
}

//=====

var _ net.Conn = (*wsConn)(nil)

type wsConn struct {
	conn   *websocket.Conn
	reader io.Reader
}

func newWSConn(conn *websocket.Conn) (net.Conn, error) {
	return &wsConn{conn: conn}, nil
}

func (c *wsConn) Read(b []byte) (int, error) {
	if c.reader == nil {
		_, r, err := c.conn.NextReader()
		if err != nil {
			return 0, err
		}
		c.reader = r
	}

	n, err := c.reader.Read(b)
	if err != nil && err != io.EOF {
		return n, err
	} else if err == io.EOF {
		_, r, err := c.conn.NextReader()
		if err != nil {
			return 0, err
		}
		c.reader = r
	}

	return n, nil
}

func (c *wsConn) Write(b []byte) (int, error) {
	err := c.conn.WriteMessage(websocket.BinaryMessage, b)
	if err != nil {
		return 0, err
	}

	return len(b), nil
}

func (c *wsConn) Close() error {
	return c.conn.Close()
}

func (c *wsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *wsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *wsConn) SetDeadline(t time.Time) error {
	if err := c.conn.SetReadDeadline(t); err != nil {
		return err
	}
	return c.conn.SetWriteDeadline(t)
}

func (c *wsConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *wsConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}
