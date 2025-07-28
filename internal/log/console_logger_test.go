package log

import (
	"errors"
	"testing"
)

func TestInfof(t *testing.T) {
	Info("hello %v", "abc")
	Info("hello", "abc")
	Info("hello", errors.New("abc"))
}
