package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShuttingDownErr(t *testing.T) {
	s := &Server{}
	err := s.Shutdown(context.Background())
	assert.Nil(t, err)
	_, err = s.Listen()
	assert.NotNil(t, err)
}

func TestShuttingDownOK(t *testing.T) {
	s := &Server{}
	l, err := s.Listen()
	assert.Nil(t, err)
	_ = l.Close()
}
