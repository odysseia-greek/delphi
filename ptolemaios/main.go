package main

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/ptolemaios/diplomat"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"

	pb "github.com/odysseia-greek/delphi/ptolemaios/proto"
)

const standardPort = ":50051"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = standardPort
	}

	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=PTOLEMAIOS
	logging.System(`
 ____  ______   ___   _        ___  ___ ___   ____  ____  ___   _____
|    \|      | /   \ | |      /  _]|   |   | /    ||    |/   \ / ___/
|  o  )      ||     || |     /  [_ | _   _ ||  o  | |  ||     (   \_ 
|   _/|_|  |_||  O  || |___ |    _]|  \_/  ||     | |  ||  O  |\__  |
|  |    |  |  |     ||     ||   [_ |   |   ||  _  | |  ||     |/  \ |
|  |    |  |  |     ||     ||     ||   |   ||  |  | |  ||     |\    |
|__|    |__|   \___/ |_____||_____||___|___||__|__||____|\___/  \___|
                                                                     
`)
	logging.System("\"Σωτήρ\"")
	logging.System("\"savior\"")
	logging.System("starting up.....")
	logging.System("starting up and getting env variables")

	ambassador, err := diplomat.CreateNewConfig()
	if err != nil {
		log.Fatalf("error creating TraceServiceClient: %v", err)
	}

	listener, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	var server *grpc.Server

	server = grpc.NewServer()

	pb.RegisterPtolemaiosServer(server, ambassador)

	logging.Info(fmt.Sprintf("Server listening on %s", port))
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
