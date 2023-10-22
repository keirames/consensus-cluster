package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type Service struct{}

func (s *Service) Hello(request string, reply *string) error {
	fmt.Println(request)
	return nil
}

func (s *Service) HealthCheck(request string, reply *string) error {
	fmt.Println("healthy check from leader")
	return nil
}

func startRPCServer(address string) {
	rpc.RegisterName("Service", new(Service))

	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("cannot listen:", address, err)
	}

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Fatal("fail to accept conn", err)
			}

			go rpc.ServeConn(conn)
		}
	}()

	ticker := time.NewTicker(time.Second * 2)
	go func() {
		for {
			<-ticker.C

			hosts := []string{"localhost:3000", "localhost:3001", "localhost:3002"}
			var validHosts []string
			for _, h := range hosts {
				if h != address {
					validHosts = append(validHosts, h)
				}
			}

			client, err := rpc.Dial("tcp", validHosts[0])
			if err != nil {
				fmt.Println("fail to dial host", err)
				continue
			}

			err = client.Call("Service.Hello", "from "+validHosts[0], nil)
			if err != nil {
				fmt.Println("client call fail", err)
				continue
			}
		}
	}()

	fmt.Println("listening on port:", address)
}

func main() {
	// id, err := strconv.ParseInt(os.Getenv("Id"), 10, 64)
	// if err != nil {
	// 	log.Fatal("cannot parse env 'Id'")
	// }

	// cm := consensusmodule.New(&consensusmodule.Opts{
	// 	ID: int(id),
	// })
	// cm.RunElectionTimer()

	var wg sync.WaitGroup
	wg.Add(1)

	go startRPCServer("localhost:3000")
	go startRPCServer("localhost:3001")
	go startRPCServer("localhost:3002")

	wg.Wait()
}
