// Code generated by protoc-gen-go.
// source: webhookconfig.proto
// DO NOT EDIT!

/*
Package hooks is a generated protocol buffer package.

It is generated from these files:
	webhookconfig.proto

It has these top-level messages:
	HeaderPb
	WebHookPb
	WebHookConfigPb
*/
package hooks

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type HeaderPb struct {
	Name             *string `protobuf:"bytes,1,req,name=name" json:"name,omitempty"`
	Value            *string `protobuf:"bytes,2,opt,name=value" json:"value,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *HeaderPb) Reset()         { *m = HeaderPb{} }
func (m *HeaderPb) String() string { return proto.CompactTextString(m) }
func (*HeaderPb) ProtoMessage()    {}

func (m *HeaderPb) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *HeaderPb) GetValue() string {
	if m != nil && m.Value != nil {
		return *m.Value
	}
	return ""
}

type WebHookPb struct {
	Url              *string     `protobuf:"bytes,1,req,name=url" json:"url,omitempty"`
	Headers          []*HeaderPb `protobuf:"bytes,2,rep,name=headers" json:"headers,omitempty"`
	XXX_unrecognized []byte      `json:"-"`
}

func (m *WebHookPb) Reset()         { *m = WebHookPb{} }
func (m *WebHookPb) String() string { return proto.CompactTextString(m) }
func (*WebHookPb) ProtoMessage()    {}

func (m *WebHookPb) GetUrl() string {
	if m != nil && m.Url != nil {
		return *m.Url
	}
	return ""
}

func (m *WebHookPb) GetHeaders() []*HeaderPb {
	if m != nil {
		return m.Headers
	}
	return nil
}

type WebHookConfigPb struct {
	Hooks            []*WebHookPb `protobuf:"bytes,1,rep,name=hooks" json:"hooks,omitempty"`
	XXX_unrecognized []byte       `json:"-"`
}

func (m *WebHookConfigPb) Reset()         { *m = WebHookConfigPb{} }
func (m *WebHookConfigPb) String() string { return proto.CompactTextString(m) }
func (*WebHookConfigPb) ProtoMessage()    {}

func (m *WebHookConfigPb) GetHooks() []*WebHookPb {
	if m != nil {
		return m.Hooks
	}
	return nil
}
