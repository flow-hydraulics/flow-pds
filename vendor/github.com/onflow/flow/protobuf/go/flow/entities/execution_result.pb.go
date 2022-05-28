// Code generated by protoc-gen-go. DO NOT EDIT.
// source: flow/entities/execution_result.proto

package entities

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
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
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type ExecutionResult struct {
	PreviousResultId     []byte          `protobuf:"bytes,1,opt,name=previous_result_id,json=previousResultId,proto3" json:"previous_result_id,omitempty"`
	BlockId              []byte          `protobuf:"bytes,2,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	Chunks               []*Chunk        `protobuf:"bytes,3,rep,name=chunks,proto3" json:"chunks,omitempty"`
	ServiceEvents        []*ServiceEvent `protobuf:"bytes,4,rep,name=service_events,json=serviceEvents,proto3" json:"service_events,omitempty"`
	ExecutionDataId      []byte          `protobuf:"bytes,5,opt,name=execution_data_id,json=executionDataId,proto3" json:"execution_data_id,omitempty"` // Deprecated: Do not use.
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *ExecutionResult) Reset()         { *m = ExecutionResult{} }
func (m *ExecutionResult) String() string { return proto.CompactTextString(m) }
func (*ExecutionResult) ProtoMessage()    {}
func (*ExecutionResult) Descriptor() ([]byte, []int) {
	return fileDescriptor_806371cb5e5e336b, []int{0}
}

func (m *ExecutionResult) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ExecutionResult.Unmarshal(m, b)
}
func (m *ExecutionResult) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ExecutionResult.Marshal(b, m, deterministic)
}
func (m *ExecutionResult) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ExecutionResult.Merge(m, src)
}
func (m *ExecutionResult) XXX_Size() int {
	return xxx_messageInfo_ExecutionResult.Size(m)
}
func (m *ExecutionResult) XXX_DiscardUnknown() {
	xxx_messageInfo_ExecutionResult.DiscardUnknown(m)
}

var xxx_messageInfo_ExecutionResult proto.InternalMessageInfo

func (m *ExecutionResult) GetPreviousResultId() []byte {
	if m != nil {
		return m.PreviousResultId
	}
	return nil
}

func (m *ExecutionResult) GetBlockId() []byte {
	if m != nil {
		return m.BlockId
	}
	return nil
}

func (m *ExecutionResult) GetChunks() []*Chunk {
	if m != nil {
		return m.Chunks
	}
	return nil
}

func (m *ExecutionResult) GetServiceEvents() []*ServiceEvent {
	if m != nil {
		return m.ServiceEvents
	}
	return nil
}

// Deprecated: Do not use.
func (m *ExecutionResult) GetExecutionDataId() []byte {
	if m != nil {
		return m.ExecutionDataId
	}
	return nil
}

type Chunk struct {
	CollectionIndex      uint32   `protobuf:"varint,1,opt,name=CollectionIndex,proto3" json:"CollectionIndex,omitempty"`
	StartState           []byte   `protobuf:"bytes,2,opt,name=start_state,json=startState,proto3" json:"start_state,omitempty"`
	EventCollection      []byte   `protobuf:"bytes,3,opt,name=event_collection,json=eventCollection,proto3" json:"event_collection,omitempty"`
	BlockId              []byte   `protobuf:"bytes,4,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	TotalComputationUsed uint64   `protobuf:"varint,5,opt,name=total_computation_used,json=totalComputationUsed,proto3" json:"total_computation_used,omitempty"`
	NumberOfTransactions uint32   `protobuf:"varint,6,opt,name=number_of_transactions,json=numberOfTransactions,proto3" json:"number_of_transactions,omitempty"`
	Index                uint64   `protobuf:"varint,7,opt,name=index,proto3" json:"index,omitempty"`
	EndState             []byte   `protobuf:"bytes,8,opt,name=end_state,json=endState,proto3" json:"end_state,omitempty"`
	ExecutionDataId      []byte   `protobuf:"bytes,9,opt,name=execution_data_id,json=executionDataId,proto3" json:"execution_data_id,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Chunk) Reset()         { *m = Chunk{} }
func (m *Chunk) String() string { return proto.CompactTextString(m) }
func (*Chunk) ProtoMessage()    {}
func (*Chunk) Descriptor() ([]byte, []int) {
	return fileDescriptor_806371cb5e5e336b, []int{1}
}

func (m *Chunk) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Chunk.Unmarshal(m, b)
}
func (m *Chunk) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Chunk.Marshal(b, m, deterministic)
}
func (m *Chunk) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Chunk.Merge(m, src)
}
func (m *Chunk) XXX_Size() int {
	return xxx_messageInfo_Chunk.Size(m)
}
func (m *Chunk) XXX_DiscardUnknown() {
	xxx_messageInfo_Chunk.DiscardUnknown(m)
}

var xxx_messageInfo_Chunk proto.InternalMessageInfo

func (m *Chunk) GetCollectionIndex() uint32 {
	if m != nil {
		return m.CollectionIndex
	}
	return 0
}

func (m *Chunk) GetStartState() []byte {
	if m != nil {
		return m.StartState
	}
	return nil
}

func (m *Chunk) GetEventCollection() []byte {
	if m != nil {
		return m.EventCollection
	}
	return nil
}

func (m *Chunk) GetBlockId() []byte {
	if m != nil {
		return m.BlockId
	}
	return nil
}

func (m *Chunk) GetTotalComputationUsed() uint64 {
	if m != nil {
		return m.TotalComputationUsed
	}
	return 0
}

func (m *Chunk) GetNumberOfTransactions() uint32 {
	if m != nil {
		return m.NumberOfTransactions
	}
	return 0
}

func (m *Chunk) GetIndex() uint64 {
	if m != nil {
		return m.Index
	}
	return 0
}

func (m *Chunk) GetEndState() []byte {
	if m != nil {
		return m.EndState
	}
	return nil
}

func (m *Chunk) GetExecutionDataId() []byte {
	if m != nil {
		return m.ExecutionDataId
	}
	return nil
}

type ServiceEvent struct {
	Type                 string   `protobuf:"bytes,1,opt,name=type,proto3" json:"type,omitempty"`
	Payload              []byte   `protobuf:"bytes,2,opt,name=payload,proto3" json:"payload,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ServiceEvent) Reset()         { *m = ServiceEvent{} }
func (m *ServiceEvent) String() string { return proto.CompactTextString(m) }
func (*ServiceEvent) ProtoMessage()    {}
func (*ServiceEvent) Descriptor() ([]byte, []int) {
	return fileDescriptor_806371cb5e5e336b, []int{2}
}

func (m *ServiceEvent) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ServiceEvent.Unmarshal(m, b)
}
func (m *ServiceEvent) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ServiceEvent.Marshal(b, m, deterministic)
}
func (m *ServiceEvent) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ServiceEvent.Merge(m, src)
}
func (m *ServiceEvent) XXX_Size() int {
	return xxx_messageInfo_ServiceEvent.Size(m)
}
func (m *ServiceEvent) XXX_DiscardUnknown() {
	xxx_messageInfo_ServiceEvent.DiscardUnknown(m)
}

