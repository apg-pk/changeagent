package main

import (
  "net/http"
  "revision.aeip.apigee.net/greg/changeagent/communication"
  "revision.aeip.apigee.net/greg/changeagent/discovery"
  "revision.aeip.apigee.net/greg/changeagent/raft"
  "revision.aeip.apigee.net/greg/changeagent/storage"
)

type ChangeAgent struct {
  stor storage.Storage
  raft *raft.RaftImpl
}

func StartChangeAgent(nodeId uint64,
                      disco discovery.Discovery,
                      dbFile string,
                      mux *http.ServeMux) (*ChangeAgent, error) {
  comm, err := communication.StartHttpCommunication(mux, disco)
  if err != nil { return nil, err }
  stor, err := storage.CreateLevelDBStorage(dbFile)
  if err != nil { return nil, err }

  agent := &ChangeAgent{
    stor: stor,
  }

  raft, err := raft.StartRaft(nodeId, comm, disco, stor)
  if err != nil { return nil, err }
  agent.raft = raft
  comm.SetRaft(raft)

  initDiagnosticApi(mux)
  agent.initAPI(mux)

  return agent, nil
}

func (a *ChangeAgent) Close() {
  a.raft.Close()
  a.stor.Close()
}

func (a *ChangeAgent) Delete() {
  a.stor.Delete()
}

func (a *ChangeAgent) GetRaftState() int {
  return a.raft.GetState()
}