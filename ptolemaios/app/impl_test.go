package app

import (
	pb "github.com/odysseia-greek/delphi/ptolemaios/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"testing"
)

func TestHealthOnImpl(t *testing.T) {
	mockService := new(MockTraceService)

	t.Run("HealthCheck", func(t *testing.T) {
		expectedResponse := &pb.HealthResponse{
			Health: true,
		}

		request := pb.HealthRequest{}

		client := &AmbassadorServiceClient{
			Impl: mockService,
		}
		mockService.On("Health", mock.Anything, &request).Return(expectedResponse, nil)
		response := client.Impl.WaitForHealthyState()

		mockService.AssertExpectations(t)
		assert.True(t, response)
	})
}
