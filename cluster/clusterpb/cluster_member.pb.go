// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.5
// 	protoc        v5.29.4
// source: cluster_member.proto

package clusterpb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type NewMemberRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	MemberInfo    *MemberInfo            `protobuf:"bytes,1,opt,name=memberInfo,proto3" json:"memberInfo,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NewMemberRequest) Reset() {
	*x = NewMemberRequest{}
	mi := &file_cluster_member_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NewMemberRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewMemberRequest) ProtoMessage() {}

func (x *NewMemberRequest) ProtoReflect() protoreflect.Message {
	mi := &file_cluster_member_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewMemberRequest.ProtoReflect.Descriptor instead.
func (*NewMemberRequest) Descriptor() ([]byte, []int) {
	return file_cluster_member_proto_rawDescGZIP(), []int{0}
}

func (x *NewMemberRequest) GetMemberInfo() *MemberInfo {
	if x != nil {
		return x.MemberInfo
	}
	return nil
}

type NewMemberResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *NewMemberResponse) Reset() {
	*x = NewMemberResponse{}
	mi := &file_cluster_member_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *NewMemberResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*NewMemberResponse) ProtoMessage() {}

func (x *NewMemberResponse) ProtoReflect() protoreflect.Message {
	mi := &file_cluster_member_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use NewMemberResponse.ProtoReflect.Descriptor instead.
func (*NewMemberResponse) Descriptor() ([]byte, []int) {
	return file_cluster_member_proto_rawDescGZIP(), []int{1}
}

type DelMemberRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ServiceAddr   string                 `protobuf:"bytes,1,opt,name=serviceAddr,proto3" json:"serviceAddr,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DelMemberRequest) Reset() {
	*x = DelMemberRequest{}
	mi := &file_cluster_member_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DelMemberRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DelMemberRequest) ProtoMessage() {}

func (x *DelMemberRequest) ProtoReflect() protoreflect.Message {
	mi := &file_cluster_member_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DelMemberRequest.ProtoReflect.Descriptor instead.
func (*DelMemberRequest) Descriptor() ([]byte, []int) {
	return file_cluster_member_proto_rawDescGZIP(), []int{2}
}

func (x *DelMemberRequest) GetServiceAddr() string {
	if x != nil {
		return x.ServiceAddr
	}
	return ""
}

type DelMemberResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *DelMemberResponse) Reset() {
	*x = DelMemberResponse{}
	mi := &file_cluster_member_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *DelMemberResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DelMemberResponse) ProtoMessage() {}

func (x *DelMemberResponse) ProtoReflect() protoreflect.Message {
	mi := &file_cluster_member_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DelMemberResponse.ProtoReflect.Descriptor instead.
func (*DelMemberResponse) Descriptor() ([]byte, []int) {
	return file_cluster_member_proto_rawDescGZIP(), []int{3}
}

var File_cluster_member_proto protoreflect.FileDescriptor

var file_cluster_member_proto_rawDesc = string([]byte{
	0x0a, 0x14, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x6d, 0x65, 0x6d, 0x62, 0x65, 0x72,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x09, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x70,
	0x62, 0x1a, 0x12, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x5f, 0x62, 0x61, 0x73, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0x49, 0x0a, 0x10, 0x4e, 0x65, 0x77, 0x4d, 0x65, 0x6d, 0x62,
	0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x35, 0x0a, 0x0a, 0x6d, 0x65, 0x6d,
	0x62, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x15, 0x2e,
	0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x70, 0x62, 0x2e, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72,
	0x49, 0x6e, 0x66, 0x6f, 0x52, 0x0a, 0x6d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x49, 0x6e, 0x66, 0x6f,
	0x22, 0x13, 0x0a, 0x11, 0x4e, 0x65, 0x77, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x34, 0x0a, 0x10, 0x44, 0x65, 0x6c, 0x4d, 0x65, 0x6d, 0x62,
	0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x20, 0x0a, 0x0b, 0x73, 0x65, 0x72,
	0x76, 0x69, 0x63, 0x65, 0x41, 0x64, 0x64, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x0b,
	0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x41, 0x64, 0x64, 0x72, 0x22, 0x13, 0x0a, 0x11, 0x44,
	0x65, 0x6c, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x32, 0x9c, 0x01, 0x0a, 0x06, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x12, 0x48, 0x0a, 0x09, 0x4e,
	0x65, 0x77, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x12, 0x1b, 0x2e, 0x63, 0x6c, 0x75, 0x73, 0x74,
	0x65, 0x72, 0x70, 0x62, 0x2e, 0x4e, 0x65, 0x77, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x1c, 0x2e, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x70,
	0x62, 0x2e, 0x4e, 0x65, 0x77, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x48, 0x0a, 0x09, 0x44, 0x65, 0x6c, 0x4d, 0x65, 0x6d, 0x62,
	0x65, 0x72, 0x12, 0x1b, 0x2e, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x70, 0x62, 0x2e, 0x44,
	0x65, 0x6c, 0x4d, 0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a,
	0x1c, 0x2e, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x70, 0x62, 0x2e, 0x44, 0x65, 0x6c, 0x4d,
	0x65, 0x6d, 0x62, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42,
	0x0c, 0x5a, 0x0a, 0x2f, 0x63, 0x6c, 0x75, 0x73, 0x74, 0x65, 0x72, 0x70, 0x62, 0x62, 0x06, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x33,
})

var (
	file_cluster_member_proto_rawDescOnce sync.Once
	file_cluster_member_proto_rawDescData []byte
)

func file_cluster_member_proto_rawDescGZIP() []byte {
	file_cluster_member_proto_rawDescOnce.Do(func() {
		file_cluster_member_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_cluster_member_proto_rawDesc), len(file_cluster_member_proto_rawDesc)))
	})
	return file_cluster_member_proto_rawDescData
}

var file_cluster_member_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_cluster_member_proto_goTypes = []any{
	(*NewMemberRequest)(nil),  // 0: clusterpb.NewMemberRequest
	(*NewMemberResponse)(nil), // 1: clusterpb.NewMemberResponse
	(*DelMemberRequest)(nil),  // 2: clusterpb.DelMemberRequest
	(*DelMemberResponse)(nil), // 3: clusterpb.DelMemberResponse
	(*MemberInfo)(nil),        // 4: clusterpb.MemberInfo
}
var file_cluster_member_proto_depIdxs = []int32{
	4, // 0: clusterpb.NewMemberRequest.memberInfo:type_name -> clusterpb.MemberInfo
	0, // 1: clusterpb.Member.NewMember:input_type -> clusterpb.NewMemberRequest
	2, // 2: clusterpb.Member.DelMember:input_type -> clusterpb.DelMemberRequest
	1, // 3: clusterpb.Member.NewMember:output_type -> clusterpb.NewMemberResponse
	3, // 4: clusterpb.Member.DelMember:output_type -> clusterpb.DelMemberResponse
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_cluster_member_proto_init() }
func file_cluster_member_proto_init() {
	if File_cluster_member_proto != nil {
		return
	}
	file_cluster_base_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_cluster_member_proto_rawDesc), len(file_cluster_member_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_cluster_member_proto_goTypes,
		DependencyIndexes: file_cluster_member_proto_depIdxs,
		MessageInfos:      file_cluster_member_proto_msgTypes,
	}.Build()
	File_cluster_member_proto = out.File
	file_cluster_member_proto_goTypes = nil
	file_cluster_member_proto_depIdxs = nil
}
