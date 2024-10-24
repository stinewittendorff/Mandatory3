package main

import (
	"ChittyChat/proto"
	"flag"
	"log"
	"sync/atomic"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type client struct {
	name    string
	port    string
	lamport int64
}

var (
	ClientPort = flag.String("port", "8081", "The server port")
	ServerPort = flag.String("serverport", "8080", "The server port")
	ServerIP   = flag.String("serverip", "localhost", "The server IP")
	name       = flag.String("name", "Unknown", "The name of the client")
	joined     bool
)

// atomic.AddInt64 is to make sure that we don't end in a racecondition
// where several threads try to increment it concurrently
func (c *client) incrementLamport() {
	atomic.AddInt64(&c.lamport, 1)
}

func main() {
	flag.Parse()

	c := &client{
		name:    *name,
		port:    *ClientPort,
		lamport: 1,
	}

	severConnection, _ := connectToServer()
}

func connectToServer() (proto.ChittyChatClient, error) {
	conn, err := grpc.Dial(*ServerIP+*ServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to the port%s\n", *ServerPort)
	}
	log.Printf("Connected to the server port %s\n", *ServerPort)

	return proto.NewChittyChatClient(conn), nil
}

func leaveChat(serverConn proto.ChittyChatClient, client *client) {

}
