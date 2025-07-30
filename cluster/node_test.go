package cluster_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/benchmark/testdata"
	"github.com/lonng/nano/test/benchmark/ws"
	. "github.com/pingcap/check"
)

type nodeSuite struct{}

var _ = Suite(&nodeSuite{})

type (
	MasterComponent struct{ component.Base }
	GateComponent   struct{ component.Base }
	GameComponent   struct{ component.Base }
)

func (c *MasterComponent) Test(session *session.Session, _ []byte) error {
	return session.Push("test", &testdata.Pong{Content: "master server pong"})
}

func (c *GateComponent) Test(session *session.Session, ping testdata.Ping) error {
	return session.Push("test", &testdata.Pong{Content: "gate server pong"})
}

func (c *GateComponent) Test2(session *session.Session, ping *testdata.Ping) error {
	return session.Response(&testdata.Pong{Content: "gate server pong2"})
}

func (c *GameComponent) Test(session *session.Session, _ []byte) error {
	return session.Push("test", &testdata.Pong{Content: "game server pong"})
}

func (c *GameComponent) Test2(session *session.Session, ping *testdata.Ping) error {
	return session.Response(&testdata.Pong{Content: "game server pong2"})
}

func TestNode(t *testing.T) {
	TestingT(t)
}

func (s *nodeSuite) TestNodeStartup(c *C) {
	go scheduler.Sched()
	defer scheduler.Close()

	//注册中心
	masterComps := &component.Components{}
	masterComps.Register(&MasterComponent{})
	masterOpts := cluster.DefaultOptions()
	masterOpts.NodeType = cluster.NodeTypeMaster | cluster.NodeTypeWorker
	masterOpts.Components = masterComps
	masterOpts.ServiceAddr = "127.0.0.1:4450"
	masterNode := cluster.NewNode(nil, masterOpts)
	masterHandler := masterNode.Handler()
	c.Assert(masterHandler.LocalService(), DeepEquals, []string{"MasterComponent"})
	//网关
	memberGate := &component.Components{}
	memberGate.Register(&GateComponent{})
	gateOpts := cluster.DefaultOptions()
	gateOpts.NodeType = cluster.NodeTypeGate
	gateOpts.AdvertiseAddr = "127.0.0.1:4450"
	gateOpts.ServiceAddr = "127.0.0.1:14451"
	gateOpts.Components = memberGate
	memberNode1 := cluster.NewNode(nil, gateOpts)
	member1Handler := memberNode1.Handler()
	c.Assert(masterHandler.LocalService(), DeepEquals, []string{"MasterComponent"})
	c.Assert(masterHandler.RemoteService(), DeepEquals, []string{"GateComponent"})
	c.Assert(member1Handler.LocalService(), DeepEquals, []string{"GateComponent"})
	c.Assert(member1Handler.RemoteService(), DeepEquals, []string{"MasterComponent"})
	//网关监听
	mux1 := http.NewServeMux()
	mux1.Handle("/ws", memberNode1)
	go http.ListenAndServe(":14452", mux1)
	//游戏
	memberWorker := &component.Components{}
	memberWorker.Register(&GameComponent{})
	workerOpts := cluster.DefaultOptions()
	workerOpts.NodeType = cluster.NodeTypeWorker
	workerOpts.AdvertiseAddr = "127.0.0.1:4450"
	workerOpts.ServiceAddr = "127.0.0.1:24451"
	workerOpts.Components = memberWorker
	memberNode2 := cluster.NewNode(nil, workerOpts)
	member2Handler := memberNode2.Handler()
	c.Assert(masterHandler.LocalService(), DeepEquals, []string{"MasterComponent"})
	c.Assert(masterHandler.RemoteService(), DeepEquals, []string{"GameComponent", "GateComponent"})
	c.Assert(member1Handler.LocalService(), DeepEquals, []string{"GateComponent"})
	c.Assert(member1Handler.RemoteService(), DeepEquals, []string{"GameComponent", "MasterComponent"})
	c.Assert(member2Handler.LocalService(), DeepEquals, []string{"GameComponent"})
	c.Assert(member2Handler.RemoteService(), DeepEquals, []string{"GateComponent", "MasterComponent"})

	//客户端
	connector := ws.NewConnector()

	chWait := make(chan struct{})
	connector.OnConnected(func() {
		chWait <- struct{}{}
	})

	// Connect to gate server
	if err := connector.Start("127.0.0.1:14452"); err != nil {
		c.Assert(err, IsNil)
	}
	<-chWait
	onResult := make(chan string)
	connector.On("test", func(data any) {
		onResult <- string(data.([]byte))
	})
	err := connector.Notify("GateComponent.Test", &testdata.Ping{Content: "ping"})
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(<-onResult, "gate server pong"), IsTrue)

	err = connector.Notify("GameComponent.Test", &testdata.Ping{Content: "ping"})
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(<-onResult, "game server pong"), IsTrue)

	err = connector.Request("GateComponent.Test2", &testdata.Ping{Content: "ping"}, func(data any) {
		onResult <- string(data.([]byte))
	})
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(<-onResult, "gate server pong2"), IsTrue)

	err = connector.Request("GameComponent.Test2", &testdata.Ping{Content: "ping"}, func(data any) {
		onResult <- string(data.([]byte))
	})
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(<-onResult, "game server pong2"), IsTrue)

	err = connector.Notify("MasterComponent.Test", &testdata.Ping{Content: "ping"})
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(<-onResult, "master server pong"), IsTrue)
}
