package architect

import (
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/agora/thales/odysseia"
	"sync"
	"time"
)

const (
	ORGANISATION string = "odysseia-greek"
)

type MappingUpdate struct {
	HostName     string
	ClientName   string
	KubeType     string
	SecretName   string
	Validity     int
	IsHostUpdate bool
}

func CreateNewConfig() (*PeriklesHandler, error) {
	kube, err := thales.CreateKubeClient(false)
	if err != nil {
		return nil, err
	}

	ciliumClient, err := versioned.NewForConfig(kube.RestConfig())
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

	tlsChecker := 1 * time.Hour
	updateMappingTimer := 30 * time.Second
	reconcileTimer := 30 * time.Second

	return &PeriklesHandler{
		Mutex:              sync.Mutex{},
		PendingUpdateTimer: updateMappingTimer,
		TLSCheckTimer:      tlsChecker,
		ReconcileTimer:     reconcileTimer,
		PendingUpdates:     map[string][]MappingUpdate{},
		Kube:               kube,
		CiliumClient:       ciliumClient,
		Mapping:            mapping,
		Cert:               cert,
		Namespace:          ns,
		CrdName:            crd,
		TLSFiles:           tlsFiles,
		L7Mode:             l7Mode,
	}, nil
}
