package config

import (
	"github.com/odysseia-greek/agora/plato/certificates"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/agora/thales/odysseia"
)

type Config struct {
	Kube      *thales.KubeClient
	Mapping   odysseia.ServiceMapping
	Cert      certificates.CertClient
	Namespace string
	CrdName   string
	TLSFiles  string
}

func CreateNewConfig(env string) (*Config, error) {
	outOfClusterKube := false
	if env == "DEVELOPMENT" {
		outOfClusterKube = true
	}

	kube, err := thales.CreateKubeClient(outOfClusterKube)
	if err != nil {
		return nil, err
	}

	org := []string{
		"odysseia",
	}

	mapping, err := odysseia.NewServiceMappingImpl(kube.RestConfig())
	if err != nil {
		return nil, err
	}

	cert, err := config.CreateCertClient(org)
	if err != nil {
		return nil, err
	}

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	crd := config.StringFromEnv(config.EnvCrdName, config.DefaultCrdName)
	tlsFiles := config.StringFromEnv(config.EnvTLSFiles, config.DefaultTLSFileLocation)

	return &Config{
		Kube:      kube,
		Cert:      cert,
		Namespace: ns,
		CrdName:   crd,
		TLSFiles:  tlsFiles,
		Mapping:   mapping,
	}, nil
}
