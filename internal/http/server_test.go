package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShuttingDownErr(t *testing.T) {
	s := &Server{Addr: ":18081"}
	err := s.Shutdown(context.Background())
	assert.NoError(t, err)
	_, err = s.Listen()
	assert.Error(t, err)
}

func TestShuttingDownOK(t *testing.T) {
	s := &Server{Addr: ":18081"}
	l, err := s.Listen()
	assert.NoError(t, err)
	_ = l.Close()
}
