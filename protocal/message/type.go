package message

// Message types
const (
	Request  Type = 0x00
	Notify   Type = 0x01
	Response Type = 0x02
	Push     Type = 0x03
)

var types = [4]string{
	Request:  "Request",
	Notify:   "Notify",
	Response: "Response",
	Push:     "Push",
}

// Type represents the type of message, which could be Request/Notify/Response/Push
type Type byte

func (t Type) String() string {
	return types[t]
}

func routable(t Type) bool {
	return t == Request || t == Notify || t == Push
}

func invalidType(t Type) bool {
	return t < Request || t > Push
}
