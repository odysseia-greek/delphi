package lawgiver

import (
	"context"
	"fmt"
	"strings"

	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	kubernetes "github.com/odysseia-greek/agora/thales"
	aristophanes "github.com/odysseia-greek/attike/aristophanes/comedy"
	arv1 "github.com/odysseia-greek/attike/aristophanes/gen/go/v1"
	"github.com/odysseia-greek/delphi/solon/lawgiver/limen"
	"github.com/odysseia-greek/delphi/solon/logoi"
)

type Config struct {
	Vault            diogenes.Client
	Elastic          aristoteles.Client
	ElasticCert      []byte
	Kube             *kubernetes.KubeClient
	Namespaces       logoi.Namespaces
	AccessAnnotation string
	RoleAnnotation   string
	TLSEnabled       bool
	Streamer         arv1.TraceService_ChorusClient
	Cancel           context.CancelFunc
	Limen            *limen.Client
}

func CreateNewConfig(ctx context.Context) (*Config, error) {
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

	cfg, err := aristoteles.ElasticConfig(false)
	if err != nil {
		logging.Error(fmt.Sprintf("failed to create Elastic client operations will be interrupted, %s", err.Error()))
	}

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return nil, err
	}

	err = aristoteles.HealthCheck(elastic)
	if err != nil {
		return nil, err
	}

	var namespaces logoi.Namespaces
	namespaces.SolonNamespace = config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)

	otherNamespacesFromEnv := config.StringFromEnv("SOLON_MANAGED_NAMESPACES", "")
	namespaces.WatchedNamespaces = strings.Split(otherNamespacesFromEnv, ";")

	tracer, err := aristophanes.NewClientTracer(aristophanes.DefaultAddress)
	if err != nil {
		logging.Error(err.Error())
	}

	streamer, err := tracer.Chorus(ctx)
	if err != nil {
		logging.Error(err.Error())
	}

	ctx, cancel := context.WithCancel(ctx)

	return &Config{
		Vault:            vault,
		Elastic:          elastic,
		ElasticCert:      []byte(cert),
		Kube:             kube,
		Namespaces:       namespaces,
		AccessAnnotation: config.DefaultAccessAnnotation,
		RoleAnnotation:   config.DefaultRoleAnnotation,
		TLSEnabled:       tls,
		Streamer:         streamer,
		Cancel:           cancel,
		Limen:            limen.NewClient(kube, namespaces, elastic, vault),
	}, nil
}
