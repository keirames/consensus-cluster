package main

type Server struct {
}

type ConsensusModule struct {
	// a server ID of this CM
	id int

	// the IDs of our peers in the cluster
	peerIDs []int

	// the server containing this CM. It's used to issue RPC calls to peers.
	server *Server
}

func main() {}
