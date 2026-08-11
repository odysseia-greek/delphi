package ktesias

import (
	"context"

	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/randomizer"
	"github.com/odysseia-greek/agora/plato/service"
)

type OdysseiaFixture struct {
	ctx              context.Context
	client           service.OdysseiaClient
	randomizer       randomizer.Random
	Kube             *KubeClient
	CiliumClient     *versioned.Clientset
	Vault            diogenes.Client
	Namespace        string
	ElasticNamespace string
	PodName          string
}

func New() (*OdysseiaFixture, error) {
	ctx := context.Background()
	svc, err := config.CreateOdysseiaClient()
	if err != nil {
		return nil, err
	}

	randomizerClient, err := randomizer.NewRandomizerClient()
	if err != nil {
		return nil, err
	}

	ns := config.StringFromEnv(config.EnvNamespace, "delphi")
	podName := config.StringFromEnv(config.EnvPodName, config.DefaultPodname)

	kube, err := newKubeClient()
	if err != nil {
		return nil, err
	}

	ciliumClient, err := versioned.NewForConfig(kube.RestConfig())
	if err != nil {
		return nil, err
	}

	vault, err := diogenes.CreateVaultClient(ctx, true)
	if err != nil {
		return nil, err
	}

	elasticNs := "agora"

	return &OdysseiaFixture{
		client:           svc,
		ctx:              ctx,
		randomizer:       randomizerClient,
		Kube:             kube,
		Namespace:        ns,
		ElasticNamespace: elasticNs,
		PodName:          podName,
		CiliumClient:     ciliumClient,
		Vault:            vault,
	}, nil
}
