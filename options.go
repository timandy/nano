package nano

import (
	"net/http"
	"time"

	"github.com/lonng/nano/cluster"
	"github.com/lonng/nano/component"
	"github.com/lonng/nano/internal/env"
	"github.com/lonng/nano/internal/log"
	"github.com/lonng/nano/pipeline"
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/protocal/serialize"
	"github.com/lonng/nano/session"
	"github.com/lonng/nano/session/service"
	"google.golang.org/grpc"
)

type Option func(*cluster.Options)

//==== 基本

// WithDebugMode 启用调试
func WithDebugMode() Option {
	return func(opt *cluster.Options) {
		env.Debug = true
	}
}

// WithLogger 设置日志
func WithLogger(logger log.Logger) Option {
	return func(opt *cluster.Options) {
		log.SetLogger(logger)
	}
}

//==== 路由

// WithDisableScan 禁止自动扫描路由
func WithDisableScan() Option {
	return func(opt *cluster.Options) {
		opt.AutoScan = false
	}
}

//==== 握手

// WithCheckOrigin 设置跨域检查函数
func WithCheckOrigin(fn func(*http.Request) bool) Option {
	return func(opt *cluster.Options) {
		opt.CheckOrigin = fn
	}
}

// WithHandshakeValidator 设置握手校验函数
func WithHandshakeValidator(fn func(*session.Session, []byte) error) Option {
	return func(opt *cluster.Options) {
		opt.HandshakeValidator = fn
	}
}

//==== 工作

// WithPipeline 所有输入前置函数和输出前置函数
func WithPipeline(pipeline pipeline.Pipeline) Option {
	return func(opt *cluster.Options) {
		opt.Pipeline = pipeline
	}
}

// WithComponents 设置业务路由组件
func WithComponents(components *component.Components) Option {
	return func(opt *cluster.Options) {
		opt.Components = components
	}
}

// WithSerializer 设置序列化器
func WithSerializer(serializer serialize.Serializer) Option {
	return func(opt *cluster.Options) {
		env.Serializer = serializer
	}
}

// WithTimerPrecision 定时器精度, 不能小于 1 毫秒
func WithTimerPrecision(precision time.Duration) Option {
	if precision < time.Millisecond {
		panic("time precision can not less than a Millisecond")
	}
	return func(opt *cluster.Options) {
		env.TimerPrecision = precision
	}
}

// WithRouteDict 设置路由字典
func WithRouteDict(dict map[string]uint16) Option {
	return func(opt *cluster.Options) {
		message.SetRouteDict(dict)
	}
}

//==== 集群

// WithNodeType 设置节点类型
func WithNodeType(nodeType cluster.NodeType) Option {
	return func(opt *cluster.Options) {
		opt.NodeType = nodeType
	}
}

// WithAdvertiseAddr 集群模式, 子节点, 要配置这个值, 以便向 Master 注册子自身的 ServiceAddr
func WithAdvertiseAddr(addr string) Option {
	return func(opt *cluster.Options) {
		opt.AdvertiseAddr = addr
	}
}

// WithServiceAddr 集群模式, 主子节点都要配置该值, 以便启用 grpc 监听
func WithServiceAddr(addr string) Option {
	return func(opt *cluster.Options) {
		opt.ServiceAddr = addr
	}
}

// WithRetryInterval 子节点向 Master 注册失败后, 重试间隔时间
func WithRetryInterval(interval time.Duration) Option {
	return func(opt *cluster.Options) {
		env.RetryInterval = interval
	}
}

// WithHeartbeatInterval 子节点向 Master 定时心跳请求间隔
func WithHeartbeatInterval(d time.Duration) Option {
	return func(opt *cluster.Options) {
		env.HeartbeatInterval = d
	}
}

// WithGrpcOptions 自建店的 Grpc 客户端连接选项
func WithGrpcOptions(opts ...grpc.DialOption) Option {
	return func(opt *cluster.Options) {
		env.GrpcOptions = append(env.GrpcOptions, opts...)
	}
}

// WithRemoteServiceRoute 自定义节点路由规则
func WithRemoteServiceRoute(route cluster.CustomerRemoteServiceRoute) Option {
	return func(opt *cluster.Options) {
		opt.RemoteServiceRoute = route
	}
}

// WithUnregisterCallback 主节点可以配置回调
func WithUnregisterCallback(fn cluster.UnregisterCallback) Option {
	return func(opt *cluster.Options) {
		opt.UnregisterCallback = fn
	}
}

// WithLabel 设置节点标签
func WithLabel(label string) Option {
	return func(opt *cluster.Options) {
		opt.Label = label
	}
}

// WithNodeId 使用 snowFlake 算法生成 sessionId 的时候, 使用此设置作为 workerId
func WithNodeId(nodeId uint64) Option {
	return func(opt *cluster.Options) {
		service.ResetNodeId(nodeId)
	}
}
