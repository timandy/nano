package component

var _ Component = Base{}
var _ Component = (*Base)(nil)

// Base implements a default component for Component.
type Base struct {
}

// Init was called to initialize the component.
func (c Base) Init() {}

// AfterInit was called after the component is initialized.
func (c Base) AfterInit() {}

// BeforeShutdown was called before the component to shutdown.
func (c Base) BeforeShutdown() {}

// Shutdown was called to shutdown the component.
func (c Base) Shutdown() {}
