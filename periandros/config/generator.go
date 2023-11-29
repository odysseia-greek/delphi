package config

import (
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/models"
	"log"
	"strings"
)

const (
	TRACING string = "TRACE_CREATION"
)

func CreateNewConfig(env string) (*Config, error) {
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	service, err := config.CreateOdysseiaClient()
	if err != nil {
		return nil, err
	}

	tracing := config.BoolFromEnv(TRACING)
	solonRequest := initCreation(tracing)

	return &Config{
		Namespace:            ns,
		HttpClients:          service,
		SolonCreationRequest: solonRequest,
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
	splitPodName := strings.Split(podName, "-")

	var username string
	if !tracing {
		username = splitPodName[0] + splitPodName[2]
	} else {
		username = config.DefaultTracingName
	}

	log.Printf("username from pod is: %s", username)

	creationRequest := models.SolonCreationRequest{
		Role:     role,
		Access:   envAccess,
		PodName:  podName,
		Username: username,
	}

	return creationRequest
}
