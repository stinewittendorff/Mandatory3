package main

import (
	"ChittyChat/proto"
	"bufio"
	"context"
	"flag"
	"io"
	"log"
	"os"
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

	// creation of a new client called c
	c := &client{
		name:    *name,
		port:    *ClientPort,
		lamport: 1,
	}

	severConnection, _ := connectToServer()

	defer leaveChat(severConnection, c)

	go JoinChatroom(severConnection, c)

	for {}
}

// The function establishes a connection with the GRPC server, at a specified Ip and port
// If it fails, it logs and error and exits the program
func connectToServer() (proto.ChittyChatClient, error) {
	conn, err := grpc.Dial(*ServerIP+":"+*ServerPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Could not connect to the port%s\n", *ServerPort)
	}
	log.Printf("Connected to the server port %s\n", *ServerPort)

	return proto.NewChittyChatClient(conn), nil
}

// "leaveChat" is used to leave chatrooms. If it fails, it logs and error and exits the program
func leaveChat(serverConn proto.ChittyChatClient, client *client) {
	joined = false
	_, err := serverConn.Leave(context.Background(), &proto.Leave{
		Timestamp: client.lamport,
		Name:      client.name,
	})
	if err != nil {
		log.Fatalf("Could not leave the chatroom %v\n", err)
	}
	log.Printf("Left the chatroom")
}

// "JoinChatroom" is used to join chatrooms and make interactions with the server while a user is active in the chatroom
func JoinChatroom(serverConn proto.ChittyChatClient, client *client) {

	//starts by setting up a scanner to read the input that the user gives
	scanner := bufio.NewScanner(os.Stdin)
	log.Printf("entering loop")

	for scanner.Scan() {
		input := scanner.Text()

		//Checks if the message is too long
		if len(input) > 128 {
			log.Printf("Your message is too long, try again with less than 128 Characters")
			continue
		}

		//Checks if the user wants to join the chatroom if not already in
		if !joined {
			if input == "/join" {
				messageStream, err := serverConn.Join(context.Background(), &proto.Join{
					Timestamp: client.lamport,
					Name:      client.name,
				})
				if err != nil {
					log.Fatalf("Could not join the chatroom %v\n", err)
				}

				joined = true
				go receiveMessages(messageStream, client)
			} else {
				log.Printf("chatroom is not joined, use /join to join the chatroom")
			}
			continue
		}
		client.incrementLamport()
		

		//Checks if the user wants to leave the chatroom
		if input == "/leave" {
			leaveChat(serverConn, client)
			continue
		}

		//Sends the message to the server
		_, err := serverConn.Broadcast(context.Background(), &proto.Chatmessage{
			Timestamp: client.lamport,
			Name:      client.name,
			Message:   input,
		})
		if err != nil {
			log.Fatalf("Could not send message %v\n", err)
		}

	}
}

// "receiveMessages" is used to continuously receive messages from a server
func receiveMessages(messageStream proto.ChittyChat_JoinClient, client *client) {
	for joined {
		message, err := messageStream.Recv()
	
		// starts by checking if the server has closed the stream
		if err == io.EOF {
			log.Printf("The server has closed the stream\n")
			return
		}
		if err != nil {
			log.Fatalf("Could not receive message %v\n", err)
		}

		// Updates the users lamport timestamp based on a received messages timestamp
		if message.Timestamp > int64(client.lamport) {
			client.lamport = message.Timestamp + 1
		} else {
			client.incrementLamport()
		}

		//And lastly logs the received message with its timestamp
		log.Printf("Received message \"%s\" at time %d\n", message.Message, client.lamport)

	}
}
