package gate

import (
	"errors"

	"github.com/lonng/nano/component"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/examples/cluster/protocol"
)

type BindService struct {
	component.Base
	nextGateUid int64
}

func newBindService() *BindService {
	return &BindService{}
}

type (
	LoginRequest struct {
		Nickname string `json:"nickname"`
	}
	LoginResponse struct {
		Code int `json:"code"`
	}
)

func (bs *BindService) Login(s *session.Session, msg *LoginRequest) error {
	bs.nextGateUid++
	uid := bs.nextGateUid
	request := &protocol.NewUserRequest{
		Nickname: msg.Nickname,
		GateUid:  uid,
	}
	if err := s.RPC("TopicService.NewUser", request); err != nil {
		return err
	}
	return s.Response(&LoginResponse{})
}

func (bs *BindService) BindChatServer(s *session.Session, msg []byte) error {
	return errors.New("not implement")
}
