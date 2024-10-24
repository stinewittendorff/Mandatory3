package main

import(
	proto"ChittyChat/proto"
	"context"
	"log"
	"flag"
	"net"
	"sync/atomic"
	"google.golang.org/grpc"
)

type server struct{
	proto.UnimplementedChittyChatServer
	lamport int64
	port string 
	messageChan map[string] chan *proto.Servermessage
}

// flag is commadn line argument in the format of a string that is 
//used to define and manipulate paremters/values in your program
var port = flag.String("port", "8080", "The server port")

func main(){
	flag.Parse()
	log.Printf("Starting server on port %s", *port)

	s := &server{
		lamport: 1,
		port: *port,
		messageChan: make(map[string] chan *proto.Servermessage),
	}
	
	launch(s);
}

// atomic.AddInt64 is to make sure that we don't end in a racecondition
// where several threads try to increment it concurrently 
func (s *server) incrementLamport(){
	atomic.AddInt64(&s.lamport, 1)
}


func (s* server) Broadcast(ctx context.Context, in *proto.Chatmessage) (*proto.Chatmessage, error){
	log.Printf("Client %s sends message \"%s\" at time %d", in.Name, in.Message, in.Timestamp)

	// This updates our servers lamport timestamp (Logical time)
	// If the servers timestamp is less than the message it adjust it to be one unit ahead of the messsage
	// and if the message is equal or less it just adds one to it's own logical clock
	if in.Timestamp > s.lamport{
		s.lamport = in.Timestamp + 1
	} else {
		s.incrementLamport()
	}

	//This defines a Chatmessage objects, where we init the fields
	chat := &proto.Chatmessage{
		Name: in.Name,
		Message: in.Message,
		Timestamp: in.Timestamp,
	}

	// The _ is used to let Go ignore the variable that iterates in the loop also called the blank identifier
	// we use range to get both the key and value, and in this case because of the underscore we ignore the key
	for _, channel := range s.messageChan{
		channel <- &proto.Servermessage{
			Message: in.Name + ": " + in.Message,
			Timestamp: chat.Timestamp,
		}
	}

	return chat, nil	
}

func (s* server) join (in *proto.Join, messageStream proto.ChittyChat_JoinClient) error{
	if in.Timestamp > int64(s.lamport){
		s.lamport = in.Timestamp + 1
	} else {
		s.incrementLamport();
	}

	log.Printf("The client %s has requested to join the chat at lamport time %d", in.Name, in.Timestamp)
	log.Printf("The server's lamport time is %d", s.lamport)

	// This checks if a message channel has already been created for the specific user
	// If there isn't already, a new one will be created with a buffer size of 10s
	if s.messageChan[in.Name] == nil {
		s.messageChan[in.Name] = make(chan *proto.Servermessage, 10)
	}

	response := &proto.Servermessage{
		Message: "The client " + in.Name + "has joined the chat",
		Timestamp: s.lamport,
	}

	for _, channel := range s.messageChan{
		channel <- response
	}
	
	for{
		select{
		case <- messageStream.Context().Done() :
			log.Printf("The client %d's stream has closed.\n", in.Name)
			return nil
		case message := <-s.messageChan[in.Name]:
			messageStream.SendMsg(message)
		}
	}
}

func (s* server) leave (ctx context.Context, in proto.Leave) (*proto.Leave, error){
	// This updates our servers lamport timestamp
	if in.Timestamp > s.lamport{
		s.lamport = in.Timestamp + 1
	} else {
		s.incrementLamport()
	}

	log.Printf("Client %s requested leaving at time %d", in.Name, in.Timestamp)
	
	announcement := &proto.Servermessage{
		Message: in.Name + " has left",
		Timestamp: in.Timestamp,
	}
	
	// Sends announcement that a user has left the chatroom to all connected clients, whereafter the users channel is deleted
	for _, channel := range s.messageChan{
		channel <- announcement
	}
	delete(s.messageChan, in.Name)
	
	// Defines a Leave object that shows that a user has left and a timestamp for when they left
	LeaveMessage:= &proto.Leave{
		Name: in.Name,
		Timestamp : in.Timestamp,
	}
	return LeaveMessage, nil		
}

func launch (s *server) {
	grpcServer := grpc.NewServer()

	//Creating a listener, to listen for messages
	//err will conatin an error if something goes wrong
	// tcp is the protocol
	listener, err := net.Listen("tcp", ":"+s.port)

	
	//If there is a faliure
	if err != nil {
		log.Fatalf("Could not create the server %v\n", err)
	}
	
	// If all works as it should
	log.Printf("started the server at port s%\n", s.port)

	proto.RegisterChittyChatServer(grpcServer, s)

	//listens to a specifik port for messages
	Errorserver := grpcServer.Serve(listener)
	
	//if it isn't nil then there is an error
	if Errorserver != nil{
		log.Fatalf("Could not serve listener\n")
	}
	
}

