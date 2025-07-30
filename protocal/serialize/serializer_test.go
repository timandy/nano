package serialize_test

import (
	"testing"

	"github.com/lonng/nano"
	"github.com/lonng/nano/protocal/serialize"
	"github.com/stretchr/testify/assert"
)

func TestSerialize(t *testing.T) {
	_ = nano.New()
	assert.NotNil(t, serialize.Marshal)
	assert.NotNil(t, serialize.Unmarshal)
}
