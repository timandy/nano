package logic

import (
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/examples/demo/tadpole/logic/protocol"
)

// Manager component
type Manager struct {
	component.Base
}

// NewManager returns  a new manager instance
func NewManager() *Manager {
	return &Manager{}
}

// Login handler was used to guest login
func (m *Manager) Login(s *session.Session, msg *protocol.JoyLoginRequest) error {
	log.Info(msg)
	id := s.ID()
	s.Bind(id)
	return s.Response(protocol.LoginResponse{
		Status: protocol.LoginStatusSucc,
		ID:     id,
	})
}
