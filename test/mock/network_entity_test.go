package mock_test

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/lonng/nano/test/mock"
	"github.com/stretchr/testify/assert"
)

func TestNetworkEntity(t *testing.T) {
	entity := NewNetworkEntity()

	assert.Equal(t, entity.RemoteAddr().String(), "mock-addr")

	assert.Nil(t, entity.findPush("t.tt"))
	assert.Nil(t, entity.Push("t.tt", "test"))
	assert.Equal(t, entity.findPush("t.tt").(string), "test")

	assert.Nil(t, entity.findResponse())
	assert.Equal(t, entity.LastMid(), uint64(1))
	assert.Nil(t, entity.Response("hello"))
	assert.Equal(t, entity.findResponse().(string), "hello")

	assert.Nil(t, entity.findResponseMID(1))
	assert.Nil(t, entity.ResponseMid(1, "test"))
	assert.Equal(t, entity.findResponseMID(1).(string), "test")

	assert.Nil(t, entity.findKick())
	assert.Nil(t, entity.Kick("you are kicked"))
	assert.Equal(t, "you are kicked", entity.findKick().(string))
	assert.NotNil(t, entity.Kick("you are kicked again"))

	assert.Nil(t, entity.Close())
}

type message struct {
	route string
	data  any
}

// NetworkEntity represents an network entity which can be used to construct the
// session object.
type NetworkEntity struct {
	rpc         []message
	push        []message
	response    []any
	responseMid map[uint64]any
	kick        any
}

// NewNetworkEntity returns an mock network entity
func NewNetworkEntity() *NetworkEntity {
	return &NetworkEntity{
		responseMid: map[uint64]any{},
	}
}

// RemoteAddr implements the session.NetworkEntity interface
func (n *NetworkEntity) RemoteAddr() net.Addr {
	return mock.NetAddr{}
}

// LastMid implements the session.NetworkEntity interface
func (n *NetworkEntity) LastMid() uint64 {
	return 1
}

// RPC implements the session.NetworkEntity interface
func (n *NetworkEntity) RPC(route string, v any) error {
	n.rpc = append(n.rpc, message{route: route, data: v})
	return nil
}

// Push implements the session.NetworkEntity interface
func (n *NetworkEntity) Push(route string, v any) error {
	n.push = append(n.push, message{route: route, data: v})
	return nil
}

// Response implements the session.NetworkEntity interface
func (n *NetworkEntity) Response(v any) error {
	n.response = append(n.response, v)
	return nil
}

// ResponseMid implements the session.NetworkEntity interface
func (n *NetworkEntity) ResponseMid(mid uint64, v any) error {
	_, found := n.responseMid[mid]
	if found {
		return fmt.Errorf("duplicated message id: %v", mid)
	}
	n.responseMid[mid] = v
	return nil
}

// Kick implements the session.NetworkEntity interface
func (n *NetworkEntity) Kick(v any) error {
	if n.kick != nil {
		return errors.New("already kicked")
	}
	n.kick = v
	return nil
}

// Close implements the session.NetworkEntity interface
func (n *NetworkEntity) Close() error {
	return nil
}

// findPush returns the response by route
func (n *NetworkEntity) findPush(route string) any {
	for i := range n.push {
		if n.push[i].route == route {
			return n.push[i].data
		}
	}
	return nil
}

// findResponse returns the last response message
func (n *NetworkEntity) findResponse() any {
	if len(n.response) < 1 {
		return nil
	}
	return n.response[len(n.response)-1]
}

// findResponseMID returns the response by message id
func (n *NetworkEntity) findResponseMID(mid uint64) any {
	return n.responseMid[mid]
}

// findKick returns the kick message
func (n *NetworkEntity) findKick() any {
	return n.kick
}
