package npi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerTrees(t *testing.T) {
	trees := HandlerTrees{}
	trees.Append("test", NewHandlerNodeWithName("hi", nil))
	assert.Equal(t, "hi", trees["test"].Name())
	//
	trees.Append("test", NewHandlerNodeWithName("hello", nil))
	assert.Equal(t, "hello", trees["test"].Name())
}
