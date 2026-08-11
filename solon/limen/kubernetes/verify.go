package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/odysseia-greek/agora/plato/logging"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) VerifyRequestOriginIP(requestIP string) (*v1.Pod, error) {
	if c == nil || c.Interface == nil {
		return nil, fmt.Errorf("limen client is not initialized with a kube client")
	}
	var strippedRequestIP string

	if strings.Contains(requestIP, ":") {
		strippedRequestIP = strings.Split(requestIP, ":")[0]
	} else {
		strippedRequestIP = requestIP
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// First check in Solon's own namespace
	pods, err := c.CoreV1().Pods(c.namespaces.SolonNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in Solon namespace: %w", err)
	}

	// Check if the IP matches any pod's IP in Solon's namespace
	for _, pod := range pods.Items {
		if pod.Status.PodIP == strippedRequestIP {
			return &pod, nil
		}
	}

	// If not found in Solon's namespace, check all watched namespaces
	for _, namespace := range c.namespaces.WatchedNamespaces {
		pods, err := c.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
		if err != nil {
			// Log the error but continue with other namespaces
			logging.Debug(fmt.Sprintf("failed to list pods in namespace %s: %v", namespace, err))
			continue
		}

		// Check if the IP matches any pod's IP in this watched namespace
		for _, pod := range pods.Items {
			if pod.Status.PodIP == strippedRequestIP {
				return &pod, nil
			}
		}
	}

	// No matching pod found in any of the namespaces
	return nil, nil
}
