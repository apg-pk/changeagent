package raft

import (
  "errors"
  "fmt"
  "sync"
  "sync/atomic"
  "time"
  "math/rand"
  "github.com/golang/glog"
  "github.com/satori/go.uuid"
  "revision.aeip.apigee.net/greg/changeagent/communication"
  "revision.aeip.apigee.net/greg/changeagent/discovery"
  "revision.aeip.apigee.net/greg/changeagent/storage"
)

const (
  // Make these hard-coded rather than "iota" because they go in a database!
  CurrentTermKey = 1
  VotedForKey = 2
  LocalIDKey = 3
  LastAppliedKey = 4
  NodeConfig = 5
)

const (
  ElectionTimeout = 10 * time.Second
  HeartbeatTimeout = 2 * time.Second
)

type State int32

const (
  Follower State = iota
  Candidate
  Leader
  Stopping
  Stopped
)

func (r State) String() string {
  switch r {
  case Follower:
    return "Follower"
  case Candidate:
    return "Candidate"
  case Leader:
    return "Leader"
  case Stopping:
    return "Stopping"
  case Stopped:
    return "Stopped"
  default:
    return ""
  }
}

type Service struct {
  id uint64
  state int32
  leaderID uint64
  comm communication.Communication
  nodeConfig atomic.Value
  configChanges <-chan discovery.Change
  stor storage.Storage
  stopChan chan chan bool
  voteCommands chan voteCommand
  appendCommands chan appendCommand
  proposals chan proposalCommand
  latch sync.Mutex
  followerOnly bool
  currentTerm uint64
  commitIndex uint64
  lastApplied uint64
  lastIndex uint64
  lastTerm uint64
  appliedTracker *ChangeTracker
  stateMachine StateMachine
}

type voteCommand struct {
  vr *communication.VoteRequest
  rc chan *communication.VoteResponse
}

type appendCommand struct {
  ar *communication.AppendRequest
  rc chan *communication.AppendResponse
}

type proposalResult struct {
  index uint64
  err error
}

type proposalCommand struct {
  entry storage.Entry
  rc chan proposalResult
}

var raftRand = makeRand()

func StartRaft(id uint64,
               comm communication.Communication,
               disco discovery.Discovery,
               stor storage.Storage,
               state StateMachine) (*Service, error) {
  r := &Service{
    state: int32(Follower),
    comm: comm,
    nodeConfig: atomic.Value{},
    stor: stor,
    stopChan: make(chan chan bool, 1),
    voteCommands: make(chan voteCommand, 1),
    appendCommands: make(chan appendCommand, 1),
    proposals: make(chan proposalCommand, 1),
    latch: sync.Mutex{},
    followerOnly: false,
    appliedTracker: CreateTracker(),
    stateMachine: state,
  }

  storedID, err := stor.GetMetadata(LocalIDKey)
  if err != nil { return nil, err }
  if storedID == 0 {
    err = stor.SetMetadata(LocalIDKey, id)
    if err != nil { return nil, err }
  } else if id != storedID {
    return nil, fmt.Errorf("ID in data store %d does not match requested value %d",
      storedID, id)
  }
  r.id = id

  err = r.loadCurrentConfig(disco, stor)
  if err != nil { return nil, err }

  if disco.GetAddress(r.id) == "" {
    return nil, fmt.Errorf("Id %d cannot be found in discovery data", r.id)
  }

  r.lastIndex, r.lastTerm, err = r.stor.GetLastIndex()
  if err != nil { return nil, err }

  r.currentTerm = r.readCurrentTerm()
  r.commitIndex = r.readLastCommit()
  r.lastApplied = r.readLastApplied()

  if len(disco.GetNodes()) == 1 {
    glog.Info("Only one node. Starting in leader mode.\n")
    r.state = int32(Leader)
  }

  r.configChanges = disco.Watch()

  go r.mainLoop()

  return r, nil
}

