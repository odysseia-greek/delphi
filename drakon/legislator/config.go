package legislator

import (
	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/plato/config"
)

func CreateNewConfig(env string) (*DrakonHandler, error) {
	healthCheck := true
	if env == "DEVELOPMENT" {
		healthCheck = false
	}

	tls := config.BoolFromEnv(config.EnvTlSKey)

	var cfg models.Config

	cfg = aristoteles.ElasticConfig(env, false, tls)

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	if healthCheck {
		err := aristoteles.HealthCheck(elastic)
		if err != nil {
			return nil, err
		}
	}

	podName := config.ParsedPodNameFromEnv()
	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	roles := config.SliceFromEnv(config.EnvRoles)
	indexes := config.SliceFromEnv(config.EnvIndexes)

	return &DrakonHandler{
		Namespace: ns,
		PodName:   podName,
		Elastic:   elastic,
		Roles:     roles,
		Indexes:   indexes,
	}, nil
}
