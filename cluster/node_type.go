package cluster

const (
	NodeTypeNone   NodeType = 0
	NodeTypeMaster NodeType = 1
	NodeTypeGate   NodeType = 2
	NodeTypeWorker NodeType = 4
)

// NodeType 节点类型, 一个节点可以同时是 Master、Gate 和 Worker
type NodeType int

// IsSingle 非集群模式
func (r NodeType) IsSingle() bool {
	return r&NodeTypeMaster == 0 && r&NodeTypeGate == 0 && r&NodeTypeWorker == 0
}

// IsMaster 是否是 Master 节点, 注册中心
func (r NodeType) IsMaster() bool {
	return r&NodeTypeMaster != 0
}

// IsGate 是否是 Gate 节点, 网关
func (r NodeType) IsGate() bool {
	return r&NodeTypeGate != 0
}

// IsWorker 是否是 Worker 节点, 工作节点
func (r NodeType) IsWorker() bool {
	return r&NodeTypeWorker != 0
}

// IsMember 是否是成员节点, 包括 Gate 和 Worker
func (r NodeType) IsMember() bool {
	return r.IsGate() || r.IsWorker()
}
