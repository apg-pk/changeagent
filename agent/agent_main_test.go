package main

import (
  "flag"
  "fmt"
  "os"
  "net"
  "net/http"
  "path"
  "testing"
  "time"
  "revision.aeip.apigee.net/greg/changeagent/discovery"
  "revision.aeip.apigee.net/greg/changeagent/raft"

  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

const (
  DataDir = "./agenttestdata"
  PreserveDatabases = false
  DebugMode = false
)

var testListener []*net.TCPListener
var testAgents []*ChangeAgent
var leaderIndex int

func TestAgent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Agent Suite")
}

var _ = BeforeSuite(func() {
  os.MkdirAll(DataDir, 0777)
  flag.Set("logtostderr", "true")
  if DebugMode {
    flag.Set("v", "5")
  }
  flag.Parse()

  // Create three TCP listeners -- we'll use them for a cluster
  anyPort := &net.TCPAddr{}
  var addrs []string
  for li := 0; li < 3; li++ {
    listener, err := net.ListenTCP("tcp4", anyPort)
    if err != nil { panic("Can't listen on a TCP port") }
    _, port, err := net.SplitHostPort(listener.Addr().String())
    if err != nil { panic("Invalid listen address") }
    addrs = append(addrs, fmt.Sprintf("localhost:%s", port))
    testListener = append(testListener, listener)
  }
  disco := discovery.CreateStaticDiscovery(addrs)

  agent1, err := startAgent(1, disco, path.Join(DataDir, "test1"), testListener[0])
  Expect(err).Should(Succeed())
  testAgents = append(testAgents, agent1)

  agent2, err := startAgent(2, disco, path.Join(DataDir, "test2"), testListener[1])
  Expect(err).Should(Succeed())
  testAgents = append(testAgents, agent2)

  agent3, err := startAgent(3, disco, path.Join(DataDir, "test3"), testListener[2])
  Expect(err).Should(Succeed())
  testAgents = append(testAgents, agent3)
})


var _ = AfterSuite(func() {
  for i := range(testAgents) {
    cleanAgent(testAgents[i], testListener[i])
  }
})

var _ = Describe("Agent startup test", func() {
  It("Leader is elected", func() {
    Expect(waitForLeader()).Should(Equal(true))
  })
})

func startAgent(id uint64, disco discovery.Discovery, dir string, listener *net.TCPListener) (*ChangeAgent, error) {
  mux := http.NewServeMux()

  agent, err := StartChangeAgent(id, disco, dir, mux)
  if err != nil { return nil, err }
  go http.Serve(listener, mux)

  return agent, nil
}

func cleanAgent(agent *ChangeAgent, l *net.TCPListener) {
  agent.Close()
  if !PreserveDatabases {
    agent.Delete()
  }
  l.Close()
}

func countRafts() (int, int) {
  var followers, leaders int

  for _, r := range(testAgents) {
    switch r.GetRaftState() {
    case raft.StateFollower:
      followers++
    case raft.StateLeader:
      leaders++
    }
  }

  return followers, leaders
}

func waitForLeader() bool {
  for i := 0; i < 40; i++ {
    _, leaders := countRafts()
    if leaders == 0 {
      time.Sleep(time.Second)
    } else if leaders == 1 {
      return true
    } else {
      panic("More than one leader elected!")
    }
  }
  return false
}

func getLeaderIndex() {
  for i, r := range(testAgents) {
    switch r.GetRaftState() {
    case raft.StateLeader:
      leaderIndex = i
    }
  }
}

func getLeaderURI() string {
  return getListenerURI(leaderIndex)
}

func getFollowerURIs() []string {
  var uris []string
  for i := range(testListener) {
    if i != leaderIndex {
      uris = append(uris, getListenerURI(i))
    }
  }
  return uris
}

func getListenerURI(index int) string {
  _, port, err := net.SplitHostPort(testListener[index].Addr().String())
  if err != nil { panic("Error parsing leader port") }
  return fmt.Sprintf("http://localhost:%s", port)
}
