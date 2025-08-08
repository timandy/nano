package mock

// NetAddr mock the net.Addr interface
type NetAddr struct{}

// Network implements the net.Addr interface
func (a NetAddr) Network() string { return "mock" }

// String implements the net.Addr interface
func (a NetAddr) String() string { return "mock-addr" }
