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

func (s *Server) Call(...any) error {
	return nil
}

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

type RequestVoteArgs struct {
	Term        int
	CandidateID int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

func (cm *ConsensusModule) startElection() {

	cm.state = Candidate
	cm.currentTerm += 1
	cm.log("becomes Candidate (currentTerm=%d); log=%v", cm.currentTerm)

	// Prevent read new term when calling for vote.
	savedCurrentTerm := cm.currentTerm

	// Vote for self
	votesReceived := 1

	// Send RequestVote RPCs to all other servers concurrently.
	for _, peerID := range cm.peerIDs {
		go func(peerID int) {
			args := RequestVoteArgs{
				Term:        savedCurrentTerm,
				CandidateID: cm.id,
			}
			var reply RequestVoteReply

			cm.log("sending RequestVote to %d: %v", peerID, args)
			err := cm.server.Call(args, &reply)
			if err != nil {
				fmt.Println(err)
				return
			}
			cm.log("received RequestVoteReply %+v", reply)

			cm.mu.Lock()
			defer cm.mu.Unlock()

			// Drop reply
			if cm.state != Candidate {
				cm.log("while waiting for reply state=%v", cm.state)
				return
			}

			if reply.Term > savedCurrentTerm {
				cm.log("current term out of date - RequestVoteReply-Term = %v", reply.Term)
				cm.becomeFollower(reply.Term)
				return
			}

			if reply.Term == savedCurrentTerm && reply.VoteGranted {
				votesReceived += 1
				if votesReceived*2 > len(cm.peerIDs)+1 {
					// won the election!
					cm.log("wins election with %d votes", votesReceived)
					cm.becomeLeader()
					return
				}
			}
		}(peerID)
	}

	// run another election in case this election is not successful
	go cm.RunElectionTimer()
}

func (cm *ConsensusModule) becomeLeader() {
	cm.state = Leader
	cm.log("become a Leader; term=%d", cm.currentTerm)

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		// Send periodic heartbeats as long as still leader.
		for {
			cm.leaderSendHeartbeats()

			<-ticker.C

			cm.mu.Lock()
			if cm.state != Leader {
				cm.mu.Unlock()
				return
			}

			cm.mu.Unlock()
		}
	}()
}

type AppendEntriesArgs struct {
	Term     int
	LeaderID int
}

func (cm *ConsensusModule) leaderSendHeartbeats() {
	cm.mu.Lock()
	savedCurrentTerm := cm.currentTerm
	cm.mu.Unlock()

	for _, peerID := range cm.peerIDs {
		args := AppendEntriesArgs{
			Term:     savedCurrentTerm,
			LeaderID: cm.id,
		}

		go func(peerID int) {
			cm.log("sending AppendEntries to %v: ni=%d, args=%+v", peerID, 0, args)
			var reply AppendEntriesArgs

			err := cm.server.Call(peerID, "Service.AppendEntries", args, &reply)
			if err != nil {
				return
			}

			cm.mu.Lock()
			defer cm.mu.Unlock()

			if reply.Term > savedCurrentTerm {
				// out of sync, no longer a leader.
				cm.log("term out of date in heartbeat reply")
				cm.becomeFollower(reply.Term)
				return
			}
		}(peerID)
	}
}

func (cm *ConsensusModule) becomeFollower(term int) {
	cm.log("becomes Follower with term=%d", term)
	cm.state = Follower
	cm.currentTerm = term

	go cm.RunElectionTimer()
}

func (cm *ConsensusModule) RunElectionTimer() {
	timeoutDuration := cm.genRandElectionTimeout()
	termStarted := cm.currentTerm

	cm.log("election timer started (%v), term=%d", timeoutDuration, termStarted)

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C

		cm.mu.Lock()
		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			cm.startElection()
			cm.mu.Unlock()
			return
		}
	}
}

// gen a pseudo-random election timeout duration.
// 150ms - 300ms
func (cm *ConsensusModule) genRandElectionTimeout() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}
