/*
 * This file represents the communication from the raft leader to all its
 * peers. We maintain one goroutine per peer, which we use to send
 * all heartbeats and appends, and to keep track of peer state.
 */

package raft

import (
  "time"
  "revision.aeip.apigee.net/greg/changeagent/log"
)

type raftPeer struct {
  id uint64
  r *RaftImpl
  proposals chan uint64
  updateChan chan bool
  changeChan chan<- peerMatchResult
  stopChan chan bool
}

func startPeer(id uint64, r *RaftImpl, changes chan<- peerMatchResult) *raftPeer {
  p := &raftPeer{
    id: id,
    r: r,
    updateChan: make(chan bool),
    changeChan: changes,
    stopChan: make(chan bool),
  }
  go p.peerLoop()
  return p
}

func (p *raftPeer) stop() {
  p.stopChan <- true
}

func (p *raftPeer) propose(ix uint64) {
  p.proposals <- ix
}

func (p *raftPeer) peerLoop() {
  // Next index that we know that we need to send to this peer
  nextIndex, _ := p.r.GetLastIndex()
  // The index, from storage, that we expect to be sending
  desiredIndex := nextIndex

  // Upon election: send initial empty AppendEntries RPCs (heartbeat) to
  // each server; repeat during idle periods to prevent
  // election timeouts (§5.2)
  p.r.sendAppend(p.id, nil)

  timeout := time.NewTimer(HeartbeatTimeout)

  for {
    select {
    case <- timeout.C:
      p.r.sendAppend(p.id, nil)
      timeout.Reset(HeartbeatTimeout)

    case newIndex := <- p.proposals:
      desiredIndex = newIndex
      p.updateChan <- true

    case <- p.updateChan:
      // Requests to update the peer happen via an internal channel.
      // This prevents starvation of stops and timeouts.
      success, err := p.sendUpdates(desiredIndex, nextIndex)
      if err != nil {
        log.Debugf("Error sending to %d: %v", p.id, err)
      } else {
        log.Debugf("Client sent back %v", success)
      }
      if success {
        // If successful: update nextIndex and matchIndex for
        // follower (§5.3)
        log.Debugf("Client %d now up to date with index %d", p.id, desiredIndex)
        nextIndex = desiredIndex
        change := peerMatchResult{
          id: p.id,
          newMatch: desiredIndex,
        }
        p.changeChan <- change

      } else {
        // If AppendEntries fails because of log inconsistency:
        // decrement nextIndex and retry (§5.3)
        if nextIndex > 0 {
          nextIndex--
          p.updateChan <- true
        }
      }

    case <- p.stopChan:
      log.Debugf("Peer %d stopping", p.id)
      return
    }
  }
}

func (p *raftPeer) sendUpdates(desired uint64, next uint64) (bool, error) {
  // If last log index ≥ nextIndex for a follower: send AppendEntries RPC
  // with log entries starting at nextIndex
  entries, err := p.r.stor.GetEntries(next, desired)
  if err != nil {
    log.Debugf("Error sending to peer %d: %v", p.id, err)
    return false, err
  }

  log.Debugf("Sending %d entries between %d and %d to %d",
    len(entries), next, desired, p.id)

  success, err := p.r.sendAppend(p.id, entries)
  return success, err
}