package diplomat

import (
	"context"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/service"
	pb "github.com/odysseia-greek/delphi/ptolemaios/proto"
	"google.golang.org/grpc"
	"time"
)

type AmbassadorService interface {
	GetSecret(ctx context.Context, in *pb.VaultRequest) (*pb.ElasticConfigVault, error)
	GetNamedSecret(ctx context.Context, in *pb.VaultRequestNamed) (*pb.ElasticConfigVault, error)
	Health(ctx context.Context, in *pb.HealthRequest) (*pb.HealthResponse, error)
	ShutDown(ctx context.Context, in *pb.ShutDownRequest) (*pb.ShutDownResponse, error)
	WaitForHealthyState() bool
}

const (
	DEFAULTADDRESS string = "localhost:50051"
)

type AmbassadorServiceImpl struct {
	HttpClients service.OdysseiaClient
	Vault       diogenes.Client
	PodName     string
	Namespace   string
	FullPodName string
	pb.UnimplementedPtolemaiosServer
}

type AmbassadorServiceClient struct {
	Impl AmbassadorService
}

type ClientAmbassador struct {
	ambassador pb.PtolemaiosClient
}

func NewClientAmbassador() *ClientAmbassador {
	// Initialize the gRPC client for the tracing service
	conn, _ := grpc.Dial(DEFAULTADDRESS, grpc.WithInsecure())
	client := pb.NewPtolemaiosClient(conn)
	return &ClientAmbassador{ambassador: client}
}

func (c *ClientAmbassador) WaitForHealthyState() bool {
	timeout := 30 * time.Second
	checkInterval := 1 * time.Second
	endTime := time.Now().Add(timeout)

	for time.Now().Before(endTime) {
		response, err := c.Health(context.Background(), &pb.HealthRequest{})
		if err == nil && response.Health {
			return true
		}

		time.Sleep(checkInterval)
	}

	return false
}

func (c *ClientAmbassador) Health(ctx context.Context, request *pb.HealthRequest) (*pb.HealthResponse, error) {
	return c.ambassador.Health(ctx, request)
}

func (c *ClientAmbassador) ShutDown(ctx context.Context, request *pb.ShutDownRequest) (*pb.ShutDownResponse, error) {
	return c.ambassador.ShutDown(ctx, request)
}

func (c *ClientAmbassador) GetNamedSecret(ctx context.Context, request *pb.VaultRequestNamed) (*pb.ElasticConfigVault, error) {
	return c.ambassador.GetNamedSecret(ctx, request)
}

func (c *ClientAmbassador) GetSecret(ctx context.Context, request *pb.VaultRequest) (*pb.ElasticConfigVault, error) {
	return c.ambassador.GetSecret(ctx, request)
}
