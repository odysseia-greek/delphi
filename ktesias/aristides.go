package ktesias

import (
	"context"
	"fmt"
	"time"

	pb "github.com/odysseia-greek/delphi/aristides/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const aristidesAddress = "localhost:50051"

type aristidesClient struct {
	pb.AristidesClient
}

func newAristidesClient(address string) (*aristidesClient, error) {
	if address == "" {
		address = aristidesAddress
	}
	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("create Aristides client: %w", err)
	}
	return &aristidesClient{AristidesClient: pb.NewAristidesClient(conn)}, nil
}

func (c *aristidesClient) waitForHealthyState(ctx context.Context) bool {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		response, err := c.Health(ctx, &pb.HealthRequest{})
		if err == nil && response.Health {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-timeout.C:
			return false
		case <-ticker.C:
		}
	}
}
