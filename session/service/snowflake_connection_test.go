package service

import (
	"testing"
)

func TestNewDefaultConnectionServer(t *testing.T) {
	service := newSnowflakeConnection()
	w := make(chan bool, paraCount)
	sidChan := make(chan int64, paraCount)
	for i := 0; i < paraCount; i++ {
		go func() {
			service.Increment()
			w <- true
			sidChan <- service.SessionID()
		}()
	}
	smap := make(map[int64]struct{}, paraCount)
	for i := 0; i < paraCount; i++ {
		<-w
		sid := <-sidChan
		if _, ok := smap[sid]; ok {
			t.Error("wrong session id repeat")
		} else {
			smap[sid] = struct{}{}
		}
	}
	if service.Count() != paraCount {
		t.Error("wrong connection count")
	}
}
