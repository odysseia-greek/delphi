package main

import (
	"fmt"
	"github.com/odysseia-greek/delphi/ptolemaios/app"
	"github.com/odysseia-greek/delphi/ptolemaios/config"
	"github.com/odysseia-greek/plato/logging"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"

	pb "github.com/odysseia-greek/plato/proto"
)

const standardPort = ":50051"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = standardPort
	}

	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=PTOLEMAIOS
	logging.System("\n ____  ______   ___   _        ___  ___ ___   ____  ____  ___   _____\n|    \\|      | /   \\ | |      /  _]|   |   | /    ||    |/   \\ / ___/\n|  o  )      ||     || |     /  [_ | _   _ ||  o  | |  ||     (   \\_ \n|   _/|_|  |_||  O  || |___ |    _]|  \\_/  ||     | |  ||  O  |\\__  |\n|  |    |  |  |     ||     ||   [_ |   |   ||  _  | |  ||     |/  \\ |\n|  |    |  |  |     ||     ||     ||   |   ||  |  | |  ||     |\\    |\n|__|    |__|   \\___/ |_____||_____||___|___||__|__||____|\\___/  \\___|\n                                                                     \n")
	logging.System("\"Σωτήρ\"")
	logging.System("\"savior\"")
	logging.System("starting up.....")
	logging.System("starting up and getting env variables")

	env := os.Getenv("ENV")

	cfg, err := config.CreateNewConfig(env)
	if err != nil {
		log.Fatal("could not parse config")
	}

	handler := app.CreateHandler(cfg)

	lis, err := net.Listen("tcp", fmt.Sprintf("%s", port))
	if err != nil {
		log.Fatal(fmt.Sprintf("failed to listen: %v", err))
	}

	logging.Info(fmt.Sprintf("%s : %s", "setting up rpc service on", port))

	s := grpc.NewServer()
	pb.RegisterPtolemaiosServer(s, handler)
	logging.Info(fmt.Sprintf("server listening at %v", lis.Addr()))
	if err := s.Serve(lis); err != nil {
		log.Fatal(fmt.Sprintf("failed to serve: %v", err))
	}
}
