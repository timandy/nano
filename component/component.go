package component

var reservedMethods = []string{"Name", "Executor", "Register", "Init", "AfterInit", "BeforeShutdown", "Shutdown"}

// Component is the interface that represent a component.
type Component interface {
	Init()
	AfterInit()
	BeforeShutdown()
	Shutdown()
}
