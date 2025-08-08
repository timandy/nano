package cluster_test

import (
	"net/http"
	"testing"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/scheduler"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/test/benchmark/testdata"
	"github.com/lonng/nano/test/benchmark/ws"
	"github.com/stretchr/testify/assert"
)

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

func TestNodeStartup(t *testing.T) {
	scheduler.Start()
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
	assert.Equal(t, masterHandler.LocalService(), []string{"MasterComponent"})
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
	assert.Equal(t, masterHandler.LocalService(), []string{"MasterComponent"})
	assert.Equal(t, masterHandler.RemoteService(), []string{"GateComponent"})
	assert.Equal(t, member1Handler.LocalService(), []string{"GateComponent"})
	assert.Equal(t, member1Handler.RemoteService(), []string{"MasterComponent"})
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
	assert.Equal(t, masterHandler.LocalService(), []string{"MasterComponent"})
	assert.Equal(t, masterHandler.RemoteService(), []string{"GameComponent", "GateComponent"})
	assert.Equal(t, member1Handler.LocalService(), []string{"GateComponent"})
	assert.Equal(t, member1Handler.RemoteService(), []string{"GameComponent", "MasterComponent"})
	assert.Equal(t, member2Handler.LocalService(), []string{"GameComponent"})
	assert.Equal(t, member2Handler.RemoteService(), []string{"GateComponent", "MasterComponent"})

	//客户端
	connector := ws.NewWsConnector()

	chWait := make(chan struct{})
	connector.OnConnected(func() {
		chWait <- struct{}{}
	})

	// Connect to gate server
	err := connector.Start("127.0.0.1:14452")
	assert.NoError(t, err)
	<-chWait
	onResult := make(chan string)
	connector.On("test", func(data any) {
		onResult <- string(data.([]byte))
	})
	err = connector.Notify("GateComponent.Test", &testdata.Ping{Content: "ping"})
	assert.NoError(t, err)
	assert.Contains(t, <-onResult, "gate server pong")

	err = connector.Notify("GameComponent.Test", &testdata.Ping{Content: "ping"})
	assert.NoError(t, err)
	assert.Contains(t, <-onResult, "game server pong")

	err = connector.Request("GateComponent.Test2", &testdata.Ping{Content: "ping"}, func(data any) {
		onResult <- string(data.([]byte))
	})
	assert.NoError(t, err)
	assert.Contains(t, <-onResult, "gate server pong2")

	err = connector.Request("GameComponent.Test2", &testdata.Ping{Content: "ping"}, func(data any) {
		onResult <- string(data.([]byte))
	})
	assert.NoError(t, err)
	assert.Contains(t, <-onResult, "game server pong2")

	err = connector.Notify("MasterComponent.Test", &testdata.Ping{Content: "ping"})
	assert.NoError(t, err)
	assert.Contains(t, <-onResult, "master server pong")
}
