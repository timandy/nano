package cluster

const (
	_ int32 = iota
	statusStart
	statusHandshake
	statusWorking
	statusClosed
)
