package stress

import (
  "encoding/json"
  "fmt"
  "io/ioutil"
  "net/http"
  "path"
  "strings"
  "time"
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

const (
  BasePort = 13000
  DefaultWait = 30
  BatchSize = 10
)

type WriteResponse struct {
  Id uint64 `json:"_id"`
}

type CollectionResponse struct {
  Id string `json:"_id"`
}

var _ = Describe("Cluster Test", func() {
  var ports []int

  It("Cluster Test", func() {
    ports = make([]int, 3)
    ports[0] = BasePort
    ports[1] = BasePort + 1
    ports[2] = BasePort + 2

    // Launch three server processes and make sure that there is only one leader between them
    server1, err := launchAgent(1, ports[0], path.Join(dataDir, "data1"))
    Expect(err).Should(Succeed())
    server2, err := launchAgent(2, ports[1], path.Join(dataDir, "data2"))
    Expect(err).Should(Succeed())
    server3, err := launchAgent(3, ports[2], path.Join(dataDir, "data3"))
    Expect(err).Should(Succeed())

    err = waitForLeader(ports, DefaultWait)
    Expect(err).Should(Succeed())

    // Send a bunch of records to each server and make sure that they get replicated to the lot
    verifyBatch(ports[0], ports, "Full Cluster 1")
    verifyBatch(ports[1], ports, "Full Cluster 2")
    verifyBatch(ports[2], ports, "Full Cluster 3")

    _, collectionId := populateCollection(ports[0], "Tenant1-1", "Collection1-1")
    verifyCollection(ports[0], collectionId)
    verifyCollection(ports[1], collectionId)
    verifyCollection(ports[2], collectionId)

    // Kill each server one at a time and repeat that test of sending stuff and seeing it replicated
    fmt.Fprintf(GinkgoWriter, "Killing server at port %d\n", ports[0])
    killAgent(server1)
    newPorts := make([]int, 2)
    newPorts[0] = ports[1]
    newPorts[1] = ports[2]

    err = waitForLeader(newPorts, DefaultWait)
    Expect(err).Should(Succeed())

    verifyBatch(newPorts[0], newPorts, "Partial Cluster 1")
    verifyBatch(newPorts[1], newPorts, "Partial Cluster 2")

    // Restart the server and test again
    server1, err = launchAgent(1, ports[0], path.Join(dataDir, "data1"))
    Expect(err).Should(Succeed())
    err = waitForLeader(ports, DefaultWait)
    Expect(err).Should(Succeed())

    verifyBatch(ports[0], ports, "Full Cluster 2, 1")
    verifyBatch(ports[1], ports, "Full Cluster 2, 2")
    verifyBatch(ports[2], ports, "Full Cluster 2, 3")

    // Kill another server and re-test again
    fmt.Fprintf(GinkgoWriter, "Killing server at port %d\n", ports[1])
    killAgent(server2)
    newPorts[0] = ports[0]
    newPorts[1] = ports[2]

    err = waitForLeader(newPorts, DefaultWait)
    Expect(err).Should(Succeed())

    verifyBatch(newPorts[0], newPorts, "Partial Cluster 2, 1")
    verifyBatch(newPorts[1], newPorts, "Partial Cluster 2, 2")

    server2, err = launchAgent(2, ports[1], path.Join(dataDir, "data2"))
    Expect(err).Should(Succeed())
    err = waitForLeader(ports, DefaultWait)
    Expect(err).Should(Succeed())

    verifyBatch(ports[0], ports, "Full Cluster 3, 1")
    verifyBatch(ports[1], ports, "Full Cluster 3, 2")
    verifyBatch(ports[2], ports, "Full Cluster 3, 3")

    // And now the third server
    fmt.Fprintf(GinkgoWriter, "Killing server at port %d\n", ports[2])
    killAgent(server3)
    newPorts[0] = ports[0]
    newPorts[1] = ports[1]

    err = waitForLeader(newPorts, DefaultWait)
    Expect(err).Should(Succeed())

    verifyBatch(newPorts[0], newPorts, "Partial Cluster 3, 1")
    verifyBatch(newPorts[1], newPorts, "Partial Cluster 3, 2")

    server3, err = launchAgent(3, ports[2], path.Join(dataDir, "data3"))
    Expect(err).Should(Succeed())
    err = waitForLeader(ports, DefaultWait)
    Expect(err).Should(Succeed())

    verifyBatch(ports[0], ports, "Full Cluster 4, 1")
    verifyBatch(ports[1], ports, "Full Cluster 4, 2")
    verifyBatch(ports[2], ports, "Full Cluster 4, 3")

    // Now kill and restart everyone one last time

    killAgent(server1)
    killAgent(server2)
    killAgent(server3)

    time.Sleep(time.Second)

    server1, err = launchAgent(1, ports[0], path.Join(dataDir, "data1"))
    Expect(err).Should(Succeed())
    server2, err = launchAgent(2, ports[1], path.Join(dataDir, "data2"))
    Expect(err).Should(Succeed())
    server3, err = launchAgent(3, ports[2], path.Join(dataDir, "data3"))
    Expect(err).Should(Succeed())

    err = waitForLeader(ports, DefaultWait)
    Expect(err).Should(Succeed())

    verifyBatch(ports[0], ports, "Full Cluster 5, 1")
    verifyBatch(ports[1], ports, "Full Cluster 5, 2")
    verifyBatch(ports[2], ports, "Full Cluster 5, 3")
  })
})

func verifyBatch(writePort int, allPorts []int, anno string) {
  firstChange, lastChange := sendBatch(writePort, BatchSize, anno)
  fmt.Fprintf(GinkgoWriter, "Sent changes from %d to %d to %d\n", firstChange, lastChange, writePort)
  for _, p := range(allPorts) {
    checkChanges(p, firstChange, lastChange, DefaultWait)
  }
}

func sendBatch(writePort int, count int, anno string) (uint64, uint64) {
  uri := fmt.Sprintf("http://localhost:%d/changes", writePort)
  first := true
  var firstChange uint64
  var lastChange uint64

  for i := 0; i < count; i++ {
    msg := fmt.Sprintf("{\"data\":{\"test\":\"%s\",\"sequence\":%d}}",
      anno, i)
    resp, err := http.Post(uri, "application/json", strings.NewReader(msg))
    Expect(err).Should(Succeed())
    bodyBytes, err := ioutil.ReadAll(resp.Body)
    Expect(err).Should(Succeed())

    fmt.Fprintf(GinkgoWriter, "Response body: %s\n", string(bodyBytes))

    Expect(resp.StatusCode).Should(BeEquivalentTo(200))

    var writeResp WriteResponse
    err = json.Unmarshal(bodyBytes, &writeResp)
    Expect(err).Should(Succeed())

    if first {
      firstChange = writeResp.Id
      first = false
    }
    lastChange = writeResp.Id
  }

  return firstChange, lastChange
}

func fetchChanges(readPort int, since uint64) []uint64 {
  uri := fmt.Sprintf("http://localhost:%d/changes?since=%d&limit=%d", readPort, since, BatchSize + 10)

  resp, err := http.Get(uri)
  Expect(err).Should(Succeed())
  Expect(resp.StatusCode).Should(BeEquivalentTo(200))
  body, err := ioutil.ReadAll(resp.Body)
  Expect(err).Should(Succeed())

  var entries []WriteResponse
  err = json.Unmarshal(body, &entries)
  Expect(err).Should(Succeed())

  ret := make([]uint64, len(entries))
  for i, entry := range(entries) {
    ret[i] = entry.Id
  }

  return ret
}

func checkChanges(readPort int, firstChange, lastChange uint64, maxWait int) {
  expectedChanges := int(lastChange - firstChange) + 1
  for i := 0; i < maxWait; i++ {
    changes := fetchChanges(readPort, firstChange - 1)
    if len(changes) == expectedChanges &&
       changes[0] == firstChange &&
       changes[len(changes) - 1] == lastChange {
      return
    }
    fmt.Fprintf(GinkgoWriter, "Incomplete set of %d changes from port %d: %v\n",
      len(changes), readPort, changes)
    time.Sleep(time.Second)
  }

  Expect(true).Should(BeFalse())
}

func populateCollection(writePort int, tenantName, collectionName string) (string, string) {
  uri := fmt.Sprintf("http://localhost:%d/tenants", writePort)
  msg := fmt.Sprintf("name=%s", tenantName)

  resp, err := http.Post(uri, "application/x-www-form-urlencoded", strings.NewReader(msg))
  Expect(err).Should(Succeed())
  Expect(resp.StatusCode).Should(BeEquivalentTo(200))
  body, err := ioutil.ReadAll(resp.Body)
  Expect(err).Should(Succeed())

  var tenant CollectionResponse
  err = json.Unmarshal(body, &tenant)
  Expect(err).Should(Succeed())
  fmt.Fprintf(GinkgoWriter, "Created tenant %s\n", tenant.Id)

  uri = fmt.Sprintf("http://localhost:%d/tenants/%s/collections", writePort, tenantName)
  msg = fmt.Sprintf("name=%s", collectionName)

  resp, err = http.Post(uri, "application/x-www-form-urlencoded", strings.NewReader(msg))
  Expect(err).Should(Succeed())
  Expect(resp.StatusCode).Should(BeEquivalentTo(200))
  body, err = ioutil.ReadAll(resp.Body)
  Expect(err).Should(Succeed())

  var collection CollectionResponse
  err = json.Unmarshal(body, &collection)
  Expect(err).Should(Succeed())
  fmt.Fprintf(GinkgoWriter, "Created collection %s\n", collection.Id)

  uri = fmt.Sprintf("http://localhost:%d/collections/%s/keys", writePort, collection.Id)
  for i := 0; i < BatchSize; i++ {
    msg = fmt.Sprintf("{\"key\":\"%d\",\"data\":{\"collection\":\"%s\",\"sequence\":%d}}",
      i, collectionName, i)
    resp, err = http.Post(uri, "application/json", strings.NewReader(msg))
    Expect(err).Should(Succeed())
    Expect(resp.StatusCode).Should(BeEquivalentTo(200))
    body, err = ioutil.ReadAll(resp.Body)
    Expect(err).Should(Succeed())
  }

  return tenant.Id, collection.Id
}

func verifyCollection(readPort int, collectionId string) {
  uri := fmt.Sprintf("http://localhost:%d/collections/%s/keys", readPort, collectionId)

  resp, err := http.Get(uri)
  Expect(err).Should(Succeed())
  body, err := ioutil.ReadAll(resp.Body)
  Expect(err).Should(Succeed())
  fmt.Fprintf(GinkgoWriter, "Collection entries: %s\n", string(body))
  Expect(resp.StatusCode).Should(BeEquivalentTo(200))

  var entries []WriteResponse
  err = json.Unmarshal(body, &entries)
  Expect(err).Should(Succeed())
  Expect(len(entries)).Should(Equal(BatchSize))
}