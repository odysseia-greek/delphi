package initiator

import (
	"fmt"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/models"
	"strings"
	"time"
)

const (
	TRACING string = "TRACE_CREATION"
)

func CreateNewConfig(duration time.Duration, timeOut time.Duration) (*PeriandrosHandler, error) {
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	service, err := config.CreateOdysseiaClient()
	if err != nil {
		return nil, err
	}

	tracing := config.BoolFromEnv(TRACING)
	solonRequest := initCreation(tracing)

	return &PeriandrosHandler{
		Namespace:            ns,
		HttpClients:          service,
		SolonCreationRequest: solonRequest,
		Duration:             duration,
		Timeout:              timeOut,
	}, nil
}

func initCreation(tracing bool) models.SolonCreationRequest {
	role := config.StringFromEnv(config.EnvRole, "")
	envAccess := config.SliceFromEnv(config.EnvIndex)
	podName := config.StringFromEnv(config.EnvPodName, config.DefaultPodname)
	secondaryAccess := config.StringFromEnv(config.EnvSecondaryIndex, "")
	if secondaryAccess != "" {
		envAccess = append(envAccess, secondaryAccess)
	}

	logging.Info(fmt.Sprintf("working on pod: %s", podName))
	splitPodName := strings.Split(podName, "-")

	var username string
	if !tracing {
		if len(splitPodName) > 1 {
			username = splitPodName[0] + splitPodName[len(splitPodName)-1]
		} else {
			username = splitPodName[0]
		}

	} else {
		username = config.DefaultTracingName
	}

	logging.Info(fmt.Sprintf("username from pod is: %s", username))

	creationRequest := models.SolonCreationRequest{
		Role:     role,
		Access:   envAccess,
		PodName:  podName,
		Username: username,
	}

	return creationRequest
}
