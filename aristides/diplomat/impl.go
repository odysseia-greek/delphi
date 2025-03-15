package diplomat

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/service"
	pb "github.com/odysseia-greek/delphi/aristides/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
	pb.UnimplementedAristidesServer
}

type AmbassadorServiceClient struct {
	Impl AmbassadorService
}

type ClientAmbassador struct {
	ambassador pb.AristidesClient
}

func NewClientAmbassador(address string) (*ClientAmbassador, error) {
	if address == "" {
		address = DEFAULTADDRESS
	}

	conn, err := grpc.NewClient(DEFAULTADDRESS, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc server: %w", err)
	}
	client := pb.NewAristidesClient(conn)
	return &ClientAmbassador{ambassador: client}, nil
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
