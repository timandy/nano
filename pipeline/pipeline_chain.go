package pipeline

import (
	"sync"

	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/session"
)

type pipelineChain struct {
	mu       sync.RWMutex
	handlers []HandlerFunc
}

func (c *pipelineChain) PushFront(h HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append([]HandlerFunc{h}, c.handlers...)
}

func (c *pipelineChain) PushBack(h HandlerFunc) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers = append(c.handlers, h)
}

func (c *pipelineChain) Process(s *session.Session, msg *message.Message) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for _, h := range c.handlers {
		err := h(s, msg)
		if err != nil {
			return err
		}
	}
	return nil
}
