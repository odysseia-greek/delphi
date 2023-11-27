package config

import (
	"fmt"
	"github.com/odysseia-greek/aristoteles"
	elasticmodels "github.com/odysseia-greek/aristoteles/models"
	"github.com/odysseia-greek/diogenes"
	"github.com/odysseia-greek/plato/config"
	"github.com/odysseia-greek/plato/generator"
	"github.com/odysseia-greek/plato/logging"
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
	outOfClusterKube := false
	debugMode := false
	if env == "LOCAL" || env == "TEST" {
		healthCheck = false
		outOfClusterKube = true
		debugMode = true
	}

	testOverWrite := config.BoolFromEnv(testOverWrite)
	tls := config.BoolFromEnv(config.EnvTlSKey)

	vault, err := diogenes.CreateVaultClient(env, healthCheck, debugMode)
	if err != nil {
		return nil, err
	}

	kube, err := kubernetes.CreateKubeClient(outOfClusterKube)
	if err != nil {
		return nil, err
	}

	var cfg elasticmodels.Config
	var cert string

	cfg = aristoteles.ElasticConfig(env, testOverWrite, tls)
	cert = cfg.ElasticCERT

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

func (s *Config) CreateTracingUser(update bool) error {
	password, err := generator.RandomPassword(24)
	if err != nil {
		return err
	}

	secretName := fmt.Sprintf("%s-elastic", config.DefaultTracingName)
	secretData := map[string][]byte{
		"user":     []byte(config.DefaultTracingName),
		"password": []byte(password),
	}

	secret, _ := s.Kube.Configuration().GetSecret(s.Namespace, secretName)

	if secret == nil {
		logging.Info(fmt.Sprintf("secret %s does not exist", secretName))
		err = s.Kube.Configuration().CreateSecret(s.Namespace, secretName, secretData)
		if err != nil {
			return err
		}
	} else if update {
		logging.Info(fmt.Sprintf("secret %s already exists update flag set so updating", secret.Name))

		err = s.Kube.Configuration().UpdateSecret(s.Namespace, secretName, secretData)
		if err != nil {
			return err
		}
	} else {
		logging.Info(fmt.Sprintf("secret %s already exists so no action required", secretName))
		return nil
	}

	putUser := elasticmodels.CreateUserRequest{
		Password: password,
		Roles:    []string{config.CreatorElasticRole},
		FullName: config.DefaultTracingName,
		Email:    fmt.Sprintf("%s@odysseia-greek.com", config.DefaultTracingName),
		Metadata: &elasticmodels.Metadata{Version: 1},
	}

	userCreated, err := s.Elastic.Access().CreateUser(config.DefaultTracingName, putUser)
	if err != nil {
		return err
	}

	logging.Info(fmt.Sprintf("user %s created: %v in elasticSearch", config.DefaultTracingName, userCreated))

	return nil
}
