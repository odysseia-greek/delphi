package lawgiver

import (
	"context"
	"fmt"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"time"
)

func (s *SolonHandler) verifyRequestOriginIP(requestIP string) (*v1.Pod, error) {
	var strippedRequestIP string

	if strings.Contains(requestIP, ":") {
		strippedRequestIP = strings.Split(requestIP, ":")[0]
	} else {
		strippedRequestIP = requestIP
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pods, err := s.Kube.CoreV1().Pods(s.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods: %w", err)
	}

	// Check if the IP matches any pod's IP
	for _, pod := range pods.Items {
		if pod.Status.PodIP == strippedRequestIP {
			return &pod, nil // IP matches
		}
	}

	return nil, nil
}
