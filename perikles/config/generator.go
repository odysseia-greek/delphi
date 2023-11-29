package config

import (
	"github.com/odysseia-greek/agora/plato/certificates"
	"github.com/odysseia-greek/agora/plato/config"
	kubernetes "github.com/odysseia-greek/agora/thales"
)

type Config struct {
	Kube      kubernetes.KubeClient
	Cert      certificates.CertClient
	Namespace string
	CrdName   string
	TLSFiles  string
}

func CreateNewConfig(env string) (*Config, error) {
	healthCheck := true
	if env == "LOCAL" || env == "TEST" {
		healthCheck = false
	}

	kube, err := kubernetes.CreateKubeClient(healthCheck)
	if err != nil {
		return nil, err
	}

	org := []string{
		"odysseia",
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
	}, nil
}
