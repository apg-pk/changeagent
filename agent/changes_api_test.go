package main

import (
  "fmt"
  "strings"
  "time"
  "encoding/json"
  "net/http"
  "io/ioutil"
  . "github.com/onsi/ginkgo"
  . "github.com/onsi/gomega"
)

const (
  jsonContent = "application/json"
)

var lastNewChange uint64
var numPosts int

var _ = Describe("Changes API Test", func() {

  It("POST new change", func() {
    request := "{\"hello\": \"world!\", \"foo\": 123}"
    resp := postChange(request)

    entry, err := unmarshalJSON(strings.NewReader(resp))
    Expect(err).Should(Succeed())
    expected := fmt.Sprintf("{\"_id\":%d}", entry.Index)
    Expect(resp).Should(MatchJSON(expected))

    lastNewChange = entry.Index

    // Upon return, change should immediately be represented at the leader
    respExpected :=
      fmt.Sprintf("[{\"_id\":%d,\"_ts\":[0-9]+,\"data\":{\"hello\":\"world!\",\"foo\":123}}]", lastNewChange)
    peerChanges := getChanges(lastNewChange - 1, 100, 0, nil)
    Expect(peerChanges).Should(MatchRegexp(respExpected))

    peerChanges = getChanges(lastNewChange - 1, 100, 10, nil)
    Expect(peerChanges).Should(MatchRegexp(respExpected))
    fmt.Fprintf(GinkgoWriter, "Get changes peer \"%s\"\n", peerChanges)
    Expect(err).Should(Succeed())

    url := fmt.Sprintf("%s/changes/%d", listenURI, lastNewChange)
    gr, err := http.Get(url)
    Expect(err).Should(Succeed())
    defer gr.Body.Close()
    Expect(gr.StatusCode).Should(Equal(200))

    change, err := unmarshalJSON(gr.Body)
    Expect(err).Should(Succeed())
    Expect(gr.StatusCode).Should(Equal(200))
    Expect(change.Index).Should(Equal(lastNewChange))
  })

  It("POST empty change", func() {
    peerChanges := getChanges(lastNewChange, 100, 0, nil)
    Expect(strings.TrimSpace(string(peerChanges))).Should(Equal("[]"))
  })

  It("POST with tag", func() {
    changes := postChanges(1, []string{"tagone"})
    lastNewChange = changes[0]

    // Response comes back just this one with the tags in it
    respExpected :=
      fmt.Sprintf("[{\"_id\":%d,\"_ts\":[0-9]+,\"tags\":\\[\"tagone\"\\],\"data\":{\"hello\":\"world!\",\"count\":1}}]", lastNewChange)
    tagChanges := getChanges(lastNewChange - 1, 1, 0, nil)
    Expect(tagChanges).Should(MatchRegexp(respExpected))

    // Requesting all changes should give us only the one with the tags on it
    tagChanges = getChanges(0, 100, 0, []string{"tagone"})
    Expect(tagChanges).Should(MatchRegexp(respExpected))

    // If we look for non-matching tags when we should get back nothing
    tagChanges = getChanges(0, 100, 0, []string{"tagone", "tagtwo"})
    Expect(tagChanges).Should(BeEquivalentTo("[]"))
  })

  It("POST with two tags", func() {
    changes := postChanges(1, []string{"tagone", "tagtwo"})
    lastNewChange = changes[0]

    // Response comes back just this one with the tags in it
    respExpected :=
      fmt.Sprintf("[{\"_id\":%d,\"_ts\":[0-9]+,\"tags\":\\[\"tagone\",\"tagtwo\"\\],\"data\":{\"hello\":\"world!\",\"count\":1}}]", lastNewChange)
    tagChanges := getChanges(lastNewChange - 1, 1, 0, nil)
    Expect(tagChanges).Should(MatchRegexp(respExpected))

    // Requesting all changes should give us only the one with the tags on it
    tagChanges = getChanges(0, 100, 0, []string{"tagone", "tagtwo"})
    Expect(tagChanges).Should(MatchRegexp(respExpected))

    // If we look for non-matching tags when we should get back nothing
    tagChanges = getChanges(0, 100, 0, []string{"tagthree"})
    Expect(tagChanges).Should(BeEquivalentTo("[]"))
  })

  It("Post and retrieve multiple", func() {
    changes := postChanges(2, nil)

    templ :=
      "[{\"_id\":%d,\"_ts\":[0-9]+,\"data\":{\"hello\":\"world!\",\"count\":1}}," +
      "{\"_id\":%d,\"_ts\":[0-9]+,\"data\":{\"hello\":\"world!\",\"count\":2}}]"
    respExpected := fmt.Sprintf(templ, lastNewChange - 1, lastNewChange)
    peerChanges := getChanges(changes[0] - 1, 100, 0, nil)

    Expect(peerChanges).Should(MatchRegexp(respExpected))
  })

  It("Retrieve all", func() {
    changes := postChanges(3, nil)

    respBody := getChanges(0, 100, 0, nil)
    var results []JSONData
    err := json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(BeNumerically(">=", 3))

    // Test various permutations of offset and limit now.
    respBody = getChanges(changes[0] - 1, 1, 0, nil)
    err = json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(Equal(1))
    Expect(results[0].ID).Should(Equal(changes[0]))

    respBody = getChanges(changes[0] - 1, 2, 0, nil)
    err = json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(Equal(2))
    Expect(results[0].ID).Should(Equal(changes[0]))
    Expect(results[1].ID).Should(Equal(changes[1]))
  })

  It("Blocking retrieval", func() {
    respBody := getChanges(lastNewChange, 100, 0, nil)
    var results []JSONData
    err := json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(Equal(0))

    ch := make(chan uint64, 1)

    go func() {
      ch <- 0
      newResp := getChanges(lastNewChange, 100, 5, nil)
      var newResults []JSONData
      json.Unmarshal(newResp, &newResults)
      if len(newResults) == 1 {
        ch <- newResults[0].ID
      } else {
        ch <- 0
      }
    }()

    <- ch
    time.Sleep(500 * time.Millisecond)
    request := "{\"hello\": \"world!\", \"foo\": 9999}"
    resp := postChange(request)

    var postResult JSONData
    err = json.Unmarshal([]byte(resp), &postResult)
    Expect(err).Should(Succeed())
    lastNewChange = postResult.ID

    Eventually(ch).Should(Receive(Equal(postResult.ID)))
  })

  It("Blocking retrieval with tag", func() {
    respBody := getChanges(lastNewChange, 100, 0, nil)
    var results []JSONData
    err := json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(Equal(0))

    ch := make(chan uint64, 1)

    go func() {
      ch <- 0
      newResp := getChanges(lastNewChange, 100, 5, []string{"block"})
      var newResults []JSONData
      json.Unmarshal(newResp, &newResults)
      if len(newResults) == 1 {
        ch <- newResults[0].ID
      } else {
        ch <- 0
      }
    }()

    <- ch
    time.Sleep(500 * time.Millisecond)
    request := "{\"tags\":[\"block\"],\"data\":{\"hello\": \"world!\", \"foo\": 9999}}"
    resp := postChange(request)

    var postResult JSONData
    err = json.Unmarshal([]byte(resp), &postResult)
    Expect(err).Should(Succeed())
    lastNewChange = postResult.ID

    Eventually(ch).Should(Receive(Equal(postResult.ID)))
  })

  It("Blocking retrieval after two changes", func() {
    respBody := getChanges(lastNewChange, 100, 0, nil)
    var results []JSONData
    err := json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(Equal(0))

    ch := make(chan uint64, 1)

    go func() {
      ch <- 0
      newResp := getChanges(lastNewChange + 1, 100, 10, nil)
      var newResults []JSONData
      json.Unmarshal(newResp, &newResults)
      if len(newResults) == 1 {
        ch <- newResults[0].ID
      } else {
        ch <- 0
      }
    }()

    <- ch
    time.Sleep(500 * time.Millisecond)
    request := "{\"hello\": \"world!\", \"foo\": 9999}"
    resp := postChange(request)

    var postResult JSONData
    err = json.Unmarshal([]byte(resp), &postResult)
    Expect(err).Should(Succeed())
    lastNewChange = postResult.ID

    Consistently(ch).ShouldNot(Receive())

    time.Sleep(500 * time.Millisecond)
    resp = postChange(request)

    err = json.Unmarshal([]byte(resp), &postResult)
    Expect(err).Should(Succeed())
    lastNewChange = postResult.ID

    Eventually(ch).Should(Receive(Equal(postResult.ID)))
  })

  It("Blocking retrieval with tag two changes", func() {
    respBody := getChanges(lastNewChange, 100, 0, nil)
    var results []JSONData
    err := json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(Equal(0))

    ch := make(chan uint64, 1)

    go func() {
      ch <- 0
      newResp := getChanges(lastNewChange, 100, 10, []string{"block"})
      var newResults []JSONData
      json.Unmarshal(newResp, &newResults)
      if len(newResults) == 1 {
        ch <- newResults[0].ID
      } else {
        ch <- 0
      }
    }()

    <- ch
    time.Sleep(250 * time.Millisecond)
    request := "{\"tags\":[\"keepblocking\"],\"data\":{\"hello\": \"world!\", \"foo\": 9999}}"
    resp := postChange(request)

    var postResult JSONData
    err = json.Unmarshal([]byte(resp), &postResult)
    Expect(err).Should(Succeed())
    lastNewChange = postResult.ID

    Consistently(ch).ShouldNot(Receive())

    time.Sleep(250 * time.Millisecond)
    request = "{\"tags\":[\"block\"],\"data\":{\"hello\": \"world!\", \"foo\": 9999}}"
    resp = postChange(request)

    err = json.Unmarshal([]byte(resp), &postResult)
    Expect(err).Should(Succeed())
    lastNewChange = postResult.ID

    Eventually(ch).Should(Receive(Equal(postResult.ID)))
  })

  It("Blocking retrieval with tag more changes", func() {
    respBody := getChanges(lastNewChange, 100, 0, nil)
    var results []JSONData
    err := json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(Equal(0))

    ch := make(chan uint64, 1)

    go func() {
      ch <- 0
      newResp := getChanges(lastNewChange, 100, 10, []string{"block"})
      var newResults []JSONData
      json.Unmarshal(newResp, &newResults)
      if len(newResults) == 1 {
        ch <- newResults[0].ID
      } else {
        ch <- 0
      }
    }()

    <- ch
    time.Sleep(250 * time.Millisecond)
    var postResult JSONData

    for c := 0; c < 5; c++ {
      request := "{\"tags\":[\"keepblocking\"],\"data\":{\"hello\": \"world!\", \"foo\": 9999}}"
      resp := postChange(request)


      err = json.Unmarshal([]byte(resp), &postResult)
      Expect(err).Should(Succeed())
      lastNewChange = postResult.ID

      Consistently(ch).ShouldNot(Receive())
    }

    time.Sleep(250 * time.Millisecond)
    request := "{\"tags\":[\"block\"],\"data\":{\"hello\": \"world!\", \"foo\": 9999}}"
    resp := postChange(request)

    err = json.Unmarshal([]byte(resp), &postResult)
    Expect(err).Should(Succeed())
    lastNewChange = postResult.ID

    Eventually(ch).Should(Receive(Equal(postResult.ID)))
  })

  It("Blocking retrieval after abnormal change", func() {
    respBody := getChanges(lastNewChange, 100, 0, nil)
    var results []JSONData
    err := json.Unmarshal(respBody, &results)
    Expect(err).Should(Succeed())
    Expect(len(results)).Should(Equal(0))

    ch := make(chan uint64, 1)

    go func() {
      newResp := getChanges(lastNewChange, 100, 5, nil)
      var newResults []JSONData
      json.Unmarshal(newResp, &newResults)
      if len(newResults) == 1 {
        ch <- newResults[0].ID
      } else {
        ch <- 0
      }
    }()

    request := "{\"hello\": \"world!\", \"foo\": 9999}"
    time.Sleep(500 * time.Millisecond)
    resp := postChange(request)

    var postResult JSONData
    err = json.Unmarshal([]byte(resp), &postResult)
    Expect(err).Should(Succeed())
    lastNewChange = postResult.ID

    Eventually(ch).Should(Receive(Equal(postResult.ID)))
  })
})

