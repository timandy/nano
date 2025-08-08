package packet

import (
	"fmt"
)

// Packet represents a network packet.
type Packet struct {
	Type   Type
	Length int
	Data   []byte
}

// New create a Packet instance.
func New() *Packet {
	return &Packet{}
}

// String represents the Packet's in text mode.
func (p *Packet) String() string {
	return fmt.Sprintf("Type: %d, Length: %d, Data: %s", p.Type, p.Length, string(p.Data))
}
