package packet

const (
	_ Type = iota
	// Handshake represents a handshake: request(client) <====> handshake response(server)
	Handshake Type = 0x01

	// HandshakeAck represents a handshake ack from client to server
	HandshakeAck Type = 0x02

	// Heartbeat represents a heartbeat
	Heartbeat Type = 0x03

	// Data represents a common data packet
	Data Type = 0x04

	// Kick represents a kick off packet
	Kick Type = 0x05 // disconnect message from server
)

var types = [6]string{
	Handshake:    "Handshake",
	HandshakeAck: "HandshakeAck",
	Heartbeat:    "Heartbeat",
	Data:         "Data",
	Kick:         "Kick",
}

// Type represents the network packet's type such as: handshake and so on.
type Type byte

func (t Type) String() string {
	return types[t]
}

func invalidType(t Type) bool {
	return t < Handshake || t > Kick
}
