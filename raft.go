package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"
)

type Server struct{}

// ConsensusModule (CM) implements a single node of Raft consensus.
type ConsensusModule struct {
	// mu protects concurrent access to a CM.
	mu sync.Mutex

	// id is the server ID of this CM.
	id int

	// peerIds lists the IDs of our peers in the cluster.
	peerIds []int

	// server is the server containing this CM. It's used to issue RPC calls
	// to peers.
	server *Server

	// persistent Raft state on all servers.
	currentTerm int
}

func New() *ConsensusModule {
	cm := new(ConsensusModule)
	return cm
}

func (cm *ConsensusModule) log(format string, args ...any) {
	format = fmt.Sprintf("[%d] ", cm.id) + format
	log.Printf(format, args...)
}

func (cm *ConsensusModule) runElectionTimer() {
	timeoutDuration := cm.genRandElectionTimeout()
	termStarted := cm.currentTerm

	cm.log("election timer started (%v), term=%d", timeoutDuration, termStarted)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C
		fmt.Println("ticker end")
	}
}

// gen a pseudo-random election timeout duration.
// 150ms - 300ms
func (cm *ConsensusModule) genRandElectionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
