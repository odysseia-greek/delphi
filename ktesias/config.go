package ktesias

import (
	"context"
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/randomizer"
	"github.com/odysseia-greek/agora/plato/service"
	kubernetes "github.com/odysseia-greek/agora/thales"
)

type OdysseiaFixture struct {
	ctx          context.Context
	client       service.OdysseiaClient
	randomizer   randomizer.Random
	Kube         *kubernetes.KubeClient
	CiliumClient *versioned.Clientset
	Vault        diogenes.Client
	Namespace    string
	PodName      string
}

func New() (*OdysseiaFixture, error) {
	svc, err := config.CreateOdysseiaClient()
	if err != nil {
		return nil, err
	}

	randomizerClient, err := randomizer.NewRandomizerClient()
	if err != nil {
		return nil, err
	}

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	podName := config.StringFromEnv(config.EnvPodName, config.DefaultPodname)

	kube, err := kubernetes.CreateKubeClient(false)
	if err != nil {
		return nil, err
	}

	ciliumClient, err := versioned.NewForConfig(kube.RestConfig())
	if err != nil {
		return nil, err
	}

	vault, err := diogenes.CreateVaultClient(true)
	if err != nil {
		return nil, err
	}

	return &OdysseiaFixture{
		client:       svc,
		ctx:          context.Background(),
		randomizer:   randomizerClient,
		Kube:         kube,
		Namespace:    ns,
		PodName:      podName,
		CiliumClient: ciliumClient,
		Vault:        vault,
	}, nil
}
