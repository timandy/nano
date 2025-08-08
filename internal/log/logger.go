package log

type Logger interface {
	Info(args ...any)
	Error(args ...any)
	Fatal(args ...any)
}

func init() {
	SetLogger(NewConsoleLogger())
}

var (
	Info  func(args ...any)
	Error func(args ...any)
	Fatal func(args ...any)
)

// SetLogger rewrites the default logger
func SetLogger(logger Logger) {
	if logger == nil {
		return
	}
	Info = logger.Info
	Error = logger.Error
	Fatal = logger.Fatal
}
