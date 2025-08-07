package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const paraCount = 500000

func TestNewConnectionService(t *testing.T) {
	service := newCounterConnection()
	sidChan := make(chan int64, paraCount)
	defer close(sidChan)
	// gen id
	for i := 0; i < paraCount; i++ {
		go func() {
			sidChan <- service.SessionID()
		}()
	}
	// predicate
	for i := 0; i < paraCount; i++ {
		<-sidChan
	}
	assert.EqualValues(t, paraCount+1, service.SessionID())
}
