package chat

import (
	"fmt"

	"github.com/lonng/nano"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/examples/cluster/protocol"
)

type RoomService struct {
	component.Base
	group *nano.Group
}

func newRoomService() *RoomService {
	return &RoomService{
		group: nano.NewGroup("all-users"),
	}
}

func (rs *RoomService) JoinRoom(s *session.Session, msg *protocol.JoinRoomRequest) error {
	if err := s.Bind(msg.MasterUid); err != nil {
		return err
	}

	broadcast := &protocol.NewUserBroadcast{
		Content: fmt.Sprintf("User user join: %v", msg.Nickname),
	}
	if err := rs.group.Broadcast("onNewUser", broadcast); err != nil {
		return err
	}
	return rs.group.Add(s)
}

type SyncMessage struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

func (rs *RoomService) SyncMessage(s *session.Session, msg *SyncMessage) error {
	// Send an RPC to master server to stats
	if err := s.RPC("TopicService.Stats", &protocol.MasterStats{Uid: s.UID()}); err != nil {
		return err
	}

	// Sync message to all members in this room
	return rs.group.Broadcast("onMessage", msg)
}

func (rs *RoomService) userDisconnected(s *session.Session) {
	if err := rs.group.Leave(s); err != nil {
		log.Error("Remove user from group failed", s.UID(), err)
		return
	}
	log.Info("User session disconnected", s.UID())
}
