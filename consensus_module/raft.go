package consensusmodule

import (
	"fmt"
	"log"
	"math/rand"
	"net/rpc"
	"sync"
	"time"
)

type CMState int

const (
	Follower CMState = iota
	Candidate
	Leader
	Dead
)

func (s CMState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Dead:
		return "Dead"
	default:
		panic("unreachable")
	}
}

type Server struct{}

func (s *Server) sendHealthCheckSignalToAllPeer(peerIDs []int) {
	for _, id := range peerIDs {
		client, err := rpc.Dial("tcp", "localhost:300"+string(id))
		if err != nil {
			fmt.Println("send health check to peer fail", err)
			continue
		}

		err = client.Call("Service.HealthCheck", "", nil)
		if err != nil {
			fmt.Println("fail to send health check signal to peer", err)
			continue
		}
	}
}

// ConsensusModule (CM) implements a single node of Raft consensus.
type ConsensusModule struct {
	// mu protects concurrent access to a CM.
	mu sync.Mutex

	// id is the server ID of this CM.
	id int

	// peerIDs lists the IDs of our peers in the cluster.
	peerIDs []int

	// server is the server containing this CM. It's used to issue RPC calls
	// to peers.
	server *Server

	// persistent Raft state on all servers.
	currentTerm int

	state CMState

	// Reset when receive a ping from Leader
	electionResetEvent time.Time
}

type Opts struct {
	ID int
}

func New(opts *Opts) *ConsensusModule {
	cm := new(ConsensusModule)
	cm.electionResetEvent = time.Now()
	cm.peerIDs = []int{1, 2, 3}
	cm.id = opts.ID

	return cm
}

func (cm *ConsensusModule) log(format string, args ...any) {
	format = fmt.Sprintf("[%d] ", cm.id) + format
	log.Printf(format, args...)
}

func (cm *ConsensusModule) sendHealthCheckSignal() {
	cm.mu.Lock()
	curState := cm.state
	cm.mu.Unlock()
	if curState == Leader {
		// remove self
		var peerIDs []int
		for _, id := range peerIDs {
			if id != cm.id {
				peerIDs = append(peerIDs, id)
			}
		}
		cm.server.sendHealthCheckSignalToAllPeer(peerIDs)
	}
}

func (cm *ConsensusModule) startElection() {
	cm.state = Candidate
	cm.currentTerm += 1
	cm.log("becomes Candidate (currentTerm=%d); log=%v", cm.currentTerm)

	// Send RequestVote RPCs to all other servers concurrently.
	for _, peerID := range cm.peerIDs {
		go func(peerID int) {}(peerID)
	}
}

func (cm *ConsensusModule) RunElectionTimer() {
	timeoutDuration := cm.genRandElectionTimeout()
	termStarted := cm.currentTerm

	cm.log("election timer started (%v), term=%d", timeoutDuration, termStarted)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C

		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			fmt.Println("start election!!!")
			return
		}
	}
}

// gen a pseudo-random election timeout duration.
// 150ms - 300ms
func (cm *ConsensusModule) genRandElectionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
