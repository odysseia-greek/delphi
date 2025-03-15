package main

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/delphi/aristides/diplomat"
	"google.golang.org/grpc"
	"log"
	"net"
	"os"

	pb "github.com/odysseia-greek/delphi/aristides/proto"
)

const standardPort = ":50051"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = standardPort
	}

	//https://patorjk.com/software/taag/#p=display&f=Crawford2&t=aristides
	logging.System(`
  ____  ____   ____ _____ ______  ____  ___      ___  _____
 /    ||    \ |    / ___/|      ||    ||   \    /  _]/ ___/
|  o  ||  D  ) |  (   \_ |      | |  | |    \  /  [_(   \_ 
|     ||    /  |  |\__  ||_|  |_| |  | |  D  ||    _]\__  |
|  _  ||    \  |  |/  \ |  |  |   |  | |     ||   [_ /  \ |
|  |  ||  .  \ |  |\    |  |  |   |  | |     ||     |\    |
|__|__||__|\_||____|\___|  |__|  |____||_____||_____| \___|

`)
	logging.System("\"τοῦ δὲ θαυμάσαντος καὶ πυθομένου, μή τι κακὸν αὐτὸν Ἀριστείδης πεποίηκεν, οὐδέν, εἶπεν, οὐδὲ γιγνώσκω τὸν ἄνθρωπον, ἀλλʼ ἐνοχλοῦμαι πανταχοῦ τὸν Δίκαιον ἀκούων\"")
	logging.System("\"He, astonished, asked the man what possible wrong Aristides had done him. None whatever, was the answer, I don’t even know the fellow, but I am tired of hearing him everywhere called The Just.\"")
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

	pb.RegisterAristidesServer(server, ambassador)

	logging.Info(fmt.Sprintf("Server listening on %s", port))
	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
