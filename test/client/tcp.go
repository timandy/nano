package client

import (
	"net"

	"github.com/lonng/nano/protocal/packet"
)

func NewTcpClient() *Client {
	return &Client{
		dialFunc: func(addr string) (net.Conn, error) {
			return net.Dial("tcp", addr)
		},
		die:               make(chan struct{}),
		codec:             packet.NewDecoder(),
		chSend:            make(chan []byte, 64),
		mid:               1,
		pushCallbacks:     map[string]Callback{},
		responseCallbacks: map[uint64]Callback{},
	}
}
