package tcp

import (
	"net"
	"sync"

	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/protocal/packet"
	"google.golang.org/protobuf/proto"
)

var (
	hsd []byte // handshake data
	had []byte // handshake ack data
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

type (
	// Callback represents the callback type which will be called when the correspond events is occurred.
	Callback func(data any)

	// TcpConnector is a tiny Nano client
	TcpConnector struct {
		conn   net.Conn        // low-level connection
		codec  *packet.Decoder // decoder
		die    chan struct{}   // connector close channel
		chSend chan []byte     // send queue
		mid    uint64          // message id

		// events handler
		muEvents sync.RWMutex
		events   map[string]Callback

		// response handler
		muResponses sync.RWMutex
		responses   map[uint64]Callback

		connectedCallback func() // connected callback
	}
)

// NewTcpConnector create a new TcpConnector
func NewTcpConnector() *TcpConnector {
	return &TcpConnector{
		die:       make(chan struct{}),
		codec:     packet.NewDecoder(),
		chSend:    make(chan []byte, 64),
		mid:       1,
		events:    map[string]Callback{},
		responses: map[uint64]Callback{},
	}
}

// Start connect to the server and send/recv between the c/s
func (c *TcpConnector) Start(addr string) error {
	conn, err := net.Dial("tcp", addr)
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
func (c *TcpConnector) OnConnected(callback func()) {
	c.connectedCallback = callback
}

// Request send a request to server and register a callbck for the response
func (c *TcpConnector) Request(route string, v proto.Message, callback Callback) error {
	data, err := serialize(v)
	if err != nil {
		return err
	}

	msg := &message.Message{
		Type:  message.Request,
		Route: route,
		ID:    c.mid,
		Data:  data,
	}

	c.setResponseHandler(c.mid, callback)
	if err = c.sendMessage(msg); err != nil {
		c.setResponseHandler(c.mid, nil)
		return err
	}

	return nil
}

// Notify send a notification to server
func (c *TcpConnector) Notify(route string, v proto.Message) error {
	data, err := serialize(v)
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

// On add the callback for the event
func (c *TcpConnector) On(event string, callback Callback) {
	c.muEvents.Lock()
	defer c.muEvents.Unlock()

	c.events[event] = callback
}

// Close the connection, and shutdown the benchmark
func (c *TcpConnector) Close() {
	_ = c.conn.Close()
	close(c.die)
}

func (c *TcpConnector) eventHandler(event string) (Callback, bool) {
	c.muEvents.RLock()
	defer c.muEvents.RUnlock()

	cb, ok := c.events[event]
	return cb, ok
}

func (c *TcpConnector) responseHandler(mid uint64) (Callback, bool) {
	c.muResponses.RLock()
	defer c.muResponses.RUnlock()

	cb, ok := c.responses[mid]
	return cb, ok
}

func (c *TcpConnector) setResponseHandler(mid uint64, cb Callback) {
	c.muResponses.Lock()
	defer c.muResponses.Unlock()

	if cb == nil {
		delete(c.responses, mid)
	} else {
		c.responses[mid] = cb
	}
}

func (c *TcpConnector) sendMessage(msg *message.Message) error {
	data, err := msg.Encode()
	if err != nil {
		return err
	}

	//log.Printf("%v",msg)

	payload, err := packet.Encode(packet.Data, data)
	if err != nil {
		return err
	}

	c.mid++
	c.send(payload)

	return nil
}

func (c *TcpConnector) write() {
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

func (c *TcpConnector) send(data []byte) {
	c.chSend <- data
}

func (c *TcpConnector) read() {
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

func (c *TcpConnector) processPacket(p *packet.Packet) {
	switch p.Type {
	case packet.Handshake:
		c.send(had)
		c.connectedCallback()
	case packet.Data:
		msg, err := message.Decode(p.Data)
		if err != nil {
			log.Error(err.Error())
			return
		}
		c.processMessage(msg)

	case packet.Kick:
		c.Close()
	}
}

func (c *TcpConnector) processMessage(msg *message.Message) {
	switch msg.Type {
	case message.Push:
		cb, ok := c.eventHandler(msg.Route)
		if !ok {
			log.Info("event handler not found", msg.Route)
			return
		}

		cb(msg.Data)

	case message.Response:
		cb, ok := c.responseHandler(msg.ID)
		if !ok {
			log.Info("response handler not found", msg.ID)
			return
		}

		cb(msg.Data)
		c.setResponseHandler(msg.ID, nil)
	}
}

func serialize(v proto.Message) ([]byte, error) {
	data, err := proto.Marshal(v)
	if err != nil {
		return nil, err
	}
	return data, nil
}
