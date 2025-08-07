package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewDefaultConnectionServer(t *testing.T) {
	service := newSnowflakeConnection()
	sidChan := make(chan int64, paraCount)
	defer close(sidChan)
	// gen id
	for i := 0; i < paraCount; i++ {
		go func() {
			sidChan <- service.SessionID()
		}()
	}
	// predicate
	mp := make(map[int64]struct{}, paraCount)
	for i := 0; i < paraCount; i++ {
		sid := <-sidChan
		if _, ok := mp[sid]; ok {
			assert.Fail(t, "duplicate session id")
		} else {
			mp[sid] = struct{}{}
		}
	}
}