func (r *Service) loadCurrentConfig(disco discovery.Discovery, stor storage.Storage) error {
  buf, err := stor.GetRawMetadata(NodeConfig)
  if err != nil { return err }

  if buf == nil {
    glog.Info("Loading configuration from the discovery file for the first time")
    cfg := disco.GetCurrentConfig()
    storBuf, err := discovery.EncodeConfig(cfg)
    if err != nil { return err }
    err = stor.SetRawMetadata(NodeConfig, storBuf)
    if err != nil { return err }
    r.nodeConfig.Store(cfg)
    return nil
  }

  cfg, err := discovery.DecodeConfig(buf)
  if err != nil { return err }
  r.nodeConfig.Store(cfg)
  return nil
}

func (r *Service) Close() {
  s := r.GetState()
  if s != Stopped && s != Stopping {
    done := make(chan bool)
    r.stopChan <- done
    <- done
  }
  r.appliedTracker.Close()
}

func (r *Service) cleanup() {
  for len(r.voteCommands) > 0 {
    glog.V(2).Info("Sending cleanup command for vote request")
    vc := <- r.voteCommands
    vc.rc <- &communication.VoteResponse{
      Error: errors.New("Raft is shutting down"),
    }
  }
  //close(r.voteCommands)

  for len(r.appendCommands) > 0 {
    glog.V(2).Info("Sending cleanup command for append request")
    vc := <- r.appendCommands
    vc.rc <- &communication.AppendResponse{
      Error: errors.New("Raft is shutting down"),
    }
  }
  //close(r.appendCommands)

  //close(r.receivedAppendChan)
}

func (r *Service) RequestVote(req *communication.VoteRequest) (*communication.VoteResponse, error) {
  if r.GetState() == Stopping || r.GetState() == Stopped {
    return nil, errors.New("Raft is stopped")
  }

  rc := make(chan *communication.VoteResponse)
  cmd := voteCommand{
    vr: req,
    rc: rc,
  }
  r.voteCommands <- cmd
  vr := <- rc
  return vr, vr.Error
}

func (r *Service) Append(req *communication.AppendRequest) (*communication.AppendResponse, error) {
  glog.V(2).Infof("Node %d append request. State is %v", r.id, r.GetState())
  if r.GetState() == Stopping || r.GetState() == Stopped {
    return nil, errors.New("Raft is stopped")
  }

  rc := make(chan *communication.AppendResponse)
  cmd := appendCommand{
    ar: req,
    rc: rc,
  }

  r.appendCommands <- cmd
  resp := <- rc
  return resp, resp.Error
}

func (r *Service) Propose(e *storage.Entry) (uint64, error) {
  if r.GetState() == Stopping || r.GetState() == Stopped {
    return 0, errors.New("Raft is stopped")
  }

  rc := make(chan proposalResult, 1)
  cmd := proposalCommand{
    entry: *e,
    rc: rc,
  }

  glog.V(2).Infof("Going to propose a value of %d bytes", len(e.Data))
  r.proposals <- cmd

  result := <- rc
  return result.index, result.err
}

func (r *Service) MyID() uint64 {
  return r.id
}

func (r *Service) GetState() State {
  s := atomic.LoadInt32(&r.state)
  return State(s)
}

func (r *Service) setState(newState State) {
  glog.V(2).Infof("Node %d: setting state to %d", r.id, newState)
  ns := int32(newState)
  atomic.StoreInt32(&r.state, ns)
}

func (r *Service) GetLeaderID() uint64 {
  return atomic.LoadUint64(&r.leaderID)
}

func (r *Service) setLeaderID(newID uint64) {
  if newID == 0 {
    glog.V(2).Infof("Node %d: No leader present", r.id)
  } else {
    glog.V(2).Infof("Node %d: Node %d is now the leader", r.id, newID)
  }
  atomic.StoreUint64(&r.leaderID, newID)
}

func (r *Service) GetCurrentTerm() uint64 {
  r.latch.Lock()
  defer r.latch.Unlock()
  return r.currentTerm
}

