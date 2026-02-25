package architect

import (
	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"github.com/odysseia-greek/agora/plato/config"
	"github.com/odysseia-greek/agora/thales"
	"github.com/odysseia-greek/delphi/perikles/pkg/service_mapping"

	"os"
	"strings"
	"sync"
	"time"
)

type MappingUpdate struct {
	HostName     string
	ClientName   string
	KubeType     string
	SecretName   string
	Namespace    string
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

	mapping, err := service_mapping.NewServiceMappingImpl(kube.RestConfig())
	if err != nil {
		return nil, err
	}

	ns := config.StringFromEnv(config.EnvNamespace, config.DefaultNamespace)
	crd := config.StringFromEnv(config.EnvCrdName, config.DefaultCrdName)
	tlsFiles := config.StringFromEnv(config.EnvTLSFiles, config.DefaultTLSFileLocation)
	l7Mode := config.BoolFromEnv("L7_MODE")

	configMapName := os.Getenv("CONFIGMAP_NAME")
	if configMapName == "" {

	}

	tlsChecker := 1 * time.Hour
	updateMappingTimer := 30 * time.Second

	vaultNs := config.StringFromEnv("VAULT_NAMESPACE", "delphi")
	elasticNs := config.StringFromEnv("ELASTIC_NAMESPACE", "agora")

	otherNamespacesFromEnv := config.StringFromEnv("WATCHED_NAMESPACES", "")
	watchedNamespaces := strings.Split(otherNamespacesFromEnv, ";")

	return &PeriklesHandler{
		Mutex:              sync.Mutex{},
		PendingUpdateTimer: updateMappingTimer,
		TLSCheckTimer:      tlsChecker,
		PendingUpdates:     map[string][]MappingUpdate{},
		Kube:               kube,
		CiliumClient:       ciliumClient,
		Mapping:            mapping,
		Namespace:          ns,
		CrdName:            crd,
		TLSFiles:           tlsFiles,
		L7Mode:             l7Mode,
		ConfigMapName:      configMapName,
		RuleSet:            make([]CnpRuleSet, 0),
		VaultNs:            vaultNs,
		ElasticNs:          elasticNs,
		WatchedNamespaces:  watchedNamespaces,
	}, nil
}
