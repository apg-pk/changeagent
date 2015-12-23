// Code generated by protoc-gen-go.
// source: communication.proto
// DO NOT EDIT!

/*
Package communication is a generated protocol buffer package.

It is generated from these files:
	communication.proto

It has these top-level messages:
	VoteRequestPb
	VoteResponsePb
	EntryPb
	AppendRequestPb
	AppendResponsePb
*/
package communication

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type VoteRequestPb struct {
	Term             *uint64 `protobuf:"varint,1,req,name=term" json:"term,omitempty"`
	CandidateId      *uint64 `protobuf:"varint,2,req,name=candidateId" json:"candidateId,omitempty"`
	LastLogIndex     *uint64 `protobuf:"varint,3,req,name=lastLogIndex" json:"lastLogIndex,omitempty"`
	LastLogTerm      *uint64 `protobuf:"varint,4,req,name=lastLogTerm" json:"lastLogTerm,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *VoteRequestPb) Reset()         { *m = VoteRequestPb{} }
func (m *VoteRequestPb) String() string { return proto.CompactTextString(m) }
func (*VoteRequestPb) ProtoMessage()    {}

func (m *VoteRequestPb) GetTerm() uint64 {
	if m != nil && m.Term != nil {
		return *m.Term
	}
	return 0
}

func (m *VoteRequestPb) GetCandidateId() uint64 {
	if m != nil && m.CandidateId != nil {
		return *m.CandidateId
	}
	return 0
}

func (m *VoteRequestPb) GetLastLogIndex() uint64 {
	if m != nil && m.LastLogIndex != nil {
		return *m.LastLogIndex
	}
	return 0
}

func (m *VoteRequestPb) GetLastLogTerm() uint64 {
	if m != nil && m.LastLogTerm != nil {
		return *m.LastLogTerm
	}
	return 0
}

type VoteResponsePb struct {
	NodeId           *uint64 `protobuf:"varint,1,req,name=nodeId" json:"nodeId,omitempty"`
	Term             *uint64 `protobuf:"varint,2,req,name=term" json:"term,omitempty"`
	VoteGranted      *bool   `protobuf:"varint,3,req,name=voteGranted" json:"voteGranted,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *VoteResponsePb) Reset()         { *m = VoteResponsePb{} }
func (m *VoteResponsePb) String() string { return proto.CompactTextString(m) }
func (*VoteResponsePb) ProtoMessage()    {}

func (m *VoteResponsePb) GetNodeId() uint64 {
	if m != nil && m.NodeId != nil {
		return *m.NodeId
	}
	return 0
}

func (m *VoteResponsePb) GetTerm() uint64 {
	if m != nil && m.Term != nil {
		return *m.Term
	}
	return 0
}

func (m *VoteResponsePb) GetVoteGranted() bool {
	if m != nil && m.VoteGranted != nil {
		return *m.VoteGranted
	}
	return false
}

type EntryPb struct {
	Index            *uint64 `protobuf:"varint,1,req,name=index" json:"index,omitempty"`
	Term             *uint64 `protobuf:"varint,2,req,name=term" json:"term,omitempty"`
	Data             []byte  `protobuf:"bytes,3,opt,name=data" json:"data,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *EntryPb) Reset()         { *m = EntryPb{} }
func (m *EntryPb) String() string { return proto.CompactTextString(m) }
func (*EntryPb) ProtoMessage()    {}

func (m *EntryPb) GetIndex() uint64 {
	if m != nil && m.Index != nil {
		return *m.Index
	}
	return 0
}

func (m *EntryPb) GetTerm() uint64 {
	if m != nil && m.Term != nil {
		return *m.Term
	}
	return 0
}

func (m *EntryPb) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

type AppendRequestPb struct {
	Term             *uint64    `protobuf:"varint,1,req,name=term" json:"term,omitempty"`
	LeaderId         *uint64    `protobuf:"varint,2,req,name=leaderId" json:"leaderId,omitempty"`
	PrevLogIndex     *uint64    `protobuf:"varint,3,req,name=prevLogIndex" json:"prevLogIndex,omitempty"`
	PrevLogTerm      *uint64    `protobuf:"varint,4,req,name=prevLogTerm" json:"prevLogTerm,omitempty"`
	LeaderCommit     *uint64    `protobuf:"varint,5,req,name=leaderCommit" json:"leaderCommit,omitempty"`
	Entries          []*EntryPb `protobuf:"bytes,6,rep,name=entries" json:"entries,omitempty"`
	XXX_unrecognized []byte     `json:"-"`
}

func (m *AppendRequestPb) Reset()         { *m = AppendRequestPb{} }
func (m *AppendRequestPb) String() string { return proto.CompactTextString(m) }
func (*AppendRequestPb) ProtoMessage()    {}

func (m *AppendRequestPb) GetTerm() uint64 {
	if m != nil && m.Term != nil {
		return *m.Term
	}
	return 0
}

func (m *AppendRequestPb) GetLeaderId() uint64 {
	if m != nil && m.LeaderId != nil {
		return *m.LeaderId
	}
	return 0
}

func (m *AppendRequestPb) GetPrevLogIndex() uint64 {
	if m != nil && m.PrevLogIndex != nil {
		return *m.PrevLogIndex
	}
	return 0
}

func (m *AppendRequestPb) GetPrevLogTerm() uint64 {
	if m != nil && m.PrevLogTerm != nil {
		return *m.PrevLogTerm
	}
	return 0
}

func (m *AppendRequestPb) GetLeaderCommit() uint64 {
	if m != nil && m.LeaderCommit != nil {
		return *m.LeaderCommit
	}
	return 0
}

func (m *AppendRequestPb) GetEntries() []*EntryPb {
	if m != nil {
		return m.Entries
	}
	return nil
}

type AppendResponsePb struct {
	Term             *uint64 `protobuf:"varint,1,req,name=term" json:"term,omitempty"`
	Success          *bool   `protobuf:"varint,2,req,name=success" json:"success,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *AppendResponsePb) Reset()         { *m = AppendResponsePb{} }
func (m *AppendResponsePb) String() string { return proto.CompactTextString(m) }
func (*AppendResponsePb) ProtoMessage()    {}

func (m *AppendResponsePb) GetTerm() uint64 {
	if m != nil && m.Term != nil {
		return *m.Term
	}
	return 0
}

func (m *AppendResponsePb) GetSuccess() bool {
	if m != nil && m.Success != nil {
		return *m.Success
	}
	return false
}
