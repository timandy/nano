package tworoom

import (
	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/examples/cluster/protocol"
)

type ChatRoomService struct {
	component.Base
	group *nano.Group
}

func newChatRoomService() *ChatRoomService {
	return &ChatRoomService{
		group: nano.NewGroup("all-users"),
	}
}

func (rs *ChatRoomService) JoinRoom(s *session.Session, msg *protocol.JoinRoomRequest) error {
	return rs.group.Add(s)
}

type SyncMessage struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (rs *ChatRoomService) SyncMessage(s *session.Session, msg *SyncMessage) error {
	// Sync message to all members in this room
	return rs.group.Broadcast("onMessage", msg)
}

func (rs *ChatRoomService) userDisconnected(s *session.Session) {
	if err := rs.group.Leave(s); err != nil {
		log.Error("Remove user from group failed", s.UID(), err)
		return
	}
	log.Info("User session disconnected", s.UID())
}
