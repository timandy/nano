package ws

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
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

//goland:noinspection GoNameStartsWithPackageName
type (
	// Callback represents the callback type which will be called when the correspond events is occurred.
	Callback func(data any)

	// WsConnector is a tiny Nano client
	WsConnector struct {
		conn   *websocket.Conn // low-level connection
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

// NewWsConnector create a new WsConnector
func NewWsConnector() *WsConnector {
	return &WsConnector{
		die:       make(chan struct{}),
		codec:     packet.NewDecoder(),
		chSend:    make(chan []byte, 64),
		mid:       1,
		events:    map[string]Callback{},
		responses: map[uint64]Callback{},
	}
}

// Start connect to the server and send/recv between the c/s
func (c *WsConnector) Start(addr string) error {
	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("ws://%v/ws", addr), nil)
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
func (c *WsConnector) OnConnected(callback func()) {
	c.connectedCallback = callback
}

// Request send a request to server and register a callbck for the response
func (c *WsConnector) Request(route string, v proto.Message, callback Callback) error {
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
func (c *WsConnector) Notify(route string, v proto.Message) error {
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
func (c *WsConnector) On(event string, callback Callback) {
	c.muEvents.Lock()
	defer c.muEvents.Unlock()

	c.events[event] = callback
}

// Close the connection, and shutdown the benchmark
func (c *WsConnector) Close() {
	_ = c.conn.Close()
	close(c.die)
}

func (c *WsConnector) eventHandler(event string) (Callback, bool) {
	c.muEvents.RLock()
	defer c.muEvents.RUnlock()

	cb, ok := c.events[event]
	return cb, ok
}

func (c *WsConnector) responseHandler(mid uint64) (Callback, bool) {
	c.muResponses.RLock()
	defer c.muResponses.RUnlock()

	cb, ok := c.responses[mid]
	return cb, ok
}

func (c *WsConnector) setResponseHandler(mid uint64, cb Callback) {
	c.muResponses.Lock()
	defer c.muResponses.Unlock()

	if cb == nil {
		delete(c.responses, mid)
	} else {
		c.responses[mid] = cb
	}
}

func (c *WsConnector) sendMessage(msg *message.Message) error {
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

func (c *WsConnector) write() {
	defer close(c.chSend)

	for {
		select {
		case data := <-c.chSend:
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Error(err.Error())
				c.Close()
			}

		case <-c.die:
			return
		}
	}
}

func (c *WsConnector) send(data []byte) {
	c.chSend <- data
}

func (c *WsConnector) read() {
	for {
		_, buf, err := c.conn.ReadMessage()
		if err != nil {
			log.Error(err.Error())
			c.Close()
			return
		}

		packets, err := c.codec.Decode(buf)
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

func (c *WsConnector) processPacket(p *packet.Packet) {
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

func (c *WsConnector) processMessage(msg *message.Message) {
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
