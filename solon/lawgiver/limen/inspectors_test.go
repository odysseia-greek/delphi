package limen

import (
	"testing"

	"github.com/odysseia-greek/delphi/solon/logoi"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHandlePodEvents_IgnoresNonPodObjects(t *testing.T) {
	access := &accessSpy{}
	vault := &vaultSpy{}
	client := &Client{
		namespaces: logoi.Namespaces{},
		elastic:    &elasticSpy{access: access},
		vault:      vault,
	}
	handlers := client.handlePodEvents()

	assert.NotPanics(t, func() {
		handlers.DeleteFunc("not-a-pod")
	})
	assert.Empty(t, access.deletedUserName)
}

func TestHandlePodEvents_TriggersCleanupInManagedNamespace(t *testing.T) {
	access := &accessSpy{}
	vault := &vaultSpy{}
	client := &Client{
		namespaces: logoi.Namespaces{
			SolonNamespace:    "solon-system",
			WatchedNamespaces: []string{"watched"},
		},
		elastic: &elasticSpy{access: access},
		vault:   vault,
	}
	handlers := client.handlePodEvents()

	handlers.DeleteFunc(&v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "worker-pod-42",
		Namespace: "watched",
	}})

	assert.Equal(t, []string{"worker42"}, access.deletedUserName)
	assert.Equal(t, []string{"worker-pod-42"}, vault.deleteSecretCalls)
	assert.Equal(t, []string{"worker-pod-42"}, vault.removeSecretCalls)
	assert.Equal(t, []string{"policy-worker-pod-42"}, vault.deletePolicyCalls)
}

func TestHandlePodEvents_SkipsCleanupOutsideManagedNamespaces(t *testing.T) {
	access := &accessSpy{}
	vault := &vaultSpy{}
	client := &Client{
		namespaces: logoi.Namespaces{
			SolonNamespace:    "solon-system",
			WatchedNamespaces: []string{"watched"},
		},
		elastic: &elasticSpy{access: access},
		vault:   vault,
	}
	handlers := client.handlePodEvents()

	handlers.DeleteFunc(&v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "worker-pod-42",
		Namespace: "random",
	}})

	assert.Empty(t, access.deletedUserName)
	assert.Empty(t, vault.deleteSecretCalls)
	assert.Empty(t, vault.removeSecretCalls)
	assert.Empty(t, vault.deletePolicyCalls)
}