func (r *Service) setCurrentTerm(t uint64) {
  // Use a mutex for this because we write it to DB and want that to be synchronized
  r.latch.Lock()
  defer r.latch.Unlock()
  r.currentTerm = t
  r.writeCurrentTerm(t)
}

func (r *Service) GetCommitIndex() uint64 {
  return atomic.LoadUint64(&r.commitIndex)
}

// Atomically update the commit index, and return whether it changed
func (r *Service) setCommitIndex(t uint64) bool {
  oldIndex := atomic.SwapUint64(&r.commitIndex, t)
  return oldIndex != t
}

func (r *Service) GetLastApplied() uint64 {
  return atomic.LoadUint64(&r.lastApplied)
}

func (r *Service) setLastApplied(t uint64) {
  entry, err := r.stor.GetEntry(t)
  if err != nil {
    glog.Errorf("Error reading entry from change %d for commit: %s", t, err)
    return
  }
  if entry == nil {
    glog.Errorf("Committed entry %d could not be read", t)
    return
  }

  err = r.stateMachine.Commit(entry)
  if err != nil {
    glog.Errorf("Error committing change %d: %s", t, err)
    return
  }

  err = r.stor.SetMetadata(LastAppliedKey, t)
  if err != nil {
    glog.Errorf("Error updating last applied key %d to the database: %s", t, err)
    return
  }

  atomic.StoreUint64(&r.lastApplied, t)

  r.appliedTracker.Update(uuid.Nil, t)

  if !uuid.Equal(entry.Tenant, uuid.Nil) {
    r.appliedTracker.Update(entry.Tenant, t)
  }
}

func (r *Service) GetAppliedTracker() *ChangeTracker {
  return r.appliedTracker
}

func (r *Service) GetLastIndex() (uint64, uint64) {
  // Use a mutex here so that both values are consistent
  r.latch.Lock()
  defer r.latch.Unlock()
  return r.lastIndex, r.lastTerm
}

func (r *Service) setLastIndex(ix uint64, term uint64) {
  r.latch.Lock()
  defer r.latch.Unlock()
  r.lastIndex = ix
  r.lastTerm = term
}

func (r *Service) getNodeConfig() *discovery.NodeConfig {
  return r.nodeConfig.Load().(*discovery.NodeConfig)
}

// Used only in unit testing. Forces us to never become a leader.
func (r *Service) setFollowerOnly(f bool) {
  r.followerOnly = f
}

func (r *Service) readCurrentTerm() uint64 {
  ct, err := r.stor.GetMetadata(CurrentTermKey)
  if err != nil { panic("Fatal error reading state from database") }
  return ct
}

func (r *Service) writeCurrentTerm(ct uint64) {
  err := r.stor.SetMetadata(CurrentTermKey, ct)
  if err != nil { panic("Fatal error writing state to database") }
}

func (r *Service) readLastVote() uint64 {
  ct, err := r.stor.GetMetadata(VotedForKey)
  if err != nil { panic("Fatal error reading state from database") }
  return ct
}

func (r *Service) writeLastVote(ct uint64) {
  err := r.stor.SetMetadata(VotedForKey, ct)
  if err != nil { panic("Fatal error writing state to database") }
}

func (r *Service) readLastCommit() uint64 {
  mi, _, err := r.stor.GetLastIndex()
  if err != nil { panic("Fatal error reading state from database") }
  return mi
}

func (r *Service) readLastApplied() uint64 {
  la, err := r.stor.GetMetadata(LastAppliedKey)
  if err != nil { panic("Fatal error reading state from state machine") }
  return la
}

// Election timeout is the default timeout, plus or minus one heartbeat interval
func (r *Service) randomElectionTimeout() time.Duration {
  rge := int64(HeartbeatTimeout * 2)
  min := int64(ElectionTimeout - HeartbeatTimeout)
  return time.Duration(raftRand.Int63n(rge) + min)
}

func makeRand() *rand.Rand {
  s := rand.NewSource(time.Now().UnixNano())
  return rand.New(s)
}