func postChanges(count int, tags []string) []uint64 {
  changes := make([]uint64, count)
  for n := 0; n < count; n++ {
    var request string
    if len(tags) > 0 {
      request =
        fmt.Sprintf("{\"data\":{\"hello\":\"world!\",\"count\":%d}, \"tags\": [", n + 1)
      for i, tag := range(tags) {
        if i > 0 {
          request += ", "
        }
        request += "\"" + tag + "\""
      }
      request += "]}"
    } else {
      request =
        fmt.Sprintf("{\"data\":{\"hello\":\"world!\",\"count\":%d}}", n + 1)
    }
    resp := postChange(request)

    lastNewChange++
    expected := fmt.Sprintf("{\"_id\":%d}", lastNewChange)
    Expect(resp).Should(MatchJSON(expected))
    changes[n] = lastNewChange
  }
  return changes
}

func postChange(request string) string {
  uri := listenURI + "/changes"

  fmt.Fprintf(GinkgoWriter, "POST change: %s\n", request)
  pr, err := http.Post(uri, jsonContent, strings.NewReader(request))
  Expect(err).Should(Succeed())
  defer pr.Body.Close()
  body, err := ioutil.ReadAll(pr.Body)
  fmt.Fprintf(GinkgoWriter, "Response: %s\n", string(body))
  Expect(pr.StatusCode).Should(Equal(200))
  numPosts++

  Expect(err).Should(Succeed())
  resp := string(body)

  fmt.Fprintf(GinkgoWriter, "Got POST response %s\n", resp)
  return resp
}

func getChanges(since uint64, limit int, block int, tags []string) []byte {
  url := fmt.Sprintf("%s/changes?since=%d&limit=%d&block=%d",
    listenURI, since, limit, block)
  for _, tag := range(tags) {
    url += fmt.Sprintf("&tag=%s", tag)
  }
  gr, err := http.Get(url)
  Expect(err).Should(Succeed())
  defer gr.Body.Close()
  Expect(gr.StatusCode).Should(Equal(200))

  respBody, err := ioutil.ReadAll(gr.Body)
  Expect(err).Should(Succeed())
  resp := string(respBody)
  fmt.Fprintf(GinkgoWriter, "Got GET response %s\n", resp)
  return respBody
}
