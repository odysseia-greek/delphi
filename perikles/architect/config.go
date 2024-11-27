package architect

import (
	"github.com/odysseia-greek/agora/plato/certificates"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/agora/thales/odysseia"
)

const (
	ORGANISATION string = "odysseia-greek"
)

type Config struct {
	Kube      *thales.KubeClient
	Mapping   odysseia.ServiceMapping
	Cert      certificates.CertClient
	Namespace string
	CrdName   string
	TLSFiles  string
	L7Mode    bool
}

func CreateNewConfig(env string) (*Config, error) {
	kube, err := thales.CreateKubeClient(false)
	if err != nil {
		return nil, err
	}

	org := []string{
		ORGANISATION,
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
	l7Mode := config.BoolFromEnv("L7_MODE")

	return &Config{
		Kube:      kube,
		Cert:      cert,
		Namespace: ns,
		CrdName:   crd,
		TLSFiles:  tlsFiles,
		Mapping:   mapping,
		L7Mode:    l7Mode,
	}, nil
}