var xxx_messageInfo_ServiceEvent proto.InternalMessageInfo

func (m *ServiceEvent) GetType() string {
	if m != nil {
		return m.Type
	}
	return ""
}

func (m *ServiceEvent) GetPayload() []byte {
	if m != nil {
		return m.Payload
	}
	return nil
}

type ExecutionReceiptMeta struct {
	ExecutorId           []byte   `protobuf:"bytes,1,opt,name=executor_id,json=executorId,proto3" json:"executor_id,omitempty"`
	ResultId             []byte   `protobuf:"bytes,2,opt,name=result_id,json=resultId,proto3" json:"result_id,omitempty"`
	Spocks               [][]byte `protobuf:"bytes,3,rep,name=spocks,proto3" json:"spocks,omitempty"`
	ExecutorSignature    []byte   `protobuf:"bytes,4,opt,name=executor_signature,json=executorSignature,proto3" json:"executor_signature,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ExecutionReceiptMeta) Reset()         { *m = ExecutionReceiptMeta{} }
func (m *ExecutionReceiptMeta) String() string { return proto.CompactTextString(m) }
func (*ExecutionReceiptMeta) ProtoMessage()    {}
func (*ExecutionReceiptMeta) Descriptor() ([]byte, []int) {
	return fileDescriptor_806371cb5e5e336b, []int{3}
}

func (m *ExecutionReceiptMeta) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ExecutionReceiptMeta.Unmarshal(m, b)
}
func (m *ExecutionReceiptMeta) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ExecutionReceiptMeta.Marshal(b, m, deterministic)
}
func (m *ExecutionReceiptMeta) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ExecutionReceiptMeta.Merge(m, src)
}
func (m *ExecutionReceiptMeta) XXX_Size() int {
	return xxx_messageInfo_ExecutionReceiptMeta.Size(m)
}
func (m *ExecutionReceiptMeta) XXX_DiscardUnknown() {
	xxx_messageInfo_ExecutionReceiptMeta.DiscardUnknown(m)
}

var xxx_messageInfo_ExecutionReceiptMeta proto.InternalMessageInfo

func (m *ExecutionReceiptMeta) GetExecutorId() []byte {
	if m != nil {
		return m.ExecutorId
	}
	return nil
}

func (m *ExecutionReceiptMeta) GetResultId() []byte {
	if m != nil {
		return m.ResultId
	}
	return nil
}

func (m *ExecutionReceiptMeta) GetSpocks() [][]byte {
	if m != nil {
		return m.Spocks
	}
	return nil
}

func (m *ExecutionReceiptMeta) GetExecutorSignature() []byte {
	if m != nil {
		return m.ExecutorSignature
	}
	return nil
}

func init() {
	proto.RegisterType((*ExecutionResult)(nil), "flow.entities.ExecutionResult")
	proto.RegisterType((*Chunk)(nil), "flow.entities.Chunk")
	proto.RegisterType((*ServiceEvent)(nil), "flow.entities.ServiceEvent")
	proto.RegisterType((*ExecutionReceiptMeta)(nil), "flow.entities.ExecutionReceiptMeta")
}

func init() {
	proto.RegisterFile("flow/entities/execution_result.proto", fileDescriptor_806371cb5e5e336b)
}

var fileDescriptor_806371cb5e5e336b = []byte{
	// 518 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x6c, 0x53, 0x4d, 0x6f, 0xd3, 0x40,
	0x10, 0x55, 0x9a, 0xef, 0x69, 0x42, 0xda, 0x55, 0x54, 0x19, 0x15, 0x89, 0x2a, 0xe2, 0x10, 0x50,
	0xb1, 0x11, 0x70, 0xe4, 0x94, 0xd0, 0x43, 0x0e, 0x08, 0xe4, 0xc0, 0x85, 0x8b, 0xb5, 0xb1, 0x27,
	0xa9, 0x55, 0x67, 0xd7, 0xda, 0x1d, 0x87, 0xf6, 0xb7, 0xf0, 0xf3, 0xf8, 0x1b, 0x1c, 0x90, 0xc7,
	0x76, 0x9c, 0x44, 0x5c, 0x2c, 0xcf, 0xbc, 0x37, 0xb3, 0xf3, 0x66, 0xdf, 0xc2, 0xab, 0x75, 0xa2,
	0x7f, 0x79, 0xa8, 0x28, 0xa6, 0x18, 0xad, 0x87, 0x8f, 0x18, 0x66, 0x14, 0x6b, 0x15, 0x18, 0xb4,
	0x59, 0x42, 0x6e, 0x6a, 0x34, 0x69, 0x31, 0xcc, 0x59, 0x6e, 0xc5, 0x9a, 0xfc, 0x6d, 0xc0, 0xe8,
	0xae, 0x62, 0xfa, 0x4c, 0x14, 0xb7, 0x20, 0x52, 0x83, 0xbb, 0x58, 0x67, 0xb6, 0xac, 0x0d, 0xe2,
	0xc8, 0x69, 0xdc, 0x34, 0xa6, 0x03, 0xff, 0xa2, 0x42, 0x0a, 0xee, 0x22, 0x12, 0xcf, 0xa1, 0xb7,
	0x4a, 0x74, 0xf8, 0x90, 0x73, 0xce, 0x98, 0xd3, 0xe5, 0x78, 0x11, 0x89, 0x5b, 0xe8, 0x84, 0xf7,
	0x99, 0x7a, 0xb0, 0x4e, 0xf3, 0xa6, 0x39, 0x3d, 0x7f, 0x3f, 0x76, 0x8f, 0x0e, 0x77, 0xe7, 0x39,
	0xe8, 0x97, 0x1c, 0x31, 0x83, 0x67, 0x16, 0xcd, 0x2e, 0x0e, 0x31, 0xc0, 0x1d, 0x2a, 0xb2, 0x4e,
	0x8b, 0xab, 0xae, 0x4f, 0xaa, 0x96, 0x05, 0xe9, 0x2e, 0xe7, 0xf8, 0x43, 0x7b, 0x10, 0x59, 0xe1,
	0xc2, 0x65, 0xad, 0x3b, 0x92, 0x24, 0xf3, 0xa9, 0xda, 0xf9, 0x54, 0xb3, 0x33, 0xa7, 0xe1, 0x8f,
	0xf6, 0xe0, 0x67, 0x49, 0x72, 0x11, 0x4d, 0xfe, 0x9c, 0x41, 0x9b, 0xa7, 0x10, 0x53, 0x18, 0xcd,
	0x75, 0x92, 0x60, 0x98, 0xa3, 0x0b, 0x15, 0xe1, 0x23, 0x2b, 0x1e, 0xfa, 0xa7, 0x69, 0xf1, 0x12,
	0xce, 0x2d, 0x49, 0x43, 0x81, 0x25, 0x49, 0x58, 0x6a, 0x06, 0x4e, 0x2d, 0xf3, 0x8c, 0x78, 0x0d,
	0x17, 0x2c, 0x20, 0x08, 0xf7, 0x95, 0x4e, 0x93, 0x59, 0x23, 0xce, 0xd7, 0x0d, 0x8f, 0x96, 0xd7,
	0x3a, 0x5e, 0xde, 0x47, 0xb8, 0x22, 0x4d, 0x32, 0x09, 0x42, 0xbd, 0x4d, 0x33, 0x92, 0x2c, 0x29,
	0xb3, 0x58, 0xe8, 0x69, 0xf9, 0x63, 0x46, 0xe7, 0x35, 0xf8, 0xc3, 0x22, 0x57, 0xa9, 0x6c, 0xbb,
	0x42, 0x13, 0xe8, 0x75, 0x40, 0x46, 0x2a, 0x2b, 0xf9, 0x24, 0xeb, 0x74, 0x58, 0xcd, 0xb8, 0x40,
	0xbf, 0xae, 0xbf, 0x1f, 0x60, 0x62, 0x0c, 0xed, 0x98, 0x25, 0x77, 0xb9, 0x75, 0x11, 0x88, 0x6b,
	0xe8, 0xa3, 0x8a, 0x4a, 0x99, 0x3d, 0x9e, 0xae, 0x87, 0x2a, 0x2a, 0x44, 0xbe, 0xf9, 0xdf, 0xa6,
	0xfb, 0xa5, 0xca, 0x93, 0x2d, 0x7f, 0x82, 0xc1, 0xe1, 0xa5, 0x09, 0x01, 0x2d, 0x7a, 0x4a, 0x91,
	0x17, 0xdc, 0xf7, 0xf9, 0x5f, 0x38, 0xd0, 0x4d, 0xe5, 0x53, 0xa2, 0xe5, 0xde, 0x45, 0x65, 0x38,
	0xf9, 0xdd, 0x80, 0xf1, 0x81, 0x45, 0x43, 0x8c, 0x53, 0xfa, 0x82, 0x24, 0xf3, 0x8b, 0x28, 0x4e,
	0xd2, 0xa6, 0x36, 0x28, 0x54, 0xa9, 0x45, 0x94, 0x0b, 0xa8, 0xfd, 0x5b, 0x74, 0xed, 0x99, 0xca,
	0xb7, 0x57, 0xd0, 0xb1, 0xa9, 0x0e, 0x4b, 0x73, 0x0e, 0xfc, 0x32, 0x12, 0x6f, 0x41, 0xec, 0xbb,
	0xda, 0x78, 0xa3, 0x24, 0x65, 0x06, 0xcb, 0xcb, 0xb9, 0xac, 0x90, 0x65, 0x05, 0xcc, 0xbe, 0xc1,
	0x0b, 0x6d, 0x36, 0xae, 0x56, 0x6c, 0x52, 0x7e, 0x63, 0xab, 0x6c, 0xbd, 0x77, 0xeb, 0xcf, 0x77,
	0x9b, 0x98, 0xee, 0xb3, 0x95, 0x1b, 0xea, 0xad, 0x57, 0x90, 0x3c, 0xfe, 0x54, 0x4c, 0x6f, 0xa3,
	0xbd, 0xa3, 0x87, 0xbb, 0xea, 0x30, 0xf4, 0xe1, 0x5f, 0x00, 0x00, 0x00, 0xff, 0xff, 0x8a, 0xcf,
	0x7e, 0x5f, 0xd0, 0x03, 0x00, 0x00,
}