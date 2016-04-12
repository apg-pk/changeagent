package communication

import (
  "fmt"
  "revision.aeip.apigee.net/greg/changeagent/storage"
)

type Raft interface {
  MyID() uint64
  RequestVote(req *VoteRequest) (*VoteResponse, error)
  Append(req *AppendRequest) (*AppendResponse, error)
  Propose(e *storage.Entry) (uint64, error)
}

type VoteRequest struct {
  Term uint64
  CandidateID uint64
  LastLogIndex uint64
  LastLogTerm uint64
}

type VoteResponse struct {
  NodeID uint64
  Term uint64
  VoteGranted bool
  Error error
}

type AppendRequest struct {
  Term uint64
  LeaderID uint64
  PrevLogIndex uint64
  PrevLogTerm uint64
  LeaderCommit uint64
  Entries []storage.Entry
}

func (a *AppendRequest) String() string {
  s := fmt.Sprintf("AppendRequest{ Term: %d Leader: %d PrevIx: %d PrevTerm: %d LeaderCommit: %d [",
    a.Term, a.LeaderID, a.PrevLogIndex, a.PrevLogTerm, a.LeaderCommit)
  for _, e := range(a.Entries) {
    s += e.String()
  }
  s += " ]}"
  return s
}

type AppendResponse struct {
  Term uint64
  Success bool
  CommitIndex uint64
  Error error
}

var DefaultAppendResponse = AppendResponse{}

func (a *AppendResponse) String() string {
  s := fmt.Sprintf("AppendResponse{ Term: %d Success: %v CommitIndex: %d ",
    a.Term, a.Success, a.CommitIndex)
  if a.Error != nil {
    s += fmt.Sprintf("Error: %s ", a.Error)
  }
  s += " }"
  return s
}

type ProposalResponse struct {
  NewIndex uint64
  Error error
}

var DefaultProposalResponse = ProposalResponse{}

func (a *ProposalResponse) String() string {
  s := fmt.Sprintf("ProposalResponse{ NewIndex: %d ", a.NewIndex)
  if a.Error != nil {
    s += fmt.Sprintf("Error: %s ", a.Error)
  }
  s += " }"
  return s
}

type Communication interface {
  SetRaft(raft Raft)
  RequestVote(id uint64, req VoteRequest, ch chan<- VoteResponse)
  Append(id uint64, req *AppendRequest) (AppendResponse, error)
  Propose(id uint64, e *storage.Entry) (ProposalResponse, error)
}
