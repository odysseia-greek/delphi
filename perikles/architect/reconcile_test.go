package architect

import (
	"context"
	"testing"
	"time"

	ciliumv2 "github.com/cilium/cilium/pkg/k8s/apis/cilium.io/v2"
	ciliumfake "github.com/cilium/cilium/pkg/k8s/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPolicyBelongsToWorkload(t *testing.T) {
	assert.True(t, policyBelongsToWorkload("restrict-elasticsearch-access-seeder-123", "seeder-123"))
	assert.True(t, policyBelongsToWorkload("allow-seeder-123-access-api", "seeder-123"))
	assert.True(t, policyBelongsToWorkload("allow-seeder-123-access-vault", "seeder-123"))
	assert.False(t, policyBelongsToWorkload("restrict-elasticsearch-access-seeder-1234", "seeder-123"))
	assert.False(t, policyBelongsToWorkload("allow-other-access-seeder-123", "seeder-123"))
}

func TestIsStaleJobNetworkPolicy(t *testing.T) {
	now := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	old := metav1.NewTime(now.Add(-25 * time.Hour))
	recent := metav1.NewTime(now.Add(-23 * time.Hour))

	tests := []struct {
		name   string
		policy ciliumv2.CiliumNetworkPolicy
		stale  bool
	}{
		{
			name: "annotated job",
			policy: ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
				Name:              "restrict-elasticsearch-access-seeder",
				CreationTimestamp: old,
				Annotations:       map[string]string{AnnotationSourceKind: "Job"},
			}},
			stale: true,
		},
		{
			name: "annotated deployment",
			policy: ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
				Name:              "restrict-elasticsearch-access-api",
				CreationTimestamp: old,
				Annotations:       map[string]string{AnnotationSourceKind: "Deployment"},
			}},
		},
		{
			name: "legacy cronjob",
			policy: ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
				Name:              "restrict-elasticsearch-access-seeder-29696010",
				CreationTimestamp: old,
			}},
			stale: true,
		},
		{
			name: "legacy deployment",
			policy: ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
				Name:              "restrict-elasticsearch-access-aristarchos",
				CreationTimestamp: old,
			}},
		},
		{
			name: "recent job",
			policy: ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
				Name:              "restrict-elasticsearch-access-seeder-29696010",
				CreationTimestamp: recent,
				Annotations:       map[string]string{AnnotationSourceKind: "Job"},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.stale, isStaleJobNetworkPolicy(&test.policy, now, 24*time.Hour))
		})
	}
}

func TestCleanUpNetworkPoliciesDeletesAllWorkloadPolicyTypes(t *testing.T) {
	const (
		workload  = "seeder-29696010"
		elasticNs = "agora"
		sourceNs  = "apologia"
		vaultNs   = "delphi"
	)

	client := ciliumfake.NewSimpleClientset(
		&ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
			Name:      "restrict-elasticsearch-access-" + workload,
			Namespace: elasticNs,
		}},
		&ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-" + workload + "-access-vault",
			Namespace: vaultNs,
		}},
		&ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
			Name:      "allow-" + workload + "-access-api",
			Namespace: sourceNs,
		}},
		&ciliumv2.CiliumNetworkPolicy{ObjectMeta: metav1.ObjectMeta{
			Name:      "restrict-elasticsearch-access-other",
			Namespace: elasticNs,
		}},
	)

	handler := &PeriklesHandler{
		CiliumClient:      client,
		Namespace:         sourceNs,
		ElasticNs:         elasticNs,
		VaultNs:           vaultNs,
		WatchedNamespaces: []string{sourceNs},
	}

	require.NoError(t, handler.cleanUpNetWorkPolicies(workload, sourceNs))

	_, err := client.CiliumV2().CiliumNetworkPolicies(elasticNs).Get(
		context.Background(),
		"restrict-elasticsearch-access-"+workload,
		metav1.GetOptions{},
	)
	assert.Error(t, err)

	_, err = client.CiliumV2().CiliumNetworkPolicies(vaultNs).Get(
		context.Background(),
		"allow-"+workload+"-access-vault",
		metav1.GetOptions{},
	)
	assert.Error(t, err)

	_, err = client.CiliumV2().CiliumNetworkPolicies(sourceNs).Get(
		context.Background(),
		"allow-"+workload+"-access-api",
		metav1.GetOptions{},
	)
	assert.Error(t, err)

	_, err = client.CiliumV2().CiliumNetworkPolicies(elasticNs).Get(
		context.Background(),
		"restrict-elasticsearch-access-other",
		metav1.GetOptions{},
	)
	assert.NoError(t, err)
}
