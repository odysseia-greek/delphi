package config

import (
	"github.com/kpango/glg"
	"github.com/odysseia-greek/aristoteles"
	"github.com/odysseia-greek/diogenes"
	"github.com/odysseia-greek/plato/config"
	kubernetes "github.com/odysseia-greek/thales"
)

const testOverWrite string = "TEST_OVERWRITE"

type Config struct {
	Vault            diogenes.Client
	Elastic          aristoteles.Client
	ElasticCert      []byte
	Kube             kubernetes.KubeClient
	Namespace        string
	AccessAnnotation string
	RoleAnnotation   string
	TLSEnabled       bool
}

func CreateNewConfig(env string) (*Config, error) {
	healthCheck := true
	if env == "LOCAL" || env == "TEST" {
		healthCheck = false
	}
	testOverWrite := config.BoolFromEnv(testOverWrite)
	tls := config.BoolFromEnv(config.EnvTlSKey)

	vault, err := diogenes.CreateVaultClient(env, healthCheck)
	if err != nil {
		return nil, err
	}

	kube, err := kubernetes.CreateKubeClient(healthCheck)
	if err != nil {
		return nil, err
	}

	var cfg aristoteles.Config
	var cert string

	if healthCheck {
		vaultConfig, err := config.ConfigFromVault()
		if err != nil {
			glg.Error(err)
			return nil, err
		}

		service := aristoteles.ElasticService(tls)

		cfg = aristoteles.Config{
			Service:     service,
			Username:    vaultConfig.ElasticUsername,
			Password:    vaultConfig.ElasticPassword,
			ElasticCERT: vaultConfig.ElasticCERT,
		}
		cert = vaultConfig.ElasticCERT
	} else {
		cfg = aristoteles.ElasticConfig(env, testOverWrite, tls)
		cert = cfg.ElasticCERT
	}

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

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	return &Config{
		Vault:            vault,
		Elastic:          elastic,
		ElasticCert:      []byte(cert),
		Kube:             kube,
		Namespace:        ns,
		AccessAnnotation: config.DefaultAccessAnnotation,
		RoleAnnotation:   config.DefaultRoleAnnotation,
		TLSEnabled:       tls,
	}, nil
}
