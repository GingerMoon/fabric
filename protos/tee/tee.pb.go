// Code generated by protoc-gen-go. DO NOT EDIT.
// source: tee/tee.proto

package protos

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	empty "github.com/golang/protobuf/ptypes/empty"
	grpc "google.golang.org/grpc"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion2 // please upgrade the proto package

type TeeArgs struct {
	Elf                  []byte            `protobuf:"bytes,1,opt,name=elf,proto3" json:"elf,omitempty"`
	PlainCipherTexts     *PlainCiphertexts `protobuf:"bytes,2,opt,name=plainCipherTexts,proto3" json:"plainCipherTexts,omitempty"`
	Nonces               [][]byte          `protobuf:"bytes,3,rep,name=nonces,proto3" json:"nonces,omitempty"`
	XXX_NoUnkeyedLiteral struct{}          `json:"-"`
	XXX_unrecognized     []byte            `json:"-"`
	XXX_sizecache        int32             `json:"-"`
}

func (m *TeeArgs) Reset()         { *m = TeeArgs{} }
func (m *TeeArgs) String() string { return proto.CompactTextString(m) }
func (*TeeArgs) ProtoMessage()    {}
func (*TeeArgs) Descriptor() ([]byte, []int) {
	return fileDescriptor_148e4d29062bab63, []int{0}
}

func (m *TeeArgs) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_TeeArgs.Unmarshal(m, b)
}
func (m *TeeArgs) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_TeeArgs.Marshal(b, m, deterministic)
}
func (m *TeeArgs) XXX_Merge(src proto.Message) {
	xxx_messageInfo_TeeArgs.Merge(m, src)
}
func (m *TeeArgs) XXX_Size() int {
	return xxx_messageInfo_TeeArgs.Size(m)
}
func (m *TeeArgs) XXX_DiscardUnknown() {
	xxx_messageInfo_TeeArgs.DiscardUnknown(m)
}

var xxx_messageInfo_TeeArgs proto.InternalMessageInfo

func (m *TeeArgs) GetElf() []byte {
	if m != nil {
		return m.Elf
	}
	return nil
}

func (m *TeeArgs) GetPlainCipherTexts() *PlainCiphertexts {
	if m != nil {
		return m.PlainCipherTexts
	}
	return nil
}

func (m *TeeArgs) GetNonces() [][]byte {
	if m != nil {
		return m.Nonces
	}
	return nil
}

type PlainCiphertexts struct {
	Plaintexts           [][]byte           `protobuf:"bytes,1,rep,name=plaintexts,proto3" json:"plaintexts,omitempty"`
	Feed4Decryptions     []*Feed4Decryption `protobuf:"bytes,2,rep,name=feed4Decryptions,proto3" json:"feed4Decryptions,omitempty"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *PlainCiphertexts) Reset()         { *m = PlainCiphertexts{} }
func (m *PlainCiphertexts) String() string { return proto.CompactTextString(m) }
func (*PlainCiphertexts) ProtoMessage()    {}
func (*PlainCiphertexts) Descriptor() ([]byte, []int) {
	return fileDescriptor_148e4d29062bab63, []int{1}
}

func (m *PlainCiphertexts) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PlainCiphertexts.Unmarshal(m, b)
}
func (m *PlainCiphertexts) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PlainCiphertexts.Marshal(b, m, deterministic)
}
func (m *PlainCiphertexts) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PlainCiphertexts.Merge(m, src)
}
func (m *PlainCiphertexts) XXX_Size() int {
	return xxx_messageInfo_PlainCiphertexts.Size(m)
}
func (m *PlainCiphertexts) XXX_DiscardUnknown() {
	xxx_messageInfo_PlainCiphertexts.DiscardUnknown(m)
}

var xxx_messageInfo_PlainCiphertexts proto.InternalMessageInfo

func (m *PlainCiphertexts) GetPlaintexts() [][]byte {
	if m != nil {
		return m.Plaintexts
	}
	return nil
}

func (m *PlainCiphertexts) GetFeed4Decryptions() []*Feed4Decryption {
	if m != nil {
		return m.Feed4Decryptions
	}
	return nil
}

type Feed4Decryption struct {
	Ciphertext           []byte   `protobuf:"bytes,1,opt,name=ciphertext,proto3" json:"ciphertext,omitempty"`
	Nonce                []byte   `protobuf:"bytes,2,opt,name=nonce,proto3" json:"nonce,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Feed4Decryption) Reset()         { *m = Feed4Decryption{} }
func (m *Feed4Decryption) String() string { return proto.CompactTextString(m) }
func (*Feed4Decryption) ProtoMessage()    {}
func (*Feed4Decryption) Descriptor() ([]byte, []int) {
	return fileDescriptor_148e4d29062bab63, []int{2}
}

func (m *Feed4Decryption) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Feed4Decryption.Unmarshal(m, b)
}
func (m *Feed4Decryption) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Feed4Decryption.Marshal(b, m, deterministic)
}
func (m *Feed4Decryption) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Feed4Decryption.Merge(m, src)
}
func (m *Feed4Decryption) XXX_Size() int {
	return xxx_messageInfo_Feed4Decryption.Size(m)
}
func (m *Feed4Decryption) XXX_DiscardUnknown() {
	xxx_messageInfo_Feed4Decryption.DiscardUnknown(m)
}

