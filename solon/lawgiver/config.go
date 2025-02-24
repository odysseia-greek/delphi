package lawgiver

import (
	"context"
	"fmt"
	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	kubernetes "github.com/odysseia-greek/agora/thales"
	aristophanes "github.com/odysseia-greek/attike/aristophanes/comedy"
	"os"
)

func CreateNewConfig(ctx context.Context) (*SolonHandler, error) {
	vault, err := diogenes.CreateVaultClient(true)
	if err != nil {
		return nil, err
	}

	tls := config.BoolFromEnv(config.EnvTlSKey)

	kube, err := kubernetes.CreateKubeClient(false)
	if err != nil {
		return nil, err
	}

	var cert string

	cfg, err := aristoteles.ElasticConfig(tls)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to create Elastic client operations will be interupted, %s", err.Error()))
	}

	cert = cfg.ElasticCERT

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	err = aristoteles.HealthCheck(elastic)
	if err != nil {
		return nil, err
	}

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	tracer, err := aristophanes.NewClientTracer(aristophanes.DefaultAddress)
	if err != nil {
		logging.Error(err.Error())
	}

	healthy := tracer.WaitForHealthyState()
	if !healthy {
		logging.Debug("tracing service not ready - restarting seems the only option")
		os.Exit(1)
	}

	streamer, err := tracer.Chorus(ctx)
	if err != nil {
		logging.Error(err.Error())
	}

	ctx, cancel := context.WithCancel(ctx)

	return &SolonHandler{
		Vault:            vault,
		Elastic:          elastic,
		ElasticCert:      []byte(cert),
		Kube:             kube,
		Namespace:        ns,
		AccessAnnotation: config.DefaultAccessAnnotation,
		RoleAnnotation:   config.DefaultRoleAnnotation,
		TLSEnabled:       tls,
		Streamer:         streamer,
		Cancel:           cancel,
	}, nil
}
