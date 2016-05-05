package main

import (
  "encoding/json"
  "fmt"
  "runtime"
  "net/http"
  "revision.aeip.apigee.net/greg/changeagent/raft"
)

const (
  BaseURI = "/diagnostics"
  StackURI = BaseURI + "/stack"
  IDURI = BaseURI + "/id"
  RaftURI = BaseURI + "/raft"
  RaftStateURI = RaftURI + "/state"
  RaftLeaderURI = RaftURI + "/leader"

  PlainText = "text/plain"
  JSON = "application/json"
)

type RaftState struct {
  State string `json:"state"`
  Leader uint64 `json:"leader"`
}

func (a *ChangeAgent) initDiagnosticAPI() {
  a.router.HandleFunc("/", a.handleRootCall).Methods("GET")
  a.router.HandleFunc(BaseURI, a.handleDiagRootCall).Methods("GET")
  a.router.HandleFunc(IDURI, a.handleIDCall).Methods("GET")
  a.router.HandleFunc(RaftStateURI, a.handleStateCall).Methods("GET")
  a.router.HandleFunc(RaftLeaderURI, a.handleLeaderCall).Methods("GET")
  a.router.HandleFunc(RaftURI, a.handleRaftInfo).Methods("GET")
  a.router.HandleFunc(StackURI, handleStackCall).Methods("GET")
}


func (a *ChangeAgent) handleRootCall(resp http.ResponseWriter, req *http.Request) {
  links := make(map[string]string)
  // TODO convert links properly
  links["changes"] = "./changes"
  links["diagnostics"] = "./diagnostics"

  body, _ := json.Marshal(&links)

  resp.Header().Set("Content-Type", JSON)
  resp.Write(body)
}

func (a *ChangeAgent) handleDiagRootCall(resp http.ResponseWriter, req *http.Request) {
  links := make(map[string]string)
  // TODO convert links properly
  links["id"] = IDURI
  links["stack"] = StackURI
  links["raft"] = RaftURI
  body, _ := json.Marshal(&links)

  resp.Header().Set("Content-Type", JSON)
  resp.Write(body)
}

func (a *ChangeAgent) handleIDCall(resp http.ResponseWriter, req *http.Request) {
  msg := fmt.Sprintf("%d\n", a.raft.MyID())
  resp.Header().Set("Content-Type", PlainText)
  resp.Write([]byte(msg))
}

func (a *ChangeAgent) handleStateCall(resp http.ResponseWriter, req *http.Request) {
  resp.Header().Set("Content-Type", PlainText)
  resp.Write([]byte(a.GetRaftState().String()))
}

func (a *ChangeAgent) handleLeaderCall(resp http.ResponseWriter, req *http.Request) {
  resp.Header().Set("Content-Type", PlainText)
  resp.Write([]byte(fmt.Sprintf("%d", a.getLeaderID())))
}

func (a *ChangeAgent) handleRaftInfo(resp http.ResponseWriter, req *http.Request) {
  state := RaftState{
    State: a.GetRaftState().String(),
    Leader: a.getLeaderID(),
  }

  body, _ := json.Marshal(&state)

  resp.Header().Set("Content-Type", JSON)
  resp.Write(body)
}

func (a *ChangeAgent) getLeaderID() uint64 {
  if a.GetRaftState() == raft.Leader {
    return a.raft.MyID()
  }
  return a.raft.GetLeaderID()
}

func handleStackCall(resp http.ResponseWriter, req *http.Request) {
  stackBufLen := 64
  for {
    stackBuf := make([]byte, stackBufLen)
    stackLen := runtime.Stack(stackBuf, true)
    if stackLen == len(stackBuf) {
      // Must be truncated
      stackBufLen *= 2
    } else {
      resp.Header().Set("Content-Type", PlainText)
      resp.Write(stackBuf)
      return
    }
  }
}
