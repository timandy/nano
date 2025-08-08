package protobuf

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

// ErrWrongValueType is the error used for marshal the value with protobuf encoding.
var ErrWrongValueType = errors.New("protobuf: convert on wrong type value")

// Serializer implements the serialize.Serializer interface
type Serializer struct{}

// NewSerializer returns a new Serializer.
func NewSerializer() *Serializer {
	return &Serializer{}
}

// Marshal returns the protobuf encoding of v.
func (s *Serializer) Marshal(v any) ([]byte, error) {
	pb, ok := v.(proto.Message)
	if !ok {
		return nil, ErrWrongValueType
	}
	return proto.Marshal(pb)
}

// Unmarshal parses the protobuf-encoded data and stores the result
// in the value pointed to by v.
func (s *Serializer) Unmarshal(data []byte, v any) error {
	pb, ok := v.(proto.Message)
	if !ok {
		return ErrWrongValueType
	}
	return proto.Unmarshal(data, pb)
}
