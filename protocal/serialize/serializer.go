package serialize

// 对外公开序列化函数, 内部不要使用
var (
	Marshal   func(any) ([]byte, error)
	Unmarshal func([]byte, any) error
)

type (
	// Marshaler represents a marshal interface
	Marshaler interface {
		Marshal(any) ([]byte, error)
	}

	// Unmarshaler represents a Unmarshal interface
	Unmarshaler interface {
		Unmarshal([]byte, any) error
	}

	// Serializer is the interface that groups the basic Marshal and Unmarshal methods.
	Serializer interface {
		Marshaler
		Unmarshaler
	}
)
