package message

import (
	"fmt"
)

// Message represents a unmarshalled message or a message which to be marshaled
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

// String, implementation of fmt.Stringer interface
func (m *Message) String() string {
	return fmt.Sprintf("%s %s (%dbytes)", types[m.Type], m.Route, len(m.Data))
}

// Encode marshals message to binary format.
func (m *Message) Encode() ([]byte, error) {
	return Encode(m)
}
