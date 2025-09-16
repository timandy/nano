package client

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/protocal/packet"
	"google.golang.org/protobuf/proto"
)

var (
	hsd []byte // handshake data
	had []byte // handshake ack data
)

var (
	ErrClientClosed = errors.New("client is closed")
)

func init() {
	var err error
	hsd, err = packet.Encode(packet.Handshake, nil)
	if err != nil {
		panic(err)
	}

	had, err = packet.Encode(packet.HandshakeAck, nil)
	if err != nil {
		panic(err)
	}
}

// Callback represents the callback type which will be called when the correspond events is occurred.
type Callback func(data []byte)

// Client is a tiny Nano client
type Client struct {
	dialFunc  func(addr string) (net.Conn, error) // dial function
	conn      net.Conn                            // low-level connection
	codec     *packet.Decoder                     // decoder
	die       chan struct{}                       // connector close channel
	chSend    chan []byte                         // send queue
	mid       uint64                              // message id
	closeOnce sync.Once                           // close once
	closed    atomic.Bool                         // is closed or not

	// connected callback
	muConnectedCallback sync.RWMutex
	connectedCallback   func()

	// disconnected callback
	muDisconnectedCallback sync.RWMutex
	disconnectedCallback   func()

	// push callbacks
	muPushCallbacks sync.RWMutex
	pushCallbacks   map[string]Callback

	// response callbacks
	muResponseCallbacks sync.RWMutex
	responseCallbacks   map[uint64]Callback

	// kick callback
	muKickCallback sync.RWMutex
	kickCallback   Callback
}

// Start connect to the server and send/recv between the c/s
func (c *Client) Start(addr string) error {
	conn, err := c.dialFunc(addr)
	if err != nil {
		return err
	}

	c.conn = conn

	go c.write()

	// send handshake packet
	c.send(hsd)

	// read and process network message
	go c.read()

	return nil
}

// OnConnected set the callback which will be called when the client connected to the server
func (c *Client) OnConnected(callback func()) {
	c.setConnectedCallback(callback)
}

// OnDisconnected set the callback which will be called when the client disconnected from the server
func (c *Client) OnDisconnected(callback func()) {
	c.setDisconnectedCallback(callback)
}

// On add the callback for the event
func (c *Client) On(event string, callback Callback) {
	c.setPushCallback(event, callback)
}

// OnKick set the callback which will be called when the client is kicked by the server
func (c *Client) OnKick(callback Callback) {
	c.setKickCallback(callback)
}

// Request send a request to server and register a callbck for the response
func (c *Client) Request(route string, v proto.Message, callback Callback) error {
	data, err := c.serialize(v)
	if err != nil {
		return err
	}

	msg := &message.Message{
		Type:  message.Request,
		Route: route,
		ID:    c.mid,
		Data:  data,
	}

	c.setResponseCallback(c.mid, callback)
	if err = c.sendMessage(msg); err != nil {
		c.setResponseCallback(c.mid, nil)
		return err
	}

	return nil
}

// Notify send a notification to server
func (c *Client) Notify(route string, v proto.Message) error {
	data, err := c.serialize(v)
	if err != nil {
		return err
	}

	msg := &message.Message{
		Type:  message.Notify,
		Route: route,
		Data:  data,
	}
	return c.sendMessage(msg)
}

// Close the connection, and shutdown the benchmark
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		_ = c.conn.Close()
		close(c.die)
	})
}

//===== write loop =====

func (c *Client) write() {
	defer close(c.chSend)

	for {
		select {
		case data := <-c.chSend:
			if _, err := c.conn.Write(data); err != nil {
				log.Error(err.Error())
				c.Close()
			}

		case <-c.die:
			return
		}
	}
}

func (c *Client) sendMessage(msg *message.Message) error {
	if c.closed.Load() {
		return ErrClientClosed
	}

	data, err := msg.Encode()
	if err != nil {
		return err
	}

	payload, err := packet.Encode(packet.Data, data)
	if err != nil {
		return err
	}

	c.mid++
	c.send(payload)

	return nil
}

func (c *Client) send(data []byte) {
	c.chSend <- data
}

//===== read loop =====

func (c *Client) read() {
	defer c.fireDisconnectedCallback()

	buf := make([]byte, 2048)
	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			log.Error(err.Error())
			c.Close()
			return
		}

		packets, err := c.codec.Decode(buf[:n])
		if err != nil {
			log.Error(err.Error())
			c.Close()
			return
		}

		for i := range packets {
			p := packets[i]
			c.processPacket(p)
		}
	}
}

func (c *Client) processPacket(p *packet.Packet) {
	switch p.Type {
	case packet.Handshake:
		c.send(had)
		c.fireConnectedCallback()

	case packet.Data:
		msg, err := message.Decode(p.Data)
		if err != nil {
			log.Error(err.Error())
			return
		}
		c.processMessage(msg)

	case packet.Kick:
		msg, err := message.Decode(p.Data)
		if err != nil {
			log.Error(err.Error())
			return
		}
		c.processKick(msg)
		c.Close()

	default:
		// do nothing
	}
}

func (c *Client) processMessage(msg *message.Message) {
	switch msg.Type {
	case message.Push:
		c.firePushCallback(msg.Route, msg.Data)

	case message.Response:
		c.fireResponseCallback(msg.ID, msg.Data)
		c.setResponseCallback(msg.ID, nil)
	}
}

func (c *Client) processKick(msg *message.Message) {
	switch msg.Type {
	case message.Push:
		c.fireKickCallback(msg.Data)
	}
}

//===== serialize =====

func (c *Client) serialize(v proto.Message) ([]byte, error) {
	data, err := proto.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}
