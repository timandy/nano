package message

import (
	"fmt"
	"strings"
)

// Message represents an unmarshalled message or a message which to be marshaled
type Message struct {
	Type       Type   // message type
	ID         uint64 // unique id, zero while notify mode
	Route      string // route for locating service
	Data       []byte // payload
	compressed bool   // is message compressed
}

// New returns a new message instance
func New() *Message {
	return &Message{}
}

// ResponseID 获取响应 ID
func (m *Message) ResponseID() (uint64, error) {
	switch m.Type {
	case Request:
		return m.ID, nil
	case Notify:
		return 0, nil
	default:
		return 0, ErrWrongMessageType
	}
}

// Service 获取服务名
func (m *Message) Service() string {
	route := m.Route
	if index := strings.LastIndex(route, "."); index != -1 {
		return route[:index]
	}
	return ""
}

// String, implementation of fmt.Stringer interface
func (m *Message) String() string {
	return fmt.Sprintf("%s %s (%dbytes)", types[m.Type], m.Route, len(m.Data))
}

// Encode marshals message to binary format.
func (m *Message) Encode() ([]byte, error) {
	return Encode(m)
}
