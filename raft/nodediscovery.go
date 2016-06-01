package raft

import (
	"time"

	"github.com/golang/glog"
	"revision.aeip.apigee.net/greg/changeagent/communication"
	"revision.aeip.apigee.net/greg/changeagent/discovery"
)

const (
	DiscoveryInterval   = 5 * time.Second
	RediscoveryInterval = time.Hour
)

/*
 * This set of functions finds out about new nodes from the discovery service and reports
 * on them to its parent. That way we have a table that can let us map a node ID to
 * an address.
 */

type nodeDiscovery struct {
	discovered map[uint64]string
	disco      discovery.Discovery
	comm       communication.Communication
	raft       *Service
	stopChan   chan bool
}

func startNodeDiscovery(disco discovery.Discovery,
	comm communication.Communication,
	raft *Service) *nodeDiscovery {
	nd := nodeDiscovery{
		discovered: make(map[uint64]string),
		disco:      disco,
		comm:       comm,
		raft:       raft,
		stopChan:   make(chan bool, 1),
	}
	go nd.nodeDiscoveryLoop()
	return &nd
}

func (d *nodeDiscovery) stop() {
	d.stopChan <- true
}

func (d *nodeDiscovery) nodeDiscoveryLoop() {
	discoChanges := d.disco.Watch()
	discoveryDone := d.doDiscovery()

	delay := time.NewTimer(RediscoveryInterval)
	for {
		if !discoveryDone {
			delay.Reset(DiscoveryInterval)
		}

		select {
		case <-delay.C:
			discoveryDone = d.doDiscovery()
		case <-discoChanges:
			discoveryDone = d.doDiscovery()
		case <-d.stopChan:
			delay.Stop()
			return
		}
	}
}

func (d *nodeDiscovery) doDiscovery() bool {
	nodes := d.disco.GetCurrentConfig().GetUniqueNodes()
	success := true

	for _, node := range nodes {
		id, err := d.comm.Discover(node)
		if err != nil {
			glog.V(2).Infof("Error discovering node at %s: %v", node, err)
			success = false
		}

		if d.discovered[id] != node {
			glog.V(2).Infof("Discovery: node at %s is %d", node, id)
			d.discovered[id] = node
			d.raft.addDiscoveredNode(id, node)
		}
	}
	return success
}
