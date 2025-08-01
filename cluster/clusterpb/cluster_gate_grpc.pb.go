// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.4
// source: cluster_gate.proto

package clusterpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Gate_HandlePush_FullMethodName     = "/clusterpb.Gate/HandlePush"
	Gate_HandleResponse_FullMethodName = "/clusterpb.Gate/HandleResponse"
	Gate_CloseSession_FullMethodName   = "/clusterpb.Gate/CloseSession"
)

// GateClient is the client API for Gate service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GateClient interface {
	HandlePush(ctx context.Context, in *PushMessage, opts ...grpc.CallOption) (*GateHandleResponse, error)
	HandleResponse(ctx context.Context, in *ResponseMessage, opts ...grpc.CallOption) (*GateHandleResponse, error)
	CloseSession(ctx context.Context, in *CloseSessionRequest, opts ...grpc.CallOption) (*CloseSessionResponse, error)
}

type gateClient struct {
	cc grpc.ClientConnInterface
}

func NewGateClient(cc grpc.ClientConnInterface) GateClient {
	return &gateClient{cc}
}

func (c *gateClient) HandlePush(ctx context.Context, in *PushMessage, opts ...grpc.CallOption) (*GateHandleResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GateHandleResponse)
	err := c.cc.Invoke(ctx, Gate_HandlePush_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gateClient) HandleResponse(ctx context.Context, in *ResponseMessage, opts ...grpc.CallOption) (*GateHandleResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GateHandleResponse)
	err := c.cc.Invoke(ctx, Gate_HandleResponse_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *gateClient) CloseSession(ctx context.Context, in *CloseSessionRequest, opts ...grpc.CallOption) (*CloseSessionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(CloseSessionResponse)
	err := c.cc.Invoke(ctx, Gate_CloseSession_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// GateServer is the server API for Gate service.
// All implementations should embed UnimplementedGateServer
// for forward compatibility.
type GateServer interface {
	HandlePush(context.Context, *PushMessage) (*GateHandleResponse, error)
	HandleResponse(context.Context, *ResponseMessage) (*GateHandleResponse, error)
	CloseSession(context.Context, *CloseSessionRequest) (*CloseSessionResponse, error)
}

// UnimplementedGateServer should be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedGateServer struct{}

func (UnimplementedGateServer) HandlePush(context.Context, *PushMessage) (*GateHandleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HandlePush not implemented")
}
func (UnimplementedGateServer) HandleResponse(context.Context, *ResponseMessage) (*GateHandleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method HandleResponse not implemented")
}
func (UnimplementedGateServer) CloseSession(context.Context, *CloseSessionRequest) (*CloseSessionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CloseSession not implemented")
}
func (UnimplementedGateServer) testEmbeddedByValue() {}

// UnsafeGateServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GateServer will
// result in compilation errors.
type UnsafeGateServer interface {
	mustEmbedUnimplementedGateServer()
}

func RegisterGateServer(s grpc.ServiceRegistrar, srv GateServer) {
	// If the following call pancis, it indicates UnimplementedGateServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Gate_ServiceDesc, srv)
}

func _Gate_HandlePush_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(PushMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GateServer).HandlePush(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Gate_HandlePush_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GateServer).HandlePush(ctx, req.(*PushMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gate_HandleResponse_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ResponseMessage)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GateServer).HandleResponse(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Gate_HandleResponse_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GateServer).HandleResponse(ctx, req.(*ResponseMessage))
	}
	return interceptor(ctx, in, info, handler)
}

func _Gate_CloseSession_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CloseSessionRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(GateServer).CloseSession(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Gate_CloseSession_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(GateServer).CloseSession(ctx, req.(*CloseSessionRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Gate_ServiceDesc is the grpc.ServiceDesc for Gate service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Gate_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "clusterpb.Gate",
	HandlerType: (*GateServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "HandlePush",
			Handler:    _Gate_HandlePush_Handler,
		},
		{
			MethodName: "HandleResponse",
			Handler:    _Gate_HandleResponse_Handler,
		},
		{
			MethodName: "CloseSession",
			Handler:    _Gate_CloseSession_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "cluster_gate.proto",
}
