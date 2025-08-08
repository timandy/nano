package component

import "github.com/lonng/nano/scheduler/schedulerapi"

type (
	options struct {
		name     string                // component name
		nameFunc func(string) string   // rename handler name
		executor schedulerapi.Executor // executor for the component
	}

	// Option used to customize handler
	Option func(options *options)
)

// WithName used to rename component name
func WithName(name string) Option {
	return func(opt *options) {
		opt.name = name
	}
}

// WithNameFunc override handler name by specific function
// such as: strings.ToUpper/strings.ToLower
func WithNameFunc(fn func(string) string) Option {
	return func(opt *options) {
		opt.nameFunc = fn
	}
}

// WithExecutor set the name of the service executor
func WithExecutor(executor schedulerapi.Executor) Option {
	return func(opt *options) {
		opt.executor = executor
	}
}