var xxx_messageInfo_Feed4Decryption proto.InternalMessageInfo

func (m *Feed4Decryption) GetCiphertext() []byte {
	if m != nil {
		return m.Ciphertext
	}
	return nil
}

func (m *Feed4Decryption) GetNonce() []byte {
	if m != nil {
		return m.Nonce
	}
	return nil
}

type DataKeyArgs struct {
	Datakey              []byte   `protobuf:"bytes,1,opt,name=datakey,proto3" json:"datakey,omitempty"`
	Label                []byte   `protobuf:"bytes,2,opt,name=label,proto3" json:"label,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *DataKeyArgs) Reset()         { *m = DataKeyArgs{} }
func (m *DataKeyArgs) String() string { return proto.CompactTextString(m) }
func (*DataKeyArgs) ProtoMessage()    {}
func (*DataKeyArgs) Descriptor() ([]byte, []int) {
	return fileDescriptor_148e4d29062bab63, []int{3}
}

func (m *DataKeyArgs) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_DataKeyArgs.Unmarshal(m, b)
}
func (m *DataKeyArgs) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_DataKeyArgs.Marshal(b, m, deterministic)
}
func (m *DataKeyArgs) XXX_Merge(src proto.Message) {
	xxx_messageInfo_DataKeyArgs.Merge(m, src)
}
func (m *DataKeyArgs) XXX_Size() int {
	return xxx_messageInfo_DataKeyArgs.Size(m)
}
func (m *DataKeyArgs) XXX_DiscardUnknown() {
	xxx_messageInfo_DataKeyArgs.DiscardUnknown(m)
}

var xxx_messageInfo_DataKeyArgs proto.InternalMessageInfo

func (m *DataKeyArgs) GetDatakey() []byte {
	if m != nil {
		return m.Datakey
	}
	return nil
}

func (m *DataKeyArgs) GetLabel() []byte {
	if m != nil {
		return m.Label
	}
	return nil
}

func init() {
	proto.RegisterType((*TeeArgs)(nil), "protos.TeeArgs")
	proto.RegisterType((*PlainCiphertexts)(nil), "protos.PlainCiphertexts")
	proto.RegisterType((*Feed4Decryption)(nil), "protos.Feed4Decryption")
	proto.RegisterType((*DataKeyArgs)(nil), "protos.DataKeyArgs")
}

func init() { proto.RegisterFile("tee/tee.proto", fileDescriptor_148e4d29062bab63) }

var fileDescriptor_148e4d29062bab63 = []byte{
	// 326 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x74, 0x52, 0x41, 0x4f, 0xb3, 0x40,
	0x10, 0x2d, 0x1f, 0xf9, 0xda, 0x64, 0x5a, 0x53, 0xb2, 0x9a, 0xba, 0xa9, 0x89, 0x69, 0xf6, 0xc4,
	0x89, 0x26, 0xd5, 0x78, 0x33, 0xd1, 0xb4, 0xd5, 0x83, 0x17, 0x43, 0xf8, 0x03, 0x0b, 0x1d, 0x28,
	0x11, 0x59, 0x02, 0xdb, 0x08, 0x27, 0xff, 0xba, 0xd9, 0x85, 0x55, 0x6c, 0xe3, 0x09, 0xde, 0xcc,
	0x7b, 0xbb, 0x6f, 0x66, 0x1f, 0x9c, 0x49, 0xc4, 0xa5, 0x44, 0xf4, 0x8a, 0x52, 0x48, 0x41, 0x86,
	0xfa, 0x53, 0xcd, 0xaf, 0x12, 0x21, 0x92, 0x0c, 0x97, 0x1a, 0x86, 0x87, 0x78, 0x89, 0xef, 0x85,
	0x6c, 0x5a, 0x12, 0x6b, 0x60, 0x14, 0x20, 0x3e, 0x96, 0x49, 0x45, 0x1c, 0xb0, 0x31, 0x8b, 0xa9,
	0xb5, 0xb0, 0xdc, 0x89, 0xaf, 0x7e, 0xc9, 0x06, 0x9c, 0x22, 0xe3, 0x69, 0xbe, 0x4e, 0x8b, 0x3d,
	0x96, 0x01, 0xd6, 0xb2, 0xa2, 0xff, 0x16, 0x96, 0x3b, 0x5e, 0xd1, 0x56, 0x5e, 0x79, 0xaf, 0x3f,
	0x7d, 0xa9, 0xfa, 0xfe, 0x89, 0x82, 0xcc, 0x60, 0x98, 0x8b, 0x3c, 0xc2, 0x8a, 0xda, 0x0b, 0xdb,
	0x9d, 0xf8, 0x1d, 0x62, 0x1f, 0xe0, 0x1c, 0xab, 0xc9, 0x35, 0x80, 0xd6, 0x6b, 0x44, 0x2d, 0xcd,
	0xef, 0x55, 0xc8, 0x1a, 0x9c, 0x18, 0x71, 0x77, 0xbb, 0xc1, 0xa8, 0x6c, 0x0a, 0x99, 0x8a, 0x5c,
	0x39, 0xb2, 0xdd, 0xf1, 0xea, 0xd2, 0x38, 0x7a, 0xfa, 0xdd, 0xf7, 0x4f, 0x04, 0xec, 0x19, 0xa6,
	0x47, 0x24, 0x75, 0x6f, 0xf4, 0x6d, 0xa3, 0x5b, 0x41, 0xaf, 0x42, 0x2e, 0xe0, 0xbf, 0x76, 0xad,
	0xc7, 0x9f, 0xf8, 0x2d, 0x60, 0xf7, 0x30, 0xde, 0x70, 0xc9, 0x5f, 0xb0, 0xd1, 0x0b, 0xa4, 0x30,
	0xda, 0x71, 0xc9, 0xdf, 0xb0, 0xe9, 0x4e, 0x30, 0x50, 0xc9, 0x33, 0x1e, 0x62, 0x66, 0xe4, 0x1a,
	0xac, 0x3e, 0xc1, 0x0e, 0x10, 0xc9, 0x1d, 0x8c, 0xb6, 0x35, 0x46, 0x07, 0x89, 0x64, 0x6a, 0x86,
	0xe8, 0xde, 0x64, 0xfe, 0xe7, 0x9e, 0xd9, 0x80, 0x3c, 0xc0, 0x74, 0x5b, 0x47, 0x7b, 0x9e, 0x27,
	0xd8, 0xb9, 0x20, 0xe7, 0x86, 0xde, 0xb3, 0x35, 0x9f, 0x79, 0x6d, 0x00, 0x3c, 0x13, 0x00, 0x6f,
	0xab, 0x02, 0xc0, 0x06, 0x61, 0x9b, 0x90, 0x9b, 0xaf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xb8, 0x07,
	0x86, 0x40, 0x39, 0x02, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// TeeClient is the client API for Tee service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type TeeClient interface {
	Execute(ctx context.Context, in *TeeArgs, opts ...grpc.CallOption) (*PlainCiphertexts, error)
	ExchangeDataKey(ctx context.Context, in *DataKeyArgs, opts ...grpc.CallOption) (*empty.Empty, error)
}

type teeClient struct {
	cc *grpc.ClientConn
}

func NewTeeClient(cc *grpc.ClientConn) TeeClient {
	return &teeClient{cc}
}

func (c *teeClient) Execute(ctx context.Context, in *TeeArgs, opts ...grpc.CallOption) (*PlainCiphertexts, error) {
	out := new(PlainCiphertexts)
	err := c.cc.Invoke(ctx, "/protos.Tee/Execute", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *teeClient) ExchangeDataKey(ctx context.Context, in *DataKeyArgs, opts ...grpc.CallOption) (*empty.Empty, error) {
	out := new(empty.Empty)
	err := c.cc.Invoke(ctx, "/protos.Tee/ExchangeDataKey", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TeeServer is the server API for Tee service.
type TeeServer interface {
	Execute(context.Context, *TeeArgs) (*PlainCiphertexts, error)
	ExchangeDataKey(context.Context, *DataKeyArgs) (*empty.Empty, error)
}

func RegisterTeeServer(s *grpc.Server, srv TeeServer) {
	s.RegisterService(&_Tee_serviceDesc, srv)
}

func _Tee_Execute_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TeeArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TeeServer).Execute(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Tee/Execute",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TeeServer).Execute(ctx, req.(*TeeArgs))
	}
	return interceptor(ctx, in, info, handler)
}

func _Tee_ExchangeDataKey_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DataKeyArgs)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TeeServer).ExchangeDataKey(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/protos.Tee/ExchangeDataKey",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TeeServer).ExchangeDataKey(ctx, req.(*DataKeyArgs))
	}
	return interceptor(ctx, in, info, handler)
}

var _Tee_serviceDesc = grpc.ServiceDesc{
	ServiceName: "protos.Tee",
	HandlerType: (*TeeServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Execute",
			Handler:    _Tee_Execute_Handler,
		},
		{
			MethodName: "ExchangeDataKey",
			Handler:    _Tee_ExchangeDataKey_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "tee/tee.proto",
}
