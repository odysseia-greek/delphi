package architect

import (
	"sync"
	"time"

	"github.com/cilium/cilium/pkg/k8s/client/clientset/versioned"
	"github.com/odysseia-greek/delphi/perikles/pkg/service_mapping"
)

type PeriklesHandler struct {
	Mutex              sync.Mutex
	PendingUpdateTimer time.Duration
	TLSCheckTimer      time.Duration
	ReconcileTimer     time.Duration
	PendingUpdates     map[string][]MappingUpdate
	Kube               *KubeClient
	Mapping            service_mapping.ServiceMapping
	CiliumClient       versioned.Interface
	RuleSet            []CnpRuleSet
	Namespace          string
	CrdName            string
	TLSFiles           string
	ConfigMapName      string
	L7Mode             bool
	VaultNs            string
	ElasticNs          string
	WatchedNamespaces  []string
	DashboardAddr      string
	Events             *EventStore
}
