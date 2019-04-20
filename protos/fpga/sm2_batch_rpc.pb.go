// Code generated by protoc-gen-go. DO NOT EDIT.
// source: fpga/sm2_batch_rpc.proto

package fpga

import (
	context "context"
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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

// 批申请数据结构
type BatchRequest struct {
	BatchId              uint64                         `protobuf:"varint,1,opt,name=batch_id,json=batchId,proto3" json:"batch_id,omitempty"`
	BatchType            uint32                         `protobuf:"varint,2,opt,name=batch_type,json=batchType,proto3" json:"batch_type,omitempty"`
	ReqCount             uint32                         `protobuf:"varint,3,opt,name=req_count,json=reqCount,proto3" json:"req_count,omitempty"`
	SgRequests           []*BatchRequest_SignGenRequest `protobuf:"bytes,4,rep,name=sg_requests,json=sgRequests,proto3" json:"sg_requests,omitempty"`
	SvRequests           []*BatchRequest_SignVerRequest `protobuf:"bytes,5,rep,name=sv_requests,json=svRequests,proto3" json:"sv_requests,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                       `json:"-"`
	XXX_unrecognized     []byte                         `json:"-"`
	XXX_sizecache        int32                          `json:"-"`
}

func (m *BatchRequest) Reset()         { *m = BatchRequest{} }
func (m *BatchRequest) String() string { return proto.CompactTextString(m) }
func (*BatchRequest) ProtoMessage()    {}
func (*BatchRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_6c6d6f2f964022bf, []int{0}
}

func (m *BatchRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchRequest.Unmarshal(m, b)
}
func (m *BatchRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchRequest.Marshal(b, m, deterministic)
}
func (m *BatchRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchRequest.Merge(m, src)
}
func (m *BatchRequest) XXX_Size() int {
	return xxx_messageInfo_BatchRequest.Size(m)
}
func (m *BatchRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BatchRequest proto.InternalMessageInfo

func (m *BatchRequest) GetBatchId() uint64 {
	if m != nil {
		return m.BatchId
	}
	return 0
}

func (m *BatchRequest) GetBatchType() uint32 {
	if m != nil {
		return m.BatchType
	}
	return 0
}

func (m *BatchRequest) GetReqCount() uint32 {
	if m != nil {
		return m.ReqCount
	}
	return 0
}

func (m *BatchRequest) GetSgRequests() []*BatchRequest_SignGenRequest {
	if m != nil {
		return m.SgRequests
	}
	return nil
}

func (m *BatchRequest) GetSvRequests() []*BatchRequest_SignVerRequest {
	if m != nil {
		return m.SvRequests
	}
	return nil
}

// 签名生成请求数据结构
type BatchRequest_SignGenRequest struct {
	ReqId                string   `protobuf:"bytes,1,opt,name=req_id,json=reqId,proto3" json:"req_id,omitempty"`
	D                    []byte   `protobuf:"bytes,2,opt,name=d,proto3" json:"d,omitempty"`
	Hash                 []byte   `protobuf:"bytes,3,opt,name=hash,proto3" json:"hash,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BatchRequest_SignGenRequest) Reset()         { *m = BatchRequest_SignGenRequest{} }
func (m *BatchRequest_SignGenRequest) String() string { return proto.CompactTextString(m) }
func (*BatchRequest_SignGenRequest) ProtoMessage()    {}
func (*BatchRequest_SignGenRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_6c6d6f2f964022bf, []int{0, 0}
}

func (m *BatchRequest_SignGenRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchRequest_SignGenRequest.Unmarshal(m, b)
}
func (m *BatchRequest_SignGenRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchRequest_SignGenRequest.Marshal(b, m, deterministic)
}
func (m *BatchRequest_SignGenRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchRequest_SignGenRequest.Merge(m, src)
}
func (m *BatchRequest_SignGenRequest) XXX_Size() int {
	return xxx_messageInfo_BatchRequest_SignGenRequest.Size(m)
}
func (m *BatchRequest_SignGenRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchRequest_SignGenRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BatchRequest_SignGenRequest proto.InternalMessageInfo

func (m *BatchRequest_SignGenRequest) GetReqId() string {
	if m != nil {
		return m.ReqId
	}
	return ""
}

func (m *BatchRequest_SignGenRequest) GetD() []byte {
	if m != nil {
		return m.D
	}
	return nil
}

func (m *BatchRequest_SignGenRequest) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

// 签名验证请求数据结构
type BatchRequest_SignVerRequest struct {
	ReqId                string   `protobuf:"bytes,1,opt,name=req_id,json=reqId,proto3" json:"req_id,omitempty"`
	SignR                []byte   `protobuf:"bytes,2,opt,name=sign_r,json=signR,proto3" json:"sign_r,omitempty"`
	SignS                []byte   `protobuf:"bytes,3,opt,name=sign_s,json=signS,proto3" json:"sign_s,omitempty"`
	Hash                 []byte   `protobuf:"bytes,4,opt,name=hash,proto3" json:"hash,omitempty"`
	Px                   []byte   `protobuf:"bytes,5,opt,name=px,proto3" json:"px,omitempty"`
	Py                   []byte   `protobuf:"bytes,6,opt,name=py,proto3" json:"py,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BatchRequest_SignVerRequest) Reset()         { *m = BatchRequest_SignVerRequest{} }
func (m *BatchRequest_SignVerRequest) String() string { return proto.CompactTextString(m) }
func (*BatchRequest_SignVerRequest) ProtoMessage()    {}
func (*BatchRequest_SignVerRequest) Descriptor() ([]byte, []int) {
	return fileDescriptor_6c6d6f2f964022bf, []int{0, 1}
}

func (m *BatchRequest_SignVerRequest) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchRequest_SignVerRequest.Unmarshal(m, b)
}
func (m *BatchRequest_SignVerRequest) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchRequest_SignVerRequest.Marshal(b, m, deterministic)
}
func (m *BatchRequest_SignVerRequest) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchRequest_SignVerRequest.Merge(m, src)
}
func (m *BatchRequest_SignVerRequest) XXX_Size() int {
	return xxx_messageInfo_BatchRequest_SignVerRequest.Size(m)
}
func (m *BatchRequest_SignVerRequest) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchRequest_SignVerRequest.DiscardUnknown(m)
}

var xxx_messageInfo_BatchRequest_SignVerRequest proto.InternalMessageInfo

func (m *BatchRequest_SignVerRequest) GetReqId() string {
	if m != nil {
		return m.ReqId
	}
	return ""
}

func (m *BatchRequest_SignVerRequest) GetSignR() []byte {
	if m != nil {
		return m.SignR
	}
	return nil
}

func (m *BatchRequest_SignVerRequest) GetSignS() []byte {
	if m != nil {
		return m.SignS
	}
	return nil
}

func (m *BatchRequest_SignVerRequest) GetHash() []byte {
	if m != nil {
		return m.Hash
	}
	return nil
}

func (m *BatchRequest_SignVerRequest) GetPx() []byte {
	if m != nil {
		return m.Px
	}
	return nil
}

func (m *BatchRequest_SignVerRequest) GetPy() []byte {
	if m != nil {
		return m.Py
	}
	return nil
}

// 批回复数据结构
type BatchReply struct {
	BatchId              uint64                     `protobuf:"varint,1,opt,name=batch_id,json=batchId,proto3" json:"batch_id,omitempty"`
	BatchType            uint32                     `protobuf:"varint,2,opt,name=batch_type,json=batchType,proto3" json:"batch_type,omitempty"`
	RepCount             uint32                     `protobuf:"varint,3,opt,name=rep_count,json=repCount,proto3" json:"rep_count,omitempty"`
	SgReplies            []*BatchReply_SignGenReply `protobuf:"bytes,4,rep,name=sg_replies,json=sgReplies,proto3" json:"sg_replies,omitempty"`
	SvReplies            []*BatchReply_SignVerReply `protobuf:"bytes,5,rep,name=sv_replies,json=svReplies,proto3" json:"sv_replies,omitempty"`
	XXX_NoUnkeyedLiteral struct{}                   `json:"-"`
	XXX_unrecognized     []byte                     `json:"-"`
	XXX_sizecache        int32                      `json:"-"`
}

func (m *BatchReply) Reset()         { *m = BatchReply{} }
func (m *BatchReply) String() string { return proto.CompactTextString(m) }
func (*BatchReply) ProtoMessage()    {}
func (*BatchReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_6c6d6f2f964022bf, []int{1}
}

func (m *BatchReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchReply.Unmarshal(m, b)
}
func (m *BatchReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchReply.Marshal(b, m, deterministic)
}
func (m *BatchReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchReply.Merge(m, src)
}
func (m *BatchReply) XXX_Size() int {
	return xxx_messageInfo_BatchReply.Size(m)
}
func (m *BatchReply) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchReply.DiscardUnknown(m)
}

var xxx_messageInfo_BatchReply proto.InternalMessageInfo

func (m *BatchReply) GetBatchId() uint64 {
	if m != nil {
		return m.BatchId
	}
	return 0
}

func (m *BatchReply) GetBatchType() uint32 {
	if m != nil {
		return m.BatchType
	}
	return 0
}

func (m *BatchReply) GetRepCount() uint32 {
	if m != nil {
		return m.RepCount
	}
	return 0
}

func (m *BatchReply) GetSgReplies() []*BatchReply_SignGenReply {
	if m != nil {
		return m.SgReplies
	}
	return nil
}

func (m *BatchReply) GetSvReplies() []*BatchReply_SignVerReply {
	if m != nil {
		return m.SvReplies
	}
	return nil
}

// 签名生成回复数据结构
type BatchReply_SignGenReply struct {
	ReqId                string   `protobuf:"bytes,1,opt,name=req_id,json=reqId,proto3" json:"req_id,omitempty"`
	SignR                []byte   `protobuf:"bytes,2,opt,name=sign_r,json=signR,proto3" json:"sign_r,omitempty"`
	SignS                []byte   `protobuf:"bytes,3,opt,name=sign_s,json=signS,proto3" json:"sign_s,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BatchReply_SignGenReply) Reset()         { *m = BatchReply_SignGenReply{} }
func (m *BatchReply_SignGenReply) String() string { return proto.CompactTextString(m) }
func (*BatchReply_SignGenReply) ProtoMessage()    {}
func (*BatchReply_SignGenReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_6c6d6f2f964022bf, []int{1, 0}
}

func (m *BatchReply_SignGenReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchReply_SignGenReply.Unmarshal(m, b)
}
func (m *BatchReply_SignGenReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchReply_SignGenReply.Marshal(b, m, deterministic)
}
func (m *BatchReply_SignGenReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchReply_SignGenReply.Merge(m, src)
}
func (m *BatchReply_SignGenReply) XXX_Size() int {
	return xxx_messageInfo_BatchReply_SignGenReply.Size(m)
}
func (m *BatchReply_SignGenReply) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchReply_SignGenReply.DiscardUnknown(m)
}

var xxx_messageInfo_BatchReply_SignGenReply proto.InternalMessageInfo

func (m *BatchReply_SignGenReply) GetReqId() string {
	if m != nil {
		return m.ReqId
	}
	return ""
}

func (m *BatchReply_SignGenReply) GetSignR() []byte {
	if m != nil {
		return m.SignR
	}
	return nil
}

func (m *BatchReply_SignGenReply) GetSignS() []byte {
	if m != nil {
		return m.SignS
	}
	return nil
}

// 签名验证回复数据结构
type BatchReply_SignVerReply struct {
	ReqId                string   `protobuf:"bytes,1,opt,name=req_id,json=reqId,proto3" json:"req_id,omitempty"`
	Verified             bool     `protobuf:"varint,2,opt,name=verified,proto3" json:"verified,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *BatchReply_SignVerReply) Reset()         { *m = BatchReply_SignVerReply{} }
func (m *BatchReply_SignVerReply) String() string { return proto.CompactTextString(m) }
func (*BatchReply_SignVerReply) ProtoMessage()    {}
func (*BatchReply_SignVerReply) Descriptor() ([]byte, []int) {
	return fileDescriptor_6c6d6f2f964022bf, []int{1, 1}
}

func (m *BatchReply_SignVerReply) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_BatchReply_SignVerReply.Unmarshal(m, b)
}
func (m *BatchReply_SignVerReply) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_BatchReply_SignVerReply.Marshal(b, m, deterministic)
}
func (m *BatchReply_SignVerReply) XXX_Merge(src proto.Message) {
	xxx_messageInfo_BatchReply_SignVerReply.Merge(m, src)
}
func (m *BatchReply_SignVerReply) XXX_Size() int {
	return xxx_messageInfo_BatchReply_SignVerReply.Size(m)
}
func (m *BatchReply_SignVerReply) XXX_DiscardUnknown() {
	xxx_messageInfo_BatchReply_SignVerReply.DiscardUnknown(m)
}

var xxx_messageInfo_BatchReply_SignVerReply proto.InternalMessageInfo

func (m *BatchReply_SignVerReply) GetReqId() string {
	if m != nil {
		return m.ReqId
	}
	return ""
}

func (m *BatchReply_SignVerReply) GetVerified() bool {
	if m != nil {
		return m.Verified
	}
	return false
}

func init() {
	proto.RegisterType((*BatchRequest)(nil), "fpga.BatchRequest")
	proto.RegisterType((*BatchRequest_SignGenRequest)(nil), "fpga.BatchRequest.SignGenRequest")
	proto.RegisterType((*BatchRequest_SignVerRequest)(nil), "fpga.BatchRequest.SignVerRequest")
	proto.RegisterType((*BatchReply)(nil), "fpga.BatchReply")
	proto.RegisterType((*BatchReply_SignGenReply)(nil), "fpga.BatchReply.SignGenReply")
	proto.RegisterType((*BatchReply_SignVerReply)(nil), "fpga.BatchReply.SignVerReply")
}

func init() { proto.RegisterFile("fpga/sm2_batch_rpc.proto", fileDescriptor_6c6d6f2f964022bf) }

var fileDescriptor_6c6d6f2f964022bf = []byte{
	// 416 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x93, 0x4f, 0xcf, 0xd2, 0x40,
	0x10, 0xc6, 0x6d, 0x69, 0x6b, 0x3b, 0x6f, 0x7d, 0x63, 0x36, 0x79, 0x93, 0x5a, 0xf3, 0x26, 0xc8,
	0x89, 0x53, 0x35, 0x78, 0xf5, 0x22, 0x1c, 0x0c, 0x37, 0xb3, 0x18, 0xae, 0x0d, 0xd0, 0xa5, 0x34,
	0xa9, 0x65, 0xbb, 0x5b, 0x1a, 0xf6, 0x23, 0xf8, 0xbd, 0xfc, 0x4e, 0x5e, 0xcd, 0xfe, 0xa1, 0x05,
	0x0c, 0xc4, 0x44, 0x6f, 0x9d, 0x99, 0x9d, 0xdf, 0xb3, 0x33, 0xcf, 0x16, 0xa2, 0x2d, 0xcd, 0x57,
	0xef, 0xf9, 0xf7, 0x49, 0xba, 0x5e, 0x35, 0x9b, 0x5d, 0xca, 0xe8, 0x26, 0xa1, 0x6c, 0xdf, 0xec,
	0x91, 0x23, 0x2b, 0xa3, 0x9f, 0x03, 0x08, 0xa7, 0xb2, 0x82, 0x49, 0x7d, 0x20, 0xbc, 0x41, 0x6f,
	0xc0, 0xd7, 0x27, 0x8b, 0x2c, 0xb2, 0x86, 0xd6, 0xd8, 0xc1, 0x2f, 0x55, 0x3c, 0xcf, 0xd0, 0x33,
	0x80, 0x2e, 0x35, 0x82, 0x92, 0xc8, 0x1e, 0x5a, 0xe3, 0x57, 0x38, 0x50, 0x99, 0x6f, 0x82, 0x12,
	0xf4, 0x16, 0x02, 0x46, 0xea, 0x74, 0xb3, 0x3f, 0x54, 0x4d, 0x34, 0x50, 0x55, 0x9f, 0x91, 0x7a,
	0x26, 0x63, 0x34, 0x85, 0x07, 0x9e, 0xa7, 0x4c, 0x8b, 0xf0, 0xc8, 0x19, 0x0e, 0xc6, 0x0f, 0x93,
	0x77, 0x89, 0xbc, 0x43, 0x72, 0xae, 0x9f, 0x2c, 0x8a, 0xbc, 0xfa, 0x42, 0x2a, 0x13, 0x62, 0xe0,
	0xb9, 0xf9, 0xe4, 0x8a, 0xd1, 0xf6, 0x0c, 0xf7, 0x2e, 0x63, 0x49, 0x58, 0xcf, 0x68, 0x4f, 0x8c,
	0x78, 0x0e, 0x8f, 0x97, 0x0a, 0xe8, 0x09, 0x3c, 0x79, 0x6d, 0x33, 0x6e, 0x80, 0x5d, 0x46, 0xea,
	0x79, 0x86, 0x42, 0xb0, 0x32, 0x35, 0x63, 0x88, 0xad, 0x0c, 0x21, 0x70, 0x76, 0x2b, 0xbe, 0x53,
	0x63, 0x85, 0x58, 0x7d, 0xc7, 0x3f, 0x2c, 0xcd, 0xea, 0x95, 0x6e, 0xb1, 0x9e, 0xc0, 0xe3, 0x45,
	0x5e, 0xa5, 0xcc, 0x00, 0x5d, 0x19, 0xe1, 0x2e, 0xcd, 0x0d, 0x56, 0xa5, 0x17, 0x9d, 0x96, 0xd3,
	0x6b, 0xa1, 0x47, 0xb0, 0xe9, 0x31, 0x72, 0x55, 0xc6, 0xa6, 0x47, 0x15, 0x8b, 0xc8, 0x33, 0xb1,
	0x18, 0xfd, 0xb2, 0x01, 0xcc, 0x0a, 0x68, 0x29, 0xfe, 0xd5, 0x44, 0x7a, 0x6d, 0x22, 0xd5, 0x26,
	0x7e, 0x02, 0x50, 0x26, 0xd2, 0xb2, 0x20, 0x27, 0x0f, 0x9f, 0x2f, 0xf6, 0x4f, 0x4b, 0xd1, 0x3b,
	0x48, 0x4b, 0x81, 0x03, 0xe9, 0x9f, 0x3a, 0xaf, 0xba, 0xdb, 0xae, 0xdb, 0xbd, 0xd3, 0xad, 0x36,
	0xaa, 0xbb, 0x5b, 0xd3, 0x1d, 0x2f, 0x20, 0x3c, 0x07, 0xff, 0x97, 0x55, 0xc7, 0x9f, 0x35, 0xf4,
	0xa4, 0x77, 0x0b, 0x1a, 0x83, 0xdf, 0x12, 0x56, 0x6c, 0x0b, 0xa2, 0x9f, 0x84, 0x8f, 0xbb, 0x78,
	0x52, 0x82, 0xaf, 0x6f, 0xff, 0x75, 0x86, 0x12, 0x70, 0x24, 0x0e, 0xa1, 0x3f, 0xdf, 0x64, 0xfc,
	0xfa, 0x7a, 0xd2, 0xd1, 0x0b, 0xf4, 0x01, 0xbc, 0xa5, 0xe4, 0x88, 0xbf, 0xed, 0x58, 0x7b, 0xea,
	0xdf, 0xfd, 0xf8, 0x3b, 0x00, 0x00, 0xff, 0xff, 0x3b, 0x15, 0x2e, 0x96, 0xd7, 0x03, 0x00, 0x00,
}

// Reference imports to suppress errors if they are not otherwise used.
var _ context.Context
var _ grpc.ClientConn

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc.SupportPackageIsVersion4

// BatchRPCClient is the client API for BatchRPC service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://godoc.org/google.golang.org/grpc#ClientConn.NewStream.
type BatchRPCClient interface {
	// Sends a Batch
	Sign(ctx context.Context, in *BatchRequest, opts ...grpc.CallOption) (*BatchReply, error)
	Verify(ctx context.Context, in *BatchRequest, opts ...grpc.CallOption) (*BatchReply, error)
}

type batchRPCClient struct {
	cc *grpc.ClientConn
}

func NewBatchRPCClient(cc *grpc.ClientConn) BatchRPCClient {
	return &batchRPCClient{cc}
}

func (c *batchRPCClient) Sign(ctx context.Context, in *BatchRequest, opts ...grpc.CallOption) (*BatchReply, error) {
	out := new(BatchReply)
	err := c.cc.Invoke(ctx, "/fpga.BatchRPC/Sign", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *batchRPCClient) Verify(ctx context.Context, in *BatchRequest, opts ...grpc.CallOption) (*BatchReply, error) {
	out := new(BatchReply)
	err := c.cc.Invoke(ctx, "/fpga.BatchRPC/Verify", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// BatchRPCServer is the server API for BatchRPC service.
type BatchRPCServer interface {
	// Sends a Batch
	Sign(context.Context, *BatchRequest) (*BatchReply, error)
	Verify(context.Context, *BatchRequest) (*BatchReply, error)
}

func RegisterBatchRPCServer(s *grpc.Server, srv BatchRPCServer) {
	s.RegisterService(&_BatchRPC_serviceDesc, srv)
}

func _BatchRPC_Sign_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchRPCServer).Sign(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/fpga.BatchRPC/Sign",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchRPCServer).Sign(ctx, req.(*BatchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _BatchRPC_Verify_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BatchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(BatchRPCServer).Verify(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/fpga.BatchRPC/Verify",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(BatchRPCServer).Verify(ctx, req.(*BatchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var _BatchRPC_serviceDesc = grpc.ServiceDesc{
	ServiceName: "fpga.BatchRPC",
	HandlerType: (*BatchRPCServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Sign",
			Handler:    _BatchRPC_Sign_Handler,
		},
		{
			MethodName: "Verify",
			Handler:    _BatchRPC_Verify_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "fpga/sm2_batch_rpc.proto",
}
