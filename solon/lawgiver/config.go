package lawgiver

import (
	"context"
	"fmt"
	"strings"

	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	aristophanes "github.com/odysseia-greek/attike/aristophanes/comedy"
	arv1 "github.com/odysseia-greek/attike/aristophanes/gen/go/v1"
	limencleanup "github.com/odysseia-greek/delphi/solon/limen/cleanup"
	limenelastic "github.com/odysseia-greek/delphi/solon/limen/elastic"
	limenkubernetes "github.com/odysseia-greek/delphi/solon/limen/kubernetes"
	limenvault "github.com/odysseia-greek/delphi/solon/limen/vault"
	"github.com/odysseia-greek/delphi/solon/logoi"
)

type Config struct {
	ElasticCert      []byte
	AccessAnnotation string
	RoleAnnotation   string
	TLSEnabled       bool
	Streamer         arv1.TraceService_ChorusClient
	Cancel           context.CancelFunc
	ElasticLimen     *limenelastic.Client
	VaultLimen       *limenvault.Client
	KubernetesLimen  *limenkubernetes.Client
	CleanupLimen     *limencleanup.Client
}

func CreateNewConfig(ctx context.Context) (*Config, error) {
	vault, err := diogenes.CreateVaultClient(ctx, true)
	if err != nil {
		return nil, err
	}

	tls := config.BoolFromEnv(config.EnvTlSKey)

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

	elasticLimen := limenelastic.NewClient(elastic)
	vaultLimen := limenvault.NewClient(vault)
	cleanupLimen := limencleanup.NewClient(elasticLimen, vaultLimen)
	kubernetesLimen, err := limenkubernetes.NewClient(namespaces, cleanupLimen)
	if err != nil {
		return nil, err
	}

	return &Config{
		ElasticCert:      []byte(cert),
		AccessAnnotation: config.DefaultAccessAnnotation,
		RoleAnnotation:   config.DefaultRoleAnnotation,
		TLSEnabled:       tls,
		Streamer:         streamer,
		Cancel:           cancel,
		ElasticLimen:     elasticLimen,
		VaultLimen:       vaultLimen,
		KubernetesLimen:  kubernetesLimen,
		CleanupLimen:     cleanupLimen,
	}, nil
}
