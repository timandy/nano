package pipeline

import (
	"github.com/lonng/nano/protocal/message"
	"github.com/lonng/nano/session"
)

// HandlerFunc is handler function
type HandlerFunc func(s *session.Session, msg *message.Message) error

// Pipeline defines the interface of a message pipeline
type Pipeline interface {
	// Inbound return the inbound pipeline chain
	Inbound() PipelineChain

	// Outbound return the outbound pipeline chain
	Outbound() PipelineChain
}

// PipelineChain defines the interface of a pipeline chain
type PipelineChain interface {
	// PushFront push a function to the front of the chain
	PushFront(h HandlerFunc)

	// PushBack push a function to the end of the chain
	PushBack(h HandlerFunc)

	// Process message with all handler functions
	Process(s *session.Session, msg *message.Message) error
}
