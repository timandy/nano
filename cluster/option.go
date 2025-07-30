package cluster

import (
	"net/http"

	"github.com/lonng/nano/component"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/session"
)

// Options 引擎选项
type Options struct {
	//注册路由
	AutoScan bool //自动扫描

	//握手
	CheckOrigin        func(*http.Request) bool             //跨域检测, WebSocket 升级阶段
	HandshakeValidator func(*session.Session, []byte) error //握手阶段回调

	//工作
	Pipeline   pipeline.Pipeline     //所有输入前置函数和输出前置函数
	Components *component.Components //业务组件, 类似 Controller

	//集群
	NodeType           NodeType                   //节点类型
	AdvertiseAddr      string                     //RPC 服务对外地址, 一般是 IP:Port; 子节点, 要配置这个值, 以便向 Master 注册子自身的 ServiceAddr
	ServiceAddr        string                     //RPC 服务监听地址, 一般是 IP:Port; 主子节点都要配置这个值
	RemoteServiceRoute CustomerRemoteServiceRoute //自定义节点路由规则
	UnregisterCallback UnregisterCallback         //主节点可以配置回调
	Label              string                     // 节点标签, 用于标识节点, 例如 "master", "slave-1", "slave-2" 等
}

// DefaultOptions 默认选项
func DefaultOptions() *Options {
	return &Options{
		//注册路由
		AutoScan: true,
		//握手
		CheckOrigin:        func(_ *http.Request) bool { return true },
		HandshakeValidator: nil,
		//工作
		Components: &component.Components{},
		//集群
		NodeType: NodeTypeNone, //默认不是主节点
	}
}
