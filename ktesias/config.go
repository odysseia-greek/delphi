package ktesias

import (
	"context"
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"github.com/google/uuid"
	"github.com/odysseia-greek/agora/aristoteles"
	"github.com/odysseia-greek/agora/aristoteles/models"
	"github.com/odysseia-greek/agora/diogenes"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/plato/logging"
	"github.com/odysseia-greek/agora/plato/randomizer"
	"github.com/odysseia-greek/agora/plato/service"
	kubernetes "github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/delphi/ptolemaios/diplomat"
	pb "github.com/odysseia-greek/delphi/ptolemaios/proto"
	"google.golang.org/grpc/metadata"
	"os"
	"time"
)

type OdysseiaFixture struct {
	ctx          context.Context
	client       service.OdysseiaClient
	randomizer   randomizer.Random
	Kube         *kubernetes.KubeClient
	CiliumClient *versioned.Clientset
	Vault        diogenes.Client
	Elastic      aristoteles.Client
	Namespace    string
	PodName      string
}

func New() (*OdysseiaFixture, error) {
	tls := config.BoolFromEnv(config.EnvTlSKey)

	var cfg models.Config
	ambassador := diplomat.NewClientAmbassador()

	healthy := ambassador.WaitForHealthyState()
	if !healthy {
		logging.Info("tracing service not ready - restarting seems the only option")
		os.Exit(1)
	}

	traceId := uuid.New().String()
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	md := metadata.New(map[string]string{service.HeaderKey: traceId})
	ctx = metadata.NewOutgoingContext(context.Background(), md)
	vaultConfig, err := ambassador.GetSecret(ctx, &pb.VaultRequest{})
	if err != nil {
		logging.Error(err.Error())
		return nil, err
	}

	elasticService := aristoteles.ElasticService(tls)

	cfg = models.Config{
		Service:     elasticService,
		Username:    vaultConfig.ElasticUsername,
		Password:    vaultConfig.ElasticPassword,
		ElasticCERT: vaultConfig.ElasticCERT,
	}

	elastic, err := aristoteles.NewClient(cfg)
	if err != nil {
		return nil, err
	}

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
		Elastic:      elastic,
		Kube:         kube,
		Namespace:    ns,
		PodName:      podName,
		CiliumClient: ciliumClient,
		Vault:        vault,
	}, nil
}
