package kubernetes

import (
	"testing"

	"github.com/odysseia-greek/delphi/solon/logoi"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type cleanerSpy struct {
	deletedUsers []string
	deletedPods  []string
}

func (c *cleanerSpy) DeleteOrphan(username, podName string) error {
	c.deletedUsers = append(c.deletedUsers, username)
	c.deletedPods = append(c.deletedPods, podName)
	return nil
}

func TestHandlePodEvents_IgnoresNonPodObjects(t *testing.T) {
	cleaner := &cleanerSpy{}
	client := &Client{
		namespaces: logoi.Namespaces{},
		cleaner:    cleaner,
	}
	handlers := client.handlePodWatchEvents()

	assert.NotPanics(t, func() {
		handlers.DeleteFunc("not-a-pod")
	})
	assert.Empty(t, cleaner.deletedPods)
}

func TestHandlePodEvents_TriggersCleanupInManagedNamespace(t *testing.T) {
	cleaner := &cleanerSpy{}
	client := &Client{
		namespaces: logoi.Namespaces{
			SolonNamespace:    "solon-system",
			WatchedNamespaces: []string{"watched"},
		},
		cleaner: cleaner,
	}
	handlers := client.handlePodWatchEvents()

	handlers.DeleteFunc(&v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "worker-pod-42",
		Namespace: "watched",
	}})

	assert.Equal(t, []string{"worker42"}, cleaner.deletedUsers)
	assert.Equal(t, []string{"worker-pod-42"}, cleaner.deletedPods)
}

func TestHandlePodEvents_SkipsCleanupOutsideManagedNamespaces(t *testing.T) {
	cleaner := &cleanerSpy{}
	client := &Client{
		namespaces: logoi.Namespaces{
			SolonNamespace:    "solon-system",
			WatchedNamespaces: []string{"watched"},
		},
		cleaner: cleaner,
	}
	handlers := client.handlePodWatchEvents()

	handlers.DeleteFunc(&v1.Pod{ObjectMeta: metav1.ObjectMeta{
		Name:      "worker-pod-42",
		Namespace: "random",
	}})

	assert.Empty(t, cleaner.deletedPods)
}
